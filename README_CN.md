# SoHot

一个基于 Go 语言的文件监控和热重载工具，能够自动监控文件变更、编译 Go 应用程序并在开发过程中重启它们。类似于 Node.js 的 nodemon，SoHot 通过消除手动构建和重启循环来简化您的 Go 开发工作流程。

## 功能特性

- **智能文件监控**：监控指定目录的变更，支持可配置的包含/排除模式
- **智能编译**：延迟编译机制防止文件快速变更时的过度重建
- **热重载**：编译成功后自动重启应用程序
- **交互式配置**：通过直观的命令行界面在多个运行配置文件之间选择
- **仅运行模式**：监控并重启预构建的可执行文件，无需重新编译
- **跨平台**：在 Windows、macOS 和 Linux 上运行，具有平台特定优化
- **结构化日志**：使用 zerolog 提供可配置级别的全面日志记录
- **进程管理**：通过信号处理实现优雅的进程终止和清理

## 安装

### 前置要求

- Go 1.23.1 或更高版本
- Git（用于克隆仓库）

### 从源码安装

```bash
git clone https://github.com/qwenode/sohot.git
cd sohot
go build -o sohot
```

### 全局安装

```bash
go install github.com/qwenode/sohot@latest
```

## 快速开始

1. **在项目根目录创建配置文件** (`sohot.toml`)：

```toml
[log]
level = -1

[watch]
include = ["."]
exclude = ["tmp/", "vendor/"]

[build]
delay = 1000
name = "./tmp/app"
package = "main.go"
command = []

[run.dev]
command = ["--port", "8080"]

[run.prod]
only = true
command = ["--env", "production"]
```

2. **启动 SoHot**：

```bash
# 交互模式 - 选择配置文件
sohot

# 直接模式 - 指定配置文件名称
sohot dev
```

3. **开始开发** - SoHot 将在文件变更时自动重建并重启您的应用程序。

## 配置

SoHot 使用 TOML 配置文件 (`sohot.toml`)，包含以下部分：

### 日志配置

```toml
[log]
level = -1  # 日志级别 (-1: trace, 0: debug, 1: info, 2: warn, 3: error, 4: fatal)
```

### 监控配置

```toml
[watch]
include = ["."]           # 要监控的目录
exclude = ["tmp/"]        # 要排除的目录（*.git, .idea, *.exe 自动排除）
```

### 构建配置

```toml
[build]
delay = 1000             # 编译延迟（毫秒）
name = "./tmp/test.exe"  # 输出可执行文件路径
package = "main.go"      # 主包路径
command = []             # 额外的构建参数
```

### 运行配置文件

为不同环境定义多个运行配置：

```toml
[run.development]
command = ["--debug", "--port", "3000"]

[run.production]
only = true              # 仅运行模式（无编译）
command = ["--env", "prod"]

[run.testing]
command = ["--test-mode"]
```

#### 运行配置文件选项

- **command**：传递给应用程序的命令行参数数组
- **only**：仅运行模式的布尔标志（监控可执行文件而不是源文件）

## 使用示例

### 基础 Web 服务器开发

```toml
[watch]
include = ["."]
exclude = ["static/", "tmp/"]

[build]
delay = 500
name = "./tmp/server"
package = "cmd/server/main.go"

[run.dev]
command = ["--port", "8080", "--debug"]
```

### 多环境微服务

```toml
[run.local]
command = ["--config", "local.yaml"]

[run.staging]
only = true
command = ["--config", "staging.yaml", "--log-level", "info"]

[run.debug]
only = true
command = ["--config", "debug.yaml", "--pprof"]
```

### 预构建应用程序的仅运行模式

```toml
[run.production]
only = true
command = ["--env", "production", "--workers", "4"]
```

## 开发指南

### 贡献代码

我们欢迎社区贡献！请遵循以下步骤：

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

### 开发环境设置

```bash
# 克隆仓库
git clone https://github.com/qwenode/sohot.git
cd sohot

# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建项目
go build -o sohot
```

### 代码规范

- 遵循 Go 官方代码风格指南
- 使用 `gofmt` 格式化代码
- 添加适当的注释和文档
- 编写单元测试覆盖新功能

## 故障排除

### 常见问题

**Q: 程序无法启动或配置文件读取失败**
A: 检查 `sohot.toml` 文件格式是否正确，确保所有必需字段都已配置。

**Q: 文件变更未触发重新编译**
A: 验证 `watch.include` 和 `watch.exclude` 配置，确保目标文件在监控范围内。

**Q: Windows 下可执行文件删除失败**
A: 这是正常现象，SoHot 会自动处理文件锁定问题并清理临时文件。

**Q: 编译过程中出现权限错误**
A: 确保对输出目录有写权限，或更改 `build.name` 配置到有权限的目录。

### 调试模式

启用详细日志记录以诊断问题：

```toml
[log]
level = -1  # 启用 trace 级别日志
```

### 报告问题

请使用 GitHub 问题跟踪器报告错误或请求功能。请包含：

- Go 版本
- 操作系统
- 配置文件（如相关）
- 重现步骤
- 预期行为与实际行为

## 许可证

本项目采用 MIT 许可证 - 详情请参阅 [LICENSE](LICENSE) 文件。

## 致谢

- 使用了来自 Go 社区的优秀库构建
- 感谢所有贡献者和用户

---

**使用 SoHot 愉快编码！** 🔥