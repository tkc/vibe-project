# Code Understanding Summary / コード理解のまとめ

## プロジェクト概要 (Project Overview)

**Vibe-Project** は、GitHub Projects V2 と Claude Code を統合する CLI ツールです。

### 主な機能 (Main Features)
1. ✅ GitHub Project のタスクを自動取得
2. ✅ Claude Code で AI 駆動のタスク実行
3. ✅ 実行結果を GitHub Project に自動更新
4. ✅ Issue にコメントとして結果を記録
5. ✅ Watch モードで継続的な自動実行

## 技術スタック (Technology Stack)

- **言語**: Go 1.24+
- **CLI フレームワーク**: Cobra
- **GitHub 統合**: GraphQL API (shurcooL/githubv4)
- **設定管理**: YAML + JSON
- **AI 実行**: Claude Code CLI

## アーキテクチャの特徴 (Architecture Highlights)

### 1. レイヤー構造 (Layered Architecture)
```
CLI Layer (cobra commands)
   ↓
Domain Layer (task, execution models)
   ↓
Integration Layer (github, claude, config)
```

### 2. 主要コンポーネント (Key Components)

#### TaskService (`internal/github/task.go`)
- GitHub Project の CRUD 操作
- フィールドメタデータのキャッシング
- Issue コメント管理

#### Executor (`internal/claude/executor.go`)
- Claude Code CLI のラッパー
- セッション管理 (resume 機能)
- 実行結果の解析

#### Config (`internal/config/config.go`)
- 2 段階設定 (グローバル + ローカル)
- Project URL 解析
- 設定の優先順位管理

### 3. ワークフロー (Workflow)

```
Ready Task → Load Prompt from Issue
           ↓
      Execute with Claude
           ↓
  Update Project (InReview)
           ↓
    Comment on Issue
```

## コードの特徴 (Code Characteristics)

### 強み (Strengths)
✅ **明確な責任分離**: 各パッケージが単一の責任を持つ  
✅ **効率的な GitHub 統合**: GraphQL + メタデータキャッシュ  
✅ **柔軟な設定管理**: グローバル + プロジェクトローカル  
✅ **堅牢なエラーハンドリング**: 失敗時も状態を保存  
✅ **拡張性**: 新機能の追加が容易な設計  

### 設計上の工夫 (Design Decisions)
- **プロンプトの動的読み込み**: Issue コメントから都度読み込み（柔軟性向上）
- **コメントマーカー**: HTML コメントで Vibe 生成コメントを識別
- **ステータス管理**: 成功/失敗に関わらず InReview に設定（人間のレビュー促進）
- **セッション永続化**: SessionID を保存して中断・再開可能

## ファイル構成の理解 (File Structure Understanding)

### エントリーポイント
- `cmd/vibe/main.go`: 最小限のエントリーポイント

### CLI コマンド (`internal/cli/`)
- `root.go`: 初期化とルートコマンド
- `auth.go`: 認証管理 (login/logout/status)
- `project.go`: プロジェクト管理
- `task.go`: タスク一覧・詳細
- `run.go`: タスク実行ロジック ⭐ 最重要
- `watch.go`: 自動監視・実行モード

### ドメインモデル (`internal/domain/`)
- `task.go`: Task 構造体、ステータス定数
- `execution.go`: ExecutionResult 構造体

### 統合層 (`internal/github/`, `internal/claude/`)
- `github/client.go`: GraphQL クライアント
- `github/task.go`: TaskService ⭐ 最重要
- `claude/executor.go`: Claude CLI ラッパー ⭐ 最重要

### 設定管理 (`internal/config/`)
- `config.go`: 設定読み込み・優先順位管理

## 開発の理解ポイント (Key Points for Development)

### 1. タスク実行の流れ
```go
// internal/cli/run.go
1. TaskService.GetTaskByID()       // タスク取得
2. TaskService.LoadPromptFromIssue() // プロンプト読み込み
3. TaskService.UpdateTaskStatus()   // Status → InProgress
4. Executor.Execute()               // Claude 実行
5. TaskService.UpdateTaskResult()   // 結果更新
6. TaskService.AddResultComment()   // コメント追加
```

### 2. GitHub Project との統合
```go
// internal/github/task.go
- GraphQL query でタスク一覧取得
- Mutation でフィールド更新（キャッシュ活用）
- Issue API でコメント操作
```

### 3. 設定の読み込み順序
```go
// internal/config/config.go
1. グローバル設定読み込み (~/.vibe/config.json)
2. ローカル設定検索 (.vibe.yaml)
3. ローカル設定でオーバーライド
4. 必須フィールドの検証
```

## テスト戦略 (Testing Strategy)

現在テストファイルは存在しないが、テストを追加する場合の推奨事項:

### 単体テスト候補
- `config.LoadWithPrecedence()` - 設定読み込みロジック
- `config.parseProjectURL()` - URL 解析
- `domain.Task.IsExecutable()` - 実行可能性判定
- `domain.ExecutionResult.Summary()` - サマリー生成

### 統合テスト候補
- GitHub API のモック化
- Claude CLI のモック化
- エンドツーエンドのタスク実行フロー

## 追加ドキュメント (Additional Documentation)

- [ARCHITECTURE.md](./ARCHITECTURE.md) - 詳細なアーキテクチャドキュメント
- [docs/README.md](./docs/README.md) - コードナビゲーションガイド
- [README.md](./README.md) - ユーザーガイド

## まとめ (Summary)

Vibe-Project は、以下の特徴を持つ、よく設計された Go アプリケーションです：

1. **明確なアーキテクチャ**: レイヤー分離と責任の明確化
2. **実用的な設計**: GitHub Projects と Claude Code のシームレスな統合
3. **保守性**: 各コンポーネントが独立しており、変更が容易
4. **拡張性**: 新機能の追加や他の AI ツールへの対応が可能

このコードベースは、CLI ツール開発のベストプラクティスを多く含んでおり、学習価値の高いプロジェクトです。
