# vibe-project

GitHub Projects と Claude Code を連携させる CLI ツール

## 概要

GitHub Project V2 からタスクを取得し、Claude Code で実行して結果をプロジェクトに反映します。

```
┌─────────────────────────────────────────────────────────────────┐
│                      GitHub Project V2                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Task: ユーザー認証APIの実装                              │   │
│  │ Status: Todo → InProgress → Done                        │   │
│  │ Prompt: JWTを使った認証エンドポイントを実装して          │   │
│  │ WorkDir: /path/to/my-api                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │  vive run       │  ← Claude Code 実行
                     └─────────────────┘
                              │
                              ▼
                     結果を Project に反映
```

## インストール

```bash
go install github.com/tkc/vibe-project/cmd/vive@latest
```

または

```bash
git clone https://github.com/tkc/vibe-project.git
cd vibe-project
make build
```

## 前提条件

- Go 1.21+
- [Claude Code](https://claude.ai/code) がインストール済み
- GitHub Personal Access Token（`project` スコープ）

## セットアップ

### 1. GitHub 認証

```bash
vive auth login
```

必要なスコープ:
- `project` (read/write)
- `read:org` (組織プロジェクトの場合)

### 2. プロジェクト選択

```bash
# プロジェクト一覧を表示
vive project list <owner>

# プロジェクトを選択
vive project select <owner> <project-number>
```

### 3. GitHub Project のカスタムフィールド設定

以下のフィールドを Project に追加してください:

| フィールド名 | 型 | 説明 |
|-------------|-----|------|
| Status | Single Select | `Todo`, `InProgress`, `Done`, `Failed` |
| Prompt | Text | Claude Code に渡すプロンプト |
| WorkDir | Text | 作業ディレクトリの絶対パス |
| Result | Text | 実行結果サマリー（自動更新） |
| SessionID | Text | セッションID（自動更新） |
| ExecutedAt | Date | 実行日時（自動更新） |

## 使い方

### タスク一覧

```bash
vive task list
vive task list --status Todo
```

### タスク詳細

```bash
vive task show <task-id>
```

### タスク実行

```bash
# 単一タスク実行
vive run <task-id>

# ドライラン（実行せず確認のみ）
vive run <task-id> --dry-run

# 全 Todo タスクを実行
vive run --all

# セッション継続
vive run <task-id> --resume <session-id>
```

### 監視モード

```bash
# Todo タスクを監視して自動実行
vive watch

# 1分間隔で監視
vive watch --interval 1m
```

## コマンド一覧

```
vive auth login      # GitHub 認証
vive auth status     # 認証状態確認
vive auth logout     # ログアウト

vive project list    # プロジェクト一覧
vive project select  # プロジェクト選択
vive project show    # 現在のプロジェクト表示

vive task list       # タスク一覧
vive task show       # タスク詳細

vive run             # タスク実行
vive watch           # 監視モード
```

## 設定ファイル

設定は `~/.vive/config.json` に保存されます:

```json
{
  "github_token": "ghp_xxx",
  "project_owner": "tkc",
  "project_number": 1,
  "claude_path": "claude"
}
```

## 開発

```bash
# ビルド
make build

# テスト
make test

# リント
make lint
```

## ライセンス

MIT
