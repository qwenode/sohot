# SoHot 

English | [简体中文](README_CN.md)

A Go-based file monitoring and hot-reload tool that automatically watches for file changes, compiles your Go applications, and restarts them during development. Similar to nodemon for Node.js, SoHot streamlines your Go development workflow by eliminating manual build and restart cycles.

## Features

- **Intelligent File Watching**: Monitors specified directories for changes with configurable include/exclude patterns
- **Smart Compilation**: Delayed compilation mechanism prevents excessive rebuilds during rapid file changes
- **Hot Reload**: Automatically restarts your application after successful compilation
- **Interactive Configuration**: Choose between multiple run profiles with an intuitive command-line interface
- **Run-Only Mode**: Monitor and restart pre-built executables without recompilation
- **Cross-Platform**: Works on Windows, macOS, and Linux with platform-specific optimizations
- **Structured Logging**: Comprehensive logging with configurable levels using zerolog
- **Process Management**: Stops the running application before each restart, logs when it exits on its own, and cleans up temporary build files on exit

## Installation

### Prerequisites

- Go 1.27.1 or later
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
package = "."
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

# Direct mode - specify profile name (case-insensitive)
sohot dev

# Show version information
sohot --version
```

3. **Start developing** - SoHot will automatically rebuild and restart your application when files change.

## Configuration

SoHot uses a TOML configuration file (`sohot.toml`) with the following sections:

### Log Configuration

```toml
[log]
level = -1    # Log level (-1: trace, 0: debug, 1: info, 2: warn, 3: error, 4: fatal)
lang = "en"   # Log language: en or zh (default: en)
```

### Watch Configuration

```toml
[watch]
include = ["."]           # Directories to watch recursively; new subdirectories are picked up automatically (default: ["."])
exclude = ["tmp/"]        # Path fragments to exclude; .git and .idea directories, *.exe files and the build output are always ignored
```

### Build Configuration

```toml
[build]
delay = 1000             # Compilation delay in milliseconds (default: 500)
name = "./tmp/app"       # Output executable path, required (".exe" is appended automatically on Windows)
package = "."            # Package to build, e.g. "." or "./cmd/server" (default: ".")
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
- **only**: Boolean flag for run-only mode (skips compilation and restarts the application only when the executable at `build.name` changes)

At least one run profile is required. Profile names are case-insensitive, so `sohot Dev` and `sohot dev` select the same profile.


## Usage Examples

### Basic Web Server Development

```toml
[watch]
include = ["."]
exclude = ["static/", "tmp/"]

[build]
delay = 500
name = "./tmp/server"
package = "./cmd/server"

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

## Development

### Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/qwenode/sohot.git
cd sohot

# Install dependencies
go mod download

# Run tests
go test ./...

# Build the project
go build -o sohot
```

### Code Style

- Follow the official Go code style guidelines
- Format code with `gofmt`
- Add appropriate comments and documentation
- Write unit tests for new features

## Troubleshooting

### Common Issues

**Q: SoHot fails to start or cannot read the configuration file**
A: Check that `sohot.toml` is valid TOML. `build.name` and at least one `[run.<name>]` profile are required; `build.package` defaults to `"."` and `watch.include` defaults to `["."]`.

**Q: File changes do not trigger a rebuild**
A: Check `watch.include` and `watch.exclude`. Exclude entries are matched as path fragments, so `tmp/` also matches `mytmp/`. `.git` and `.idea` directories, `*.exe` files and the build output are always ignored.

**Q: Deleting the executable fails on Windows**
A: This is expected. SoHot handles locked files automatically and cleans up the temporary files.

**Q: Permission errors during compilation**
A: Make sure the output directory is writable, or point `build.name` to a directory you can write to.

### Debug Mode

Enable verbose logging to diagnose issues:

```toml
[log]
level = -1  # Enable trace-level logging
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