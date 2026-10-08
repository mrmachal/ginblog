// Package session 基于 Redis 的登录态存储。
//
// 职责边界：这里只管 token ↔ userID 的存取与过期，不碰 cookie——
// cookie 是 HTTP 关注点，由 handler 层用 c.SetCookie / c.Cookie 处理。
// 因此 Config 只需要 key 前缀和 TTL 两个字段。
package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound 表示 token 不存在或已过期。
// 调用方用 errors.Is 判断，不要字符串比较错误信息。
var ErrNotFound = errors.New("session not found")

// tokenBytes 随机 token 的字节数。32 字节 = 256 位熵，
// base64url 裸编码后 43 个字符，无需分隔符即可放进 cookie。
const tokenBytes = 32

type Config struct {
	// Prefix Redis key 前缀，如 "ginblog:session:"，避免与其他业务撞 key
	Prefix string
	// TTL 单个 session 的存活时间；来自 config.Auth.ExpireHours，转换在接线处完成
	TTL time.Duration
}

type Store struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// New 创建 session 存储。rdb 的生命周期由调用方（main）管理，Store 不负责 Close。
func New(cfg Config, rdb *redis.Client) (*Store, error) {
	if rdb == nil {
		return nil, errors.New("session: redis client is nil")
	}
	// TTL <= 0 会让 SETEX 直接返回 invalid expire time，启动期就拦掉
	if cfg.TTL <= 0 {
		return nil, errors.New("session: ttl must be greater than 0")
	}

	return &Store{
		client: rdb,
		prefix: cfg.Prefix,
		ttl:    cfg.TTL,
	}, nil
}

// Create 生成随机 token 并写入 Redis，返回 token（原样下发给客户端，不要记录日志）。
// 值存 userID 字符串，key = prefix + token。
func (s *Store) Create(c context.Context, userID uint) (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// RawURLEncoding 无 padding，token 里不会出现 "=" "+" "/"，可安全放进 cookie
	token := base64.RawURLEncoding.EncodeToString(buf)

	if err := s.client.Set(c, s.key(token), strconv.FormatUint(uint64(userID), 10), s.ttl).Err(); err != nil {
		return "", err
	}

	return token, nil
}

// Get 查 token 对应的 userID；不存在或已过期统一返回 ErrNotFound。
func (s *Store) Get(c context.Context, token string) (uint, error) {
	if token == "" {
		return 0, ErrNotFound
	}

	val, err := s.client.Get(c, s.key(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrNotFound
		}
		return 0, err
	}

	userID, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		// 值被外部改坏：按失效处理，顺手删掉脏 key
		_ = s.client.Del(c, s.key(token)).Err()
		return 0, ErrNotFound
	}

	return uint(userID), nil
}

// Delete 删除 session（登出）。key 不存在视为已登出，不报错。
func (s *Store) Delete(c context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.client.Del(c, s.key(token)).Err()
}

// Touch 滑动续期：每次活跃请求调用一次，把剩余 TTL 重置为满值。
// key 已不存在时返回 ErrNotFound，由调用方决定是否要求重新登录。
func (s *Store) Touch(c context.Context, token string) error {
	if token == "" {
		return ErrNotFound
	}

	ok, err := s.client.Expire(c, s.key(token), s.ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}

	return nil
}

func (s *Store) key(token string) string {
	return s.prefix + token
}
