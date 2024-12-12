这是一个go编写的监控文件变更,自动编译,自动热重启的程序

## 配置说明

程序通过 `example.toml` 进行配置,主要包含以下配置项:
1
### 应用配置 (app)
- name: 应用名称
- version: 应用版本

### 监控配置 (watch)
- directories: 需要监控的目录列表
- ignore_patterns: 忽略的文件或目录模式
- extensions: 监控的文件扩展名

### 构建配置 (build)
- main_path: 主程序入口路径
- output: 编译输出路径
- command: 编译命令
