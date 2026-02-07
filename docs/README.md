# Documentation Directory

This directory contains additional documentation for the vibe-project.

## Available Documentation

- [../ARCHITECTURE.md](../ARCHITECTURE.md) - Comprehensive architecture documentation (Japanese/English)
- [../README.md](../README.md) - User guide and quick start

## Understanding the Codebase

For developers who want to understand how vibe-project works:

1. Start with [../README.md](../README.md) to understand what the tool does
2. Read [../ARCHITECTURE.md](../ARCHITECTURE.md) for detailed technical documentation
3. Explore the code starting from:
   - `cmd/vibe/main.go` - Entry point
   - `internal/cli/root.go` - CLI initialization
   - `internal/cli/run.go` - Main task execution logic
   - `internal/github/task.go` - GitHub integration
   - `internal/claude/executor.go` - Claude Code integration

## Quick Code Navigation

### CLI Commands
- `internal/cli/auth.go` - Authentication (login/logout/status)
- `internal/cli/project.go` - Project management
- `internal/cli/task.go` - Task listing
- `internal/cli/run.go` - Task execution
- `internal/cli/watch.go` - Auto-execution mode

### Core Business Logic
- `internal/domain/task.go` - Task model and states
- `internal/domain/execution.go` - Execution result model
- `internal/github/task.go` - GitHub Project integration
- `internal/claude/executor.go` - Claude Code integration
- `internal/config/config.go` - Configuration management
