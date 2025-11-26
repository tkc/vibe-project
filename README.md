# vibe-project

GitHub Projects と Claude Code を連携させる CLI ツール

## 概要

GitHub Project V2 からタスクを取得し、Claude Code で実行して結果をプロジェクトに反映します。

```
┌─────────────────────────────────────────────────────────────────┐
│                      GitHub Project V2                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Task: ユーザー認証APIの実装                              │   │
│  │ Status: Ready → InProgress → InReview                    │   │
│  │ Prompt: JWTを使った認証エンドポイントを実装して          │   │
│  │ WorkDir: /path/to/my-api                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │  vibe run       │  ← Claude Code 実行
                     └─────────────────┘
                              │
                              ▼
                     結果を Project に反映
```

## インストール

```bash
go install github.com/tkc/vibe-project/cmd/vibe@latest
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
- GitHub Personal Access Token（`project`, `repo` スコープ）

## セットアップ

### 1. GitHub 認証

```bash
vibe auth login
```

必要なスコープ:

- `project` (read/write)
- `read:org` (組織プロジェクトの場合)
- `repo` (Issue へのコメントに必要)

### 2. プロジェクト選択

```bash
# プロジェクト一覧を表示
vibe project list <owner>

# プロジェクトを選択
vibe project select <owner> <project-number>
```

### 3. GitHub Project のカスタムフィールド設定

以下のフィールドを Project に追加してください:

| フィールド名 | 型            | 説明                                |
| ------------ | ------------- | ----------------------------------- |
| Status       | Single Select | `Ready`, `In progress`, `In review` |
| Result       | Text          | 実行結果サマリー（自動更新）        |
| SessionID    | Text          | セッション ID（自動更新）           |
| ExecutedAt   | Date          | 実行日時（自動更新）                |

**プロンプトについて:**
プロンプトは GitHub Project のフィールドではなく、**Issue の本文とコメント**から自動的に読み込まれます。
タスク実行時に、関連する Issue の全てのコメントが結合されて Claude Code に渡されます。

## 使い方

### タスク一覧

```bash
vibe task list
vibe task list --status Ready
```

### タスク詳細

```bash
vibe task show <task-id>
```

### タスク実行

```bash
# 単一タスク実行
vibe run <task-id>

# ドライラン（実行せず確認のみ）
vibe run <task-id> --dry-run

# 全 Ready タスクを実行
vibe run --all

# セッション継続
vibe run <task-id> --resume <session-id>
```

### 監視モード

```bash
# Ready タスクを監視して自動実行
vibe watch

# 1分間隔で監視
vibe watch --interval 1m
```

## コマンド一覧

```
vibe auth login      # GitHub 認証
vibe auth status     # 認証状態確認
vibe auth logout     # ログアウト

vibe project list    # プロジェクト一覧
vibe project select  # プロジェクト選択
vibe project show    # 現在のプロジェクト表示

vibe task list       # タスク一覧
vibe task show       # タスク詳細

vibe run             # タスク実行
vibe watch           # 監視モード
```

## 設定ファイル

設定は `~/.vibe/config.json` に保存されます:

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
