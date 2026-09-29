# ginblog

基于 **Gin + GORM + Viper + slog** 的博客后端服务（进行中）。

当前已完成：配置加载、结构化日志（控制台 + 文件 + 滚动切割）、GORM 日志桥接、访问日志中间件、panic 恢复、数据库接入（GORM + SQLite，自动迁移）、HTTP 服务启动与优雅退出。

## 技术栈

| 组件 | 用途 |
|---|---|
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP 框架 |
| [spf13/viper](https://github.com/spf13/viper) | YAML 配置加载 |
| `log/slog`（标准库） | 结构化日志（JSON 格式） |
| [lumberjack.v2](https://github.com/natefinch/lumberjack) | 日志文件滚动切割 |
| [gorm.io/gorm](https://gorm.io) | ORM（模型定义、自动迁移） |
| [glebarez/sqlite](https://github.com/glebarez/sqlite) | SQLite 驱动（纯 Go，无需 CGO） |

> Go 版本要求：`go 1.25.0+`

## 项目结构

```text
ginblog/
├── main.go                    # 入口：配置 → 日志 → 数据库 → 迁移 → 路由 → 启动 → 优雅退出
├── config/
│   ├── config.go              # 配置结构体定义与加载（viper）
│   ├── ginblog.yaml           # 实际使用的配置文件（已 gitignore）
│   └── ginblog.yaml.example   # 配置示例（含注释）
├── middleware/
│   ├── logger.go              # 访问日志中间件（slog，按状态码分级）
│   └── recovery.go            # panic 恢复中间件（记录堆栈 + 返回 500）
├── model/
│   └── article.go             # 文章模型（gorm.Model + 标题/摘要/内容）
├── pkg/
│   ├── database/
│   │   └── database.go        # GORM + SQLite 初始化（PRAGMA、连接池）与自动迁移
│   └── logger/
│       └── logger.go          # slog 初始化、GORM 日志桥接、gin 日志转发
├── build/                     # 编译产物（已 gitignore）
├── data/                      # SQLite 数据库文件（已 gitignore，自动创建）
├── logs/                      # 运行时日志目录（已 gitignore，自动创建）
├── go.mod
└── go.sum
```

## 快速开始

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
{"time":"...","level":"INFO","source":{"function":"main.main","file":".../main.go","line":112},"msg":"start server success"}
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

支持 `Ctrl+C` / `SIGTERM` 信号，触发**优雅退出**：等待在途请求完成（最多 10 秒）后关闭服务，并关闭数据库连接与日志文件。

## 配置说明

| 键 | 说明 | 默认示例 |
|---|---|---|
| `server.server_name` | 服务名称 | `ginblog` |
| `server.port` | 监听端口 | `8080` |
| `server.read_timeout` | 读超时（秒） | `15` |
| `server.write_timeout` | 写超时（秒） | `15` |
| `server.idle_timeout` | 空闲连接超时（秒） | `60` |
| `logger.level` | 日志等级：`debug` / `info` / `warn` / `error` / `silent`（不区分大小写） | `info` |
| `logger.log_dir` | 日志目录（自动创建） | `./logs` |
| `logger.log_file` | 日志文件名 | `ginblog.log` |
| `logger.max_size` | 单个日志文件最大体积（MB） | `100` |
| `logger.max_age` | 旧日志最多保留天数 | `7` |
| `logger.max_backups` | 最多保留几个日志文件 | `7` |
| `logger.compress` | 是否压缩旧日志 | `true` |
| `database.file` | SQLite 数据库文件路径 | `./data/ginblog.db` |

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

- **gin 自身日志**：转发到 slog（`tag=gin`），避免与 JSON 格式混排；默认 release 模式，调试路由信息可设置 `GIN_MODE=debug`。

## 数据库

- **驱动**：SQLite（`glebarez/sqlite`，纯 Go），路径由 `database.file` 配置，所在目录自动创建。
- **PRAGMA**：初始化时执行 `journal_mode=WAL`、`busy_timeout=5000`、`foreign_keys=ON`、`synchronous=NORMAL`，保证并发读写与外键约束生效。
- **连接池**：`MaxOpenConns=1`（SQLite 写操作串行，避免自锁），空闲连接 1 小时回收。
- **自动迁移**：启动时 `database.Migrate` → `AutoMigrate`，当前模型：`model.Article`；迁移失败会记录日志并退出。

## 接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/` | 欢迎接口，返回 `{"message":"hello world"}` |

## 路线规划

- [x] 配置加载（viper + YAML）
- [x] 结构化日志与滚动切割
- [x] 访问日志中间件
- [x] 优雅退出
- [x] 数据库接入（GORM + SQLite，WAL + 自动迁移）
- [x] GORM / gin 日志接入 slog
- [x] panic 恢复（Recovery 中间件）
- [ ] 用户认证（JWT）
- [ ] 文章 / 标签 / 分类 CRUD
- [ ] 评论与分页

## 常见问题

**Q：启动报 `failed to create log dir: mkdir : The system cannot find the path specified`？**

配置字段未映射上，`logger.log_dir` 为空。注意：

1. 使用 `log_dir` / `log_file` 键，不要写成 `file`；
2. 结构体必须带 `mapstructure:"log_dir"` 标签——`viper.Unmarshal` 底层是 mapstructure，**只认 `mapstructure` 标签，不认 `yaml` 标签**。没有该标签时，含下划线的键（`read_timeout`、`log_dir` 等）会全部解析失败，字段保持零值。

## License

[MIT](LICENSE)
