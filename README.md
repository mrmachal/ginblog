# ginblog

基于 **Gin + Viper + slog** 的博客后端服务（进行中）。

当前已完成：配置加载、结构化日志（控制台 + 文件 + 滚动切割）、访问日志中间件、HTTP 服务启动与优雅退出。

## 技术栈

| 组件 | 用途 |
|---|---|
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | HTTP 框架 |
| [spf13/viper](https://github.com/spf13/viper) | YAML 配置加载 |
| `log/slog`（标准库） | 结构化日志（JSON 格式） |
| [lumberjack.v2](https://github.com/natefinch/lumberjack) | 日志文件滚动切割 |

> Go 版本要求：`go 1.25.0+`

## 项目结构

```
ginblog/
├── main.go                    # 入口：加载配置 → 初始化日志 → 注册路由 → 启动服务 → 优雅退出
├── config/
│   ├── config.go              # 配置结构体定义与加载（viper）
│   ├── ginblog.yaml           # 实际使用的配置文件
│   └── ginblog.yaml.example   # 配置示例（含注释）
├── middleware/
│   └── logger.go              # 访问日志中间件（slog，按状态码分级）
├── pkg/
│   └── logger/
│       └── logger.go          # 日志初始化：slog + MultiWriter + lumberjack
├── build/                     # 编译产物（已 gitignore）
├── logger/                    # 运行时日志目录（已 gitignore）
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

启动成功后输出类似：

```json
{"time":"...","level":"INFO","msg":"init logger success"}
{"time":"...","level":"INFO","msg":"start server success"}
```

访问测试接口：

```bash
curl http://localhost:8080/
# {"message":"hello workd"}
```

### 3. 编译

```bash
go build -o build/ginblog.exe .
```

### 4. 停止

支持 `Ctrl+C` / `SIGTERM` 信号，触发**优雅退出**：等待在途请求完成（最多 10 秒）后关闭服务。

## 配置说明

| 键 | 说明 | 默认示例 |
|---|---|---|
| `server.server_name` | 服务名称 | `ginblog` |
| `server.port` | 监听端口 | `8080` |
| `server.read_timeout` | 读超时（秒） | `15` |
| `server.write_timeout` | 写超时（秒） | `15` |
| `server.idle_timeout` | 空闲连接超时（秒） | `60` |
| `logger.level` | 日志等级：`DEBUG` / `INFO` / `WARN` / `ERROR` | `info` |
| `logger.log_dir` | 日志目录（自动创建） | `./logger` |
| `logger.log_file` | 日志文件名 | `ginblog.log` |
| `logger.max_size` | 单个日志文件最大体积（MB） | `100` |
| `logger.max_age` | 旧日志最多保留天数 | `7` |
| `logger.max_backups` | 最多保留几个日志文件 | `7` |
| `logger.compress` | 是否压缩旧日志 | `true` |
| `database.file` | SQLite 数据库文件路径 | `./data/ginblog.db` |

## 日志

- **格式**：JSON，每行一条，带 `time` / `level` / `source`（文件:行号）/ `msg` 及业务字段。
- **输出**：`io.MultiWriter` 同时写**控制台**和**日志文件**。
- **切割**：lumberjack 按 `max_size` 分卷，按 `max_age` / `max_backups` 清理，可选 gzip 压缩。
- **访问日志中间件**（`middleware/logger.go`）：记录 `method`、`path`、`query`、`status`、`latency`、`size`、`client_ip`、`user_agent`、`errors`；`5xx`/`4xx` 记为 `ERROR`，其余为 `INFO`。

## 接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/` | 欢迎接口，返回 `{"message":"hello workd"}` |

## 路线规划

- [x] 配置加载（viper + YAML）
- [x] 结构化日志与滚动切割
- [x] 访问日志中间件
- [x] 优雅退出
- [ ] 数据库接入（GORM + SQLite，配置项 `database.file` 已预留）
- [ ] 用户认证（JWT）
- [ ] 文章 / 标签 / 分类 CRUD
- [ ] 评论与分页

## 常见问题

**Q：启动报 `faild to create log dir: mkdir : The system cannot find the path specified`？**

配置字段未映射上，`logger.log_dir` 为空。注意：

1. 使用 `log_dir` / `log_file` 键，不要写成 `file`；
2. 结构体必须带 `mapstructure:"log_dir"` 标签——`viper.Unmarshal` 底层是 mapstructure，**只认 `mapstructure` 标签，不认 `yaml` 标签**。没有该标签时，含下划线的键（`read_timeout`、`log_dir` 等）会全部解析失败，字段保持零值。

## License

[MIT](LICENSE)
