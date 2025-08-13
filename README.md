# SoHot

A Go-based file monitoring and hot-reload tool that automatically watches for file changes, compiles your Go applications, and restarts them during development. Similar to nodemon for Node.js, SoHot streamlines your Go development workflow by eliminating manual build and restart cycles.

## Features

- **Intelligent File Watching**: Monitors specified directories for changes with configurable include/exclude patterns
- **Smart Compilation**: Delayed compilation mechanism prevents excessive rebuilds during rapid file changes
- **Hot Reload**: Automatically restarts your application after successful compilation
- **Interactive Configuration**: Choose between multiple run profiles with an intuitive command-line interface
- **Run-Only Mode**: Monitor and restart pre-built executables without recompilation
- **Cross-Platform**: Works on Windows, macOS, and Linux with platform-specific optimizations
- **Structured Logging**: Comprehensive logging with configurable levels using zerolog
- **Process Management**: Graceful process termination and cleanup with signal handling

## Installation

### Prerequisites

- Go 1.23.1 or later
- Git (for cloning the repository)

### Install from Source

```bash
git clone https://github.com/qwenode/sohot.git
cd sohot
go build -o sohot
```

### Install Globally

```bash
go install github.com/qwenode/sohot@latest
```

## Quick Start

1. **Create a configuration file** (`sohot.toml`) in your project root:

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

2. **Start SoHot**:

```bash
# Interactive mode - choose profile
sohot

# Direct mode - specify profile name
sohot dev
```

3. **Start developing** - SoHot will automatically rebuild and restart your application when files change.

## Configuration

SoHot uses a TOML configuration file (`sohot.toml`) with the following sections:

### Log Configuration

```toml
[log]
level = -1  # Log level (-1: trace, 0: debug, 1: info, 2: warn, 3: error, 4: fatal)
```

### Watch Configuration

```toml
[watch]
include = ["."]           # Directories to monitor
exclude = ["tmp/"]        # Directories to exclude (*.git, .idea, *.exe automatically excluded)
```

### Build Configuration

```toml
[build]
delay = 1000             # Compilation delay in milliseconds
name = "./tmp/test.exe"  # Output executable path
package = "main.go"      # Main package path
command = []             # Additional build arguments
```

### Run Profiles

Define multiple run configurations for different environments:

```toml
[run.development]
command = ["--debug", "--port", "3000"]

[run.production]
only = true              # Run-only mode (no compilation)
command = ["--env", "prod"]

[run.testing]
only = true
command = ["--test-mode"]
```

#### Run Profile Options

- **command**: Array of command-line arguments to pass to your application
- **only**: Boolean flag for run-only mode (monitors executable file instead of source files)


## Usage Examples

### Basic Web Server Development

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

### Microservice with Multiple Environments

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

### Run-Only Mode for Pre-built Applications

```toml
[run.production]
only = true
command = ["--env", "production", "--workers", "4"]
```

### Reporting Issues

Please use the GitHub issue tracker to report bugs or request features. Include:

- Go version
- Operating system
- Configuration file (if relevant)
- Steps to reproduce
- Expected vs actual behavior

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with excellent Go libraries from the community
- Thanks to all contributors and users

---

**Happy coding with SoHot!** 🔥