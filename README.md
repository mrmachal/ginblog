# ginblog

基于 **Gin + GORM + Viper + slog + Redis** 的博客后端服务（进行中）。

当前已完成：配置加载、结构化日志（控制台 + 文件 + 滚动切割）、GORM 日志桥接、访问日志中间件、panic 恢复、数据库接入（GORM + SQLite，自动迁移）、文章 CRUD 接口（列表/详情/创建/更新/删除）、统一响应格式、HTTP 服务启动与优雅退出、**基于 Redis Session + Cookie 的登录态**（注册/登录/登出、个人信息读取与修改、修改密码）、**管理员用户管理**（列表/详情/创建/修改/删除，双层认证中间件保护）。

## 技术栈

| 组件 | 用途 |
|---|---|
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP 框架 |
| [spf13/viper](https://github.com/spf13/viper) | YAML 配置加载 |
| `log/slog`（标准库） | 结构化日志（JSON 格式） |
| [lumberjack.v2](https://github.com/natefinch/lumberjack) | 日志文件滚动切割 |
| [gorm.io/gorm](https://gorm.io) | ORM（模型定义、自动迁移） |
| [glebarez/sqlite](https://github.com/glebarez/sqlite) | SQLite 驱动（纯 Go，无需 CGO） |
| [redis/go-redis/v9](https://github.com/redis/go-redis) | Redis 客户端（登录态 Session 存储） |
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | bcrypt 密码哈希 |

> Go 版本要求：`go 1.25.0+`
> **运行依赖 Redis**：启动时会 Ping Redis，连不上直接退出（见「快速开始」）。

## 项目结构

```text
ginblog/
├── main.go                    # 入口：配置 → 日志 → 数据库 → 迁移 → Redis → Session → 路由 → 启动 → 优雅退出
├── api/
│   ├── api.go                 # 对外统一入口：创建 /api 分组，注入 session 与 auth 配置，注册各业务模块
│   ├── article/               # 文章模块（handler → service → repository 三层）
│   │   ├── router.go          # 路由注册
│   │   ├── handler.go         # 参数绑定、响应映射
│   │   ├── service.go         # 业务逻辑（校验、组装）
│   │   ├── repository.go      # 数据访问（GORM）
│   │   ├── dto.go             # 请求/响应结构与校验标签
│   │   └── article.go         # 模块装配：repo → service → handler → router
│   └── user/                  # 用户模块（同上三层 + error.go 业务错误）
│       ├── router.go          # 公开 / 登录态 / 管理员 三组路由，中间件在此挂载
│       ├── handler.go         # 参数绑定、cookie 读写、错误码映射
│       ├── service.go         # 注册/登录/改密/资料/管理端业务逻辑
│       ├── repository.go      # 数据访问（GORM）
│       ├── dto.go             # 请求/响应结构与校验标签
│       └── error.go           # 业务错误哨兵（ErrUserNotFound 等）
├── middleware/
│   ├── logger.go              # 访问日志中间件（slog，按状态码分级）
│   ├── recovery.go            # panic 恢复中间件（记录堆栈 + 返回 500）
│   └── auth.go                # RequireAuth / RequireAdmin / OptionalAuth 认证鉴权
├── config/
│   ├── config.go              # 配置结构体定义、默认值、启动期校验（viper）
│   ├── ginblog.yaml           # 实际使用的配置文件（已 gitignore）
│   └── ginblog.yaml.example   # 配置示例（含注释）
├── model/
│   ├── article.go             # 文章模型（gorm.Model + 标题/摘要/内容/作者）
│   └── user.go                # 用户模型（用户名/昵称/邮箱/密码哈希/角色）
├── pkg/
│   ├── database/
│   │   └── database.go        # GORM + SQLite 初始化（PRAGMA、连接池）与自动迁移
│   ├── logger/
│   │   └── logger.go          # slog 初始化、GORM 日志桥接、gin 日志转发
│   ├── redis/
│   │   └── redis.go           # go-redis 初始化（Ping 校验、超时与连接池）
│   ├── session/
│   │   └── session.go         # 基于 Redis 的 Session 存储（Create/Get/Delete/Touch）
│   └── response/
│       └── response.go        # 统一响应结构与业务错误码
├── build/                     # 编译产物（已 gitignore）
├── data/                      # SQLite 数据库文件（已 gitignore，自动创建）
├── logs/                      # 运行时日志目录（已 gitignore，自动创建）
├── go.mod
└── go.sum
```

## 快速开始

### 0. 准备 Redis

登录态存在 Redis 里，**启动前必须有可达的 Redis**，否则进程直接退出：

```bash
# 本机最简方式
docker run -d --name ginblog-redis -p 6379:6379 redis:7-alpine
```

地址、密码等见配置 `redis.*`，连不上会报 `failed to init redis: failed to ping redis: ...`。

### 1. 准备配置

配置文件查找顺序：`./ginblog.yaml` → `./config/ginblog.yaml`。

```bash
# 首次使用：复制示例配置
cp config/ginblog.yaml.example config/ginblog.yaml
```

### 2. 运行

```bash
go run .
```

首次运行会自动创建日志目录（`logs/`）和数据库目录（`data/`）。启动成功后输出类似：

```json
{"time":"...","level":"INFO","source":{"function":"main.main","file":".../main.go","line":36},"msg":"init logger success"}
{"time":"...","level":"INFO","source":{"function":"main.main","file":".../main.go","line":54},"msg":"init database success"}
{"time":"...","level":"INFO","source":{"function":"main.main","file":".../main.go","line":73},"msg":"migrate all database success"}
{"time":"...","level":"INFO","source":{"function":"main.main","file":".../main.go","line":94},"msg":"init redis success"}
{"time":"...","level":"INFO","source":{"function":"main.main","file":".../main.go","line":104},"msg":"init session store success"}
{"time":"...","level":"INFO","source":{"function":"main.main","file":".../main.go","line":138},"msg":"start server success"}
```

访问测试接口：

```bash
curl http://localhost:8080/
# {"message":"hello world"}
```

### 3. 编译

```bash
go build -o build/ginblog.exe .
```

### 4. 停止

支持 `Ctrl+C` / `SIGTERM` 信号，触发**优雅退出**：等待在途请求完成（最多 10 秒）后关闭服务，并关闭数据库连接、Redis 连接与日志文件。

## 配置说明

| 键 | 说明 | 默认示例 |
|---|---|---|
| `server.server_name` | 服务名称 | `ginblog` |
| `server.port` | 监听端口 | `8080` |
| `server.read_timeout` | 读超时（秒） | `15` |
| `server.write_timeout` | 写超时（秒） | `15` |
| `server.idle_timeout` | 空闲连接超时（秒） | `60` |
| `server.server_mode` | gin 运行模式：`debug` / `release` / `test`（未配置默认 `release`） | `release` |
| `logger.level` | 日志等级：`debug` / `info` / `warn` / `error` / `silent`（不区分大小写） | `info` |
| `logger.log_dir` | 日志目录（自动创建） | `./logs` |
| `logger.log_file` | 日志文件名 | `ginblog.log` |
| `logger.max_size` | 单个日志文件最大体积（MB） | `100` |
| `logger.max_age` | 旧日志最多保留天数 | `7` |
| `logger.max_backups` | 最多保留几个日志文件 | `7` |
| `logger.compress` | 是否压缩旧日志 | `true` |
| `database.file` | SQLite 数据库文件路径 | `./data/ginblog.db` |
| `redis.addr` | Redis 地址 | `127.0.0.1:6379` |
| `redis.password` | Redis 密码（无密码留空） | `""` |
| `redis.db` | 逻辑库编号 | `0` |
| `redis.dial_timeout` | 连接超时（**必须带单位**，`3s` 不是 3 纳秒） | `3s` |
| `redis.read_timeout` | 读超时 | `3s` |
| `redis.write_timeout` | 写超时 | `3s` |
| `redis.pool_size` | 连接池大小（0 = `10 * GOMAXPROCS`） | `10` |
| `auth.cookie_name` | 登录态 Cookie 名称 | `ginblog_sid` |
| `auth.expire_hours` | 登录态有效期（小时）；Cookie maxAge 与 Redis TTL 同源于此值 | `168` |
| `auth.secure` | 仅通过 HTTPS 下发 Cookie（本地开发保持 `false`） | `false` |
| `auth.session_prefix` | Session key 前缀，最终 key 形如 `{prefix}{token}`，与同库其他业务隔离 | `ginblog:` |

启动期校验：`redis.addr` 非空、三个超时必须带时间单位、`auth.expire_hours > 0`、`auth.cookie_name` 非空，不满足直接退出。

## 认证与会话

**方案：Redis Session + HttpOnly Cookie**（不是 JWT——凭证可主动作废，登出/踢人立即生效）。

```text
登录 POST /api/login
  └─ service 验证 bcrypt 密码 → session.Create(userID) 生成 32 字节随机 token
       ├─ Redis:  SET  {prefix}{token} = userID   EXPIRE = expire_hours
       └─ handler: Set-Cookie ginblog_sid={token}（HttpOnly、path=/、maxAge=expire_hours*3600）

后续请求
  └─ RequireAuth 中间件：Cookie 取 token → session.Get → 写入 context（KeyUserID = "user_id"）
       ├─ token 缺失/过期 → 401 + code=10002
       ├─ Redis 故障      → 500 + code=50000（不误报 401，避免把在线用户踢回登录页）
       └─ 成功            → handler 用 c.GetUint(middleware.KeyUserID) 拿当前用户

登出 POST /api/logout
  └─ session.Delete(token) + Set-Cookie(maxAge=-1) 清除；未带 Cookie 视为已登出，幂等返回成功
```

**三个中间件**（`middleware/auth.go`，按路由组挂载，不全局启用）：

| 中间件 | 作用 | 失败响应 |
|---|---|---|
| `RequireAuth` | 必须登录：校验 Cookie → Redis，注入 `KeyUserID` | `401` + `10002 未登录或登录态已失效` |
| `RequireAdmin` | 必须具备指定角色（查库实时校验，降级立即生效） | `403` + `10003 没有权限执行该操作` |
| `OptionalAuth` | 有登录态就注入，没有也放行（游客视图） | 不拦截 |

挂载顺序固定为 `RequireAuth` → `RequireAdmin` → handler：后者依赖前者注入的 `KeyUserID`，反序会导致取到 0 而误判。

**角色**（`model.UserInfo.Role`，注册默认 `user`）：`user` / `moderator` / `admin`。

**密码**：bcrypt 哈希存储（`x/crypto/bcrypt`，`DefaultCost`），`PasswordHash` 在模型上 `json:"-"`、DTO 不含该字段，双重防线不外泄。

## 日志

- **格式**：JSON，每行一条。应用日志带 `source`（文件:行号），GORM 日志不带 `source`（调用方恒为桥接层，无参考价值）。
- **输出**：`io.MultiWriter` 同时写**控制台**和**日志文件**（`logs/ginblog.log`）。
- **切割**：lumberjack 按 `max_size` 分卷，按 `max_age` / `max_backups` 清理，可选 gzip 压缩。
- **Duration 字段**：统一格式化为可读字符串，如 `"latency":"1.2ms"`（而非裸纳秒数字）。
- **访问日志中间件**（`middleware/logger.go`）：记录 `method`、`path`、`query`、`status`、`latency`、`size`、`client_ip`、`user_agent`、`errors`；`5xx` 记为 `ERROR`、`4xx` 记为 `WARN`，其余为 `INFO`。
- **panic 恢复中间件**（`middleware/recovery.go`）：捕获 handler 中的 panic，记录 ERROR 日志（含堆栈），返回 500；注册在 Logger **内层**，panic 恢复后访问日志不会丢失，panic 信息会出现在访问日志的 `errors` 字段。
- **GORM 日志桥接**：`logger.NewGormLogger` 将 SQL 日志接入 slog，按 `logger.level` 映射：

  | `logger.level` | GORM 等级 | 输出内容 |
  |---|---|---|
  | `debug` | Info | 每条 SQL（slog `DEBUG`） |
  | `info` / `warn` | Warn | 仅错误 + 慢查询（阈值 200ms） |
  | `error` | Error | 仅错误 |
  | `silent` | Silent | 不输出 |

- **gin 自身日志**：转发到 slog（`tag=gin`），避免与 JSON 格式混排；运行模式由 `server.server_mode` 配置（`debug` 模式可查看路由注册信息）。

## 数据库

- **驱动**：SQLite（`glebarez/sqlite`，纯 Go），路径由 `database.file` 配置，所在目录自动创建。
- **PRAGMA**：初始化时执行 `journal_mode=WAL`、`busy_timeout=5000`、`foreign_keys=ON`、`synchronous=NORMAL`，保证并发读写与外键约束生效。
- **连接池**：`MaxOpenConns=1`（SQLite 写操作串行，避免自锁），空闲连接 1 小时回收。
- **错误翻译**：`TranslateError: true`，唯一索引冲突会被 GORM 翻译为 `gorm.ErrDuplicatedKey`——用户注册/创建的唯一性校验**不走「先查后插」**（有竞态），直接依赖约束冲突判定，需要区分是用户名还是邮箱撞了时才回查一次。
- **自动迁移**：启动时 `database.Migrate` → `AutoMigrate`，当前模型：`model.Article`、`model.UserInfo`；迁移失败会记录日志并退出。

## 响应格式

所有业务接口统一返回：

```json
{"code": 0, "msg": "success", "data": {}}
```

- `code == 0` 为成功；前端以 `code` 做分支判断，`msg` 仅用于展示。
- 分页接口的 `data` 为 `{ "total": 0, "list": [], "page": 1, "size": 20 }`。
- 业务错误码与 HTTP 状态码分开维护（`pkg/response`）：

| 段 | 示例 | 含义 |
|---|---|---|
| `0` | — | 成功 |
| `1xxxx` | `10001` 参数校验失败、`10002` 未登录、`10003` 无权限 | 客户端错误 |
| `2xxxx` | `20001` 资源不存在、`20002` 资源冲突（用户名/邮箱重复） | 业务错误 |
| `5xxxx` | `50000` 内部错误、`50001` 数据库错误 | 服务端错误 |

## 接口

### 公开（无需登录）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/` | 欢迎接口，返回 `{"message":"hello world"}` |
| POST | `/api/login` | 登录；body：`{user_name, password}`；成功后凭证经 `Set-Cookie` 下发，不进响应体 |
| POST | `/api/register` | 注册；body：`{user_name, nick_name, email, password, confirm_password}` |

### 需要登录（`RequireAuth`）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/me` | 当前登录用户信息 |
| POST | `/api/update_me` | 修改个人资料；body：`{nick_name?, email?}` 全可选 |
| POST | `/api/change_password` | 修改密码；body：`{old_password, new_password, confirm_password}` |
| POST | `/api/logout` | 登出：删 session + 清 Cookie，幂等 |

### 管理员（`RequireAuth` + `RequireAdmin`，仅 `role=admin`）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/user/list` | 用户列表；参数：`page`、`page_size`（≤100）、`keyword`、`sort`（`user_name`/`nick_name`/`email`/`role`/`created_at`，`-` 前缀降序） |
| GET | `/api/user/detail?id=` | 用户详情 |
| POST | `/api/user/create` | 创建用户；body：`{user_name, nick_name, email, password, role?}`（`role` ∈ `user/moderator/admin`，缺省 `user`） |
| POST | `/api/user/update?id=` | 修改用户；body：`{nick_name?, email?, role?}` |
| POST | `/api/user/delete?id=` | 删除用户 |

### 文章（当前**未挂认证中间件**，公开可访问）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/article/list` | 文章列表；参数：`page`、`page_size`（≤100）、`keyword`、`sort`（`title`/`created_at` 等，`-` 前缀降序） |
| GET | `/api/article/detail?id=` | 文章详情 |
| POST | `/api/article/create` | 创建文章；body：`{title, description, content}` |
| POST | `/api/article/update?id=` | 更新文章；body 同上 |
| POST | `/api/article/delete?id=` | 删除文章 |

## 路线规划

- [x] 配置加载（viper + YAML，默认值 + 启动期校验）
- [x] 结构化日志与滚动切割
- [x] 访问日志中间件
- [x] 优雅退出
- [x] 数据库接入（GORM + SQLite，WAL + 自动迁移 + 错误翻译）
- [x] GORM / gin 日志接入 slog
- [x] panic 恢复（Recovery 中间件）
- [x] 文章 CRUD 接口（分页、关键词搜索、排序）
- [x] Redis 接入与 Session 存储（Redis Session + Cookie，非 JWT）
- [x] 用户注册 / 登录 / 登出 / 个人信息 / 修改密码
- [x] 认证与鉴权中间件（RequireAuth / RequireAdmin，双层路由保护）
- [x] 管理员用户管理（列表 / 详情 / 创建 / 修改 / 删除）
- [ ] 文章接口鉴权与归属校验（`RequireAuth`、`UserOwnsArticle` 已就位，尚未接线）
- [ ] 管理员重置密码与强制首次改密（临时密码 + `MustChangePassword` + 踢 session）
- [ ] 会话列表与多设备管理（`/api/user/sessions`，需补反向索引）
- [ ] 标签 / 分类 CRUD
- [ ] 评论与分页

## 常见问题

**Q：启动报 `failed to create log dir: mkdir : The system cannot find the path specified`？**

配置字段未映射上，`logger.log_dir` 为空。注意：

1. 使用 `log_dir` / `log_file` 键，不要写成 `file`；
2. 结构体必须带 `mapstructure:"log_dir"` 标签——`viper.Unmarshal` 底层是 mapstructure，**只认 `mapstructure` 标签，不认 `yaml` 标签**。没有该标签时，含下划线的键（`read_timeout`、`log_dir` 等）会全部解析失败，字段保持零值。

**Q：启动报 `failed to init redis: failed to ping redis: ...`？**

Redis 没起来或 `redis.addr` 不对。确认 Redis 进程与端口：`redis-cli ping` 应返回 `PONG`。本机可先 `docker run -d -p 6379:6379 redis:7-alpine`。

**Q：`redis.dial_timeout = 3` 导致连接几乎立刻失败？**

时间类型必须带单位：Go 的 `time.Duration` 解析裸数字得到的是**纳秒**，`3` = 3ns。写成 `3s`。

**Q：接口返回 `401 / code=10002`，但确定刚登录过？**

Cookie 没带上（前端跨域需 `credentials: 'include'` 且服务端允许携带凭证），或 session 已过期（`auth.expire_hours`，默认 168 小时），或重启 Redis 后 session 丢失。

## License

[MIT](LICENSE)
