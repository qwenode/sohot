# GoReleaser Configuration

This directory documents the automated release configuration in `.goreleaser.yaml`.

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

These are automatically provided by the GitHub Actions workflow.
