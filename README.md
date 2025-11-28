# vibe-project

A CLI tool that integrates GitHub Projects with Claude Code

## Overview

Fetch tasks from GitHub Project V2, execute them with Claude Code, and update the results back to the project.

```
┌─────────────────────────────────────────────────────────────────┐
│                      GitHub Project V2                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Task: Implement user authentication API                 │   │
│  │ Status: Ready → InProgress → InReview                   │   │
│  │ Prompt: Implement JWT-based auth endpoint               │   │
│  │ WorkDir: /path/to/my-api                                │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │  vibe run       │  ← Execute with Claude Code
                     └─────────────────┘
                              │
                              ▼
                     Update results to Project
```

## Installation

```bash
go install github.com/tkc/vibe-project/cmd/vibe@latest
```

Or

```bash
git clone https://github.com/tkc/vibe-project.git
cd vibe-project
make build
```

## Prerequisites

- Go 1.21+
- [Claude Code](https://claude.ai/code) installed
- GitHub Personal Access Token (`project`, `repo` scopes)

## Setup

### 1. GitHub Authentication

```bash
vibe auth login
```

Required scopes:

- `project` (read/write)
- `read:org` (for organization projects)
- `repo` (required for commenting on Issues)

### 2. Project Selection

You can configure the project in two ways:

#### Option A: Using CLI commands

```bash
# List projects
vibe project list <owner>

# Select a project
vibe project select <owner> <project-number>
```

#### Option B: Using configuration file (Recommended)

Create a `.vibe.yaml` file in your project root:

```yaml
project:
  # Specify the GitHub Project URL
  url: https://github.com/users/tkc/projects/6
```

Supported URL formats:
- User projects: `https://github.com/users/{owner}/projects/{number}`
- Organization projects: `https://github.com/orgs/{owner}/projects/{number}`

### 3. GitHub Project Custom Fields

Add the following fields to your GitHub Project:

| Field Name  | Type          | Description                           |
| ----------- | ------------- | ------------------------------------- |
| Status      | Single Select | `Ready`, `In progress`, `In review`   |
| Result      | Text          | Execution result summary (auto-updated) |
| SessionID   | Text          | Session ID (auto-updated)             |
| ExecutedAt  | Date          | Execution timestamp (auto-updated)    |

**About Prompts:**
Prompts are not stored in GitHub Project fields. Instead, they are automatically loaded from the **Issue body and comments**.
When executing a task, all comments from the associated Issue are combined and passed to Claude Code.

## Usage

### List Tasks

```bash
vibe task list
vibe task list --status Ready
```

### Show Task Details

```bash
vibe task show <task-id>
```

### Execute Tasks

```bash
# Execute a single task
vibe run <task-id>

# Dry run (preview without executing)
vibe run <task-id> --dry-run

# Execute all Ready tasks
vibe run --all

# Resume a session
vibe run <task-id> --resume <session-id>
```

### Watch Mode

```bash
# Watch and auto-execute Ready tasks
vibe watch

# Watch with 1-minute interval
vibe watch --interval 1m
```

## Command Reference

```
vibe auth login      # GitHub authentication
vibe auth status     # Check authentication status
vibe auth logout     # Logout

vibe project list    # List projects
vibe project select  # Select a project
vibe project show    # Show current project

vibe task list       # List tasks
vibe task show       # Show task details

vibe run             # Execute task
vibe watch           # Watch mode
```

## Configuration Files

vibe supports two types of configuration files:

### 1. Global Configuration (JSON)

Global configuration is stored at `~/.vibe/config.json`:

```json
{
  "github_token": "ghp_xxx",
  "project_owner": "tkc",
  "project_number": 1,
  "claude_path": "claude"
}
```

This configuration is automatically created and updated by `vibe auth login` and `vibe project select` commands.

### 2. Project Local Configuration (YAML)

Place a `.vibe.yaml` file in your project root to manage project-specific settings:

```yaml
# .vibe.yaml
project:
  # Recommended: Use URL
  url: https://github.com/users/tkc/projects/6

  # Alternative: Specify owner and number directly
  # owner: tkc
  # number: 6

# Optional
claude_path: /usr/local/bin/claude
```

**Security Note:**
Do not include GitHub tokens in `.vibe.yaml`. Store them in the global configuration (`~/.vibe/config.json`).

### Configuration Precedence

Configuration is loaded with the following precedence:

1. **Local `.vibe.yaml`** (highest priority)
   - Project-specific settings
   - Searches from current directory up to parent directories
2. **Global `~/.vibe/config.json`**
   - GitHub token
   - Default project settings

Values specified in local configuration override global configuration (except for GitHub token).

### Sample File

Create a `.vibe.yaml` based on [.vibe.yaml.example](.vibe.yaml.example):

```bash
cp .vibe.yaml.example .vibe.yaml
# Edit .vibe.yaml
```

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint
```

## License

MIT
