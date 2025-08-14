# GoReleaser Configuration

This directory contains configuration files for the automated release system.

## Files

### `release-notes.tmpl`
Custom template for generating GitHub release notes. This template provides:

- **Structured release information** with version, commit, and build details
- **Download table** with direct links to all platform binaries
- **Verification instructions** for checksum validation
- **Quick start guide** for installation and usage
- **Changelog integration** with categorized changes
- **Support links** to documentation and issue tracking

### Template Variables

The template uses GoReleaser's built-in template variables:

- `{{ .Tag }}` - Git tag (e.g., "v1.0.0")
- `{{ .Date }}` - Build date
- `{{ .ShortCommit }}` - Short commit hash
- `{{ .ProjectName }}` - Project name from configuration
- `{{ .Changes }}` - List of changes from changelog
- `{{ .Env.GITHUB_REPOSITORY_OWNER }}` - Repository owner from environment
- `{{ .Env.GITHUB_REPOSITORY_NAME }}` - Repository name from environment
- `{{ .PreviousTag }}` - Previous git tag for changelog links

## Checksum Configuration

The release system generates SHA256 checksums for:

- All binary archives (tar.gz, zip)
- Additional files (README.md, README_CN.md, LICENSE)
- Output stored in `checksums.txt`

## Changelog Configuration

Automatic changelog generation with categorized commit types:

- 🚀 **Features** - commits starting with `feat:`
- 🐛 **Bug Fixes** - commits starting with `fix:`
- 📚 **Documentation** - commits starting with `docs:`
- 🔧 **Improvements** - commits starting with `perf:` or `improve:`
- 🧹 **Maintenance** - commits starting with `build:` or `deps:`
- 📦 **Other Changes** - all other commits

Excluded commit types: `docs:`, `test:`, `chore:`, `ci:`, `style:`, `refactor:`, merge commits.

## Environment Variables

Required environment variables for release metadata:

- `GITHUB_REPOSITORY_OWNER` - Repository owner name
- `GITHUB_REPOSITORY_NAME` - Repository name
- `GO_VERSION` - Go version used for building (optional, defaults to "1.21+")

These are automatically provided by the GitHub Actions workflow.