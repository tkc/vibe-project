# Vibe-Project Architecture Documentation

## 概要 (Overview)

Vibe-Projectは、GitHub Projects V2とClaude Codeを統合するGoベースのCLIツールです。GitHub Projectのタスクを自動的に取得し、Claude Codeで実行し、結果をプロジェクトに反映します。

Vibe-Project is a Go-based CLI tool that integrates GitHub Projects V2 with Claude Code. It automatically fetches tasks from GitHub Projects, executes them with Claude Code, and updates the results back to the project.

## アーキテクチャ (Architecture)

### システム全体の構成 (System Overview)

```
┌─────────────────────────────────────────────────────────────────┐
│                      GitHub Project V2                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ Task: タスクの説明                                       │   │
│  │ Status: Ready → InProgress → InReview                   │   │
│  │ IssueURL: https://github.com/owner/repo/issues/1        │   │
│  │ Result: 実行結果のサマリー                               │   │
│  │ SessionID: claude_session_xxx                           │   │
│  │ ExecutedAt: 2024-01-01                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │   Vibe CLI Tool      │
                    │  ┌────────────────┐  │
                    │  │ Config Layer   │  │ ← .vibe.yaml + ~/.vibe/config.json
                    │  └────────────────┘  │
                    │  ┌────────────────┐  │
                    │  │ GitHub Client  │  │ ← GraphQL API (shurcooL/githubv4)
                    │  └────────────────┘  │
                    │  ┌────────────────┐  │
                    │  │ Claude Executor│  │ ← CLI wrapper (`claude --print`)
                    │  └────────────────┘  │
                    │  ┌────────────────┐  │
                    │  │ Task Service   │  │ ← Business Logic
                    │  └────────────────┘  │
                    └──────────────────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │   Claude Code CLI    │ ← AI-powered task execution
                    └──────────────────────┘
```

### パッケージ構成 (Package Structure)

```
vibe-project/
├── cmd/vibe/
│   └── main.go                 # エントリーポイント (Entry point)
├── internal/
│   ├── cli/                    # CLIコマンド (CLI commands - Cobra-based)
│   │   ├── root.go            # ルートコマンドと初期化
│   │   ├── auth.go            # 認証 (login/logout/status)
│   │   ├── project.go         # プロジェクト選択とリスト表示
│   │   ├── task.go            # タスクリストと詳細表示
│   │   ├── run.go             # 単一タスク実行
│   │   ├── watch.go           # 自動監視・実行モード
│   │   └── status.go          # ステータス表示
│   ├── config/                 # 設定管理 (Configuration management)
│   │   └── config.go          # YAML + JSON設定、ファイルパス管理
│   ├── domain/                 # ドメインモデル (Domain models)
│   │   ├── task.go            # Taskモデルとステータス定数
│   │   └── execution.go       # 実行結果モデル
│   ├── github/                 # GitHub API統合 (GitHub API integration)
│   │   ├── client.go          # GraphQLクライアントラッパー
│   │   └── task.go            # TaskService (CRUD + コメント管理)
│   ├── claude/                 # Claude Code統合 (Claude Code integration)
│   │   └── executor.go        # Claude CLIラッパー
│   └── notify/                 # システム通知 (System notifications)
│       ├── notify_darwin.go   # macOS通知
│       └── notify_other.go    # フォールバック (no-op)
└── go.mod                      # Go依存関係
```

## 主要コンポーネント (Key Components)

### 1. CLI Layer (`internal/cli/`)

#### `root.go` - ルートコマンド
- Cobraフレームワークの初期化
- 設定ファイルの読み込み (precedence: .vibe.yaml → ~/.vibe/config.json)
- GitHub/Claudeクライアントの初期化
- グローバルフラグの管理

#### `auth.go` - 認証管理
```go
// コマンド
vibe auth login   // GitHub Personal Access Token入力・保存
vibe auth status  // 認証状態の確認
vibe auth logout  // トークンの削除
```

#### `project.go` - プロジェクト管理
```go
// コマンド
vibe project list <owner>            // プロジェクト一覧表示
vibe project select <owner> <number> // プロジェクト選択・保存
vibe project show                    // 現在のプロジェクト表示
```

#### `task.go` - タスク管理
```go
// コマンド
vibe task list             // タスク一覧表示
vibe task list --status Ready  // ステータスフィルタ
vibe task show <task-id>   // タスク詳細表示
```

#### `run.go` - タスク実行
- **単一タスク実行**: 指定されたタスクIDを実行
- **全タスク実行**: `--all`フラグで全Readyタスクを実行
- **セッション再開**: `--resume`フラグで既存セッション継続
- **ドライラン**: `--dry-run`で実行内容プレビュー

#### `watch.go` - 監視モード
- 定期的にReadyタスクをチェック (デフォルト: 5分)
- 新規Readyタスクを自動実行
- macOS通知で結果を通知

### 2. Domain Layer (`internal/domain/`)

#### `task.go` - タスクモデル
```go
type Task struct {
    ID          string    // ProjectV2ItemのID
    Title       string    // タスクタイトル
    Status      string    // Ready, InProgress, InReview
    Prompt      string    // 実行時にIssueコメントから読み込み
    Result      string    // 実行結果のサマリー
    SessionID   string    // Claude Code セッションID
    ExecutedAt  time.Time // 実行日時
    IssueURL    string    // 関連するGitHub IssueのURL
    IssueNumber int       // Issueの番号
}

// ステータス遷移
const (
    StatusReady      = "Ready"       // 実行待ち
    StatusInProgress = "In progress" // 実行中
    StatusInReview   = "In review"   // レビュー待ち
)

// 実行可能性チェック
func (t *Task) IsExecutable() bool {
    return t.Status == StatusReady && t.IssueURL != ""
}
```

#### `execution.go` - 実行結果モデル
```go
type ExecutionResult struct {
    SessionID string        // Claudeセッションの識別子
    Output    string        // 実行結果の全出力
    Error     error         // エラー情報
    StartedAt time.Time     // 開始時刻
    EndedAt   time.Time     // 終了時刻
    Duration  time.Duration // 実行時間
}

// 結果のサマリー化 (最大500文字に短縮)
func (r *ExecutionResult) Summary() string

// 次のステータスを決定 (成功/失敗に関わらず InReview)
func (r *ExecutionResult) NewStatus() string
```

### 3. GitHub Integration (`internal/github/`)

#### `client.go` - GitHubクライアント
```go
type Client struct {
    client *githubv4.Client // GraphQLクライアント
    token  string           // Personal Access Token
}

// プロジェクト一覧取得 (UserまたはOrganization)
func (c *Client) ListUserProjects(owner string) ([]Project, error)
func (c *Client) ListOrgProjects(org string) ([]Project, error)

// Issueコメント操作
func (c *Client) GetIssueComments(owner, repo string, number int) ([]Comment, error)
func (c *Client) AddIssueComment(owner, repo string, number int, body string) error
```

#### `task.go` - TaskService
```go
type TaskService struct {
    client      *Client
    owner       string // プロジェクトオーナー
    number      int    // プロジェクト番号
    fieldCache  map[string]string // フィールド名→IDのキャッシュ
    optionCache map[string]map[string]string // フィールドID→オプション名→IDのキャッシュ
}

// タスク取得
func (s *TaskService) GetTasks(statusFilter string) ([]domain.Task, error)
func (s *TaskService) GetTaskByID(id string) (*domain.Task, error)

// プロンプト読み込み (Issueボディ + コメント統合)
func (s *TaskService) LoadPromptFromIssue(task *domain.Task) error

// タスク更新
func (s *TaskService) UpdateTaskStatus(taskID, status string) error
func (s *TaskService) UpdateTaskResult(task *domain.Task, result *domain.ExecutionResult) error

// Issueコメント追加 (vibe-project-commentマーカー付き)
func (s *TaskService) AddResultComment(task *domain.Task, result *domain.ExecutionResult) error
```

**重要な設計ポイント:**
- **フィールドメタデータキャッシュ**: 初回クエリでStatus選択肢やフィールドIDをキャッシュし、以降の更新で再利用
- **プロンプトの動的読み込み**: タスク実行時に常にIssueコメントから最新のプロンプトを読み込む
- **コメントマーカー**: `<!-- vibe-project-comment -->`でVibeが生成したコメントを識別し、無限ループを防止

### 4. Claude Integration (`internal/claude/`)

#### `executor.go` - Claude CLIラッパー
```go
type Executor struct {
    claudePath string // Claudeコマンドのパス
}

// タスク実行
func (e *Executor) Execute(ctx context.Context, prompt string, sessionID string) (*domain.ExecutionResult, error)

// コマンド構築例
// $ claude --print --resume <sessionID> "<prompt>"
```

**実行フロー:**
1. `claude`コマンドを`--print`フラグで実行 (JSON出力)
2. `--resume`フラグでセッションID指定 (2回目以降の実行)
3. 標準出力/標準エラー出力をキャプチャ
4. JSON出力から新しいSessionIDを抽出
5. タイムアウト: 30分 (context.WithTimeout)

### 5. Configuration (`internal/config/`)

#### 設定ファイルの優先順位 (Precedence)
```
1. ローカル設定 (.vibe.yaml) ← 最優先
   ├─ プロジェクト固有の設定
   └─ カレントディレクトリから親ディレクトリへ検索

2. グローバル設定 (~/.vibe/config.json)
   ├─ GitHubトークン
   └─ デフォルトプロジェクト設定
```

#### `.vibe.yaml` (プロジェクトローカル)
```yaml
project:
  # 推奨: URLで指定
  url: https://github.com/users/tkc/projects/6
  
  # または: owner/numberで直接指定
  # owner: tkc
  # number: 6

# オプション
claude_path: /usr/local/bin/claude
```

#### `~/.vibe/config.json` (グローバル)
```json
{
  "github_token": "ghp_xxx",
  "project_owner": "tkc",
  "project_number": 6,
  "claude_path": "claude"
}
```

#### URL解析機能
```go
// サポートされるURL形式
// - https://github.com/users/{owner}/projects/{number}
// - https://github.com/orgs/{owner}/projects/{number}

func parseProjectURL(url string) (owner string, number int, err error)
```

## ワークフロー詳細 (Detailed Workflows)

### A. タスク実行フロー (`vibe run <task-id>`)

```
1. タスク取得 (Get Task)
   ├─ GraphQLでProjectV2から取得
   └─ フィールド: Status, Prompt, SessionID, Result, ExecutedAt

2. プロンプト読み込み (Load Prompt)
   ├─ IssueURLからIssue番号を抽出
   ├─ Issueボディを取得
   ├─ 全コメントを取得 (vibe生成コメント除く)
   └─ "Issue Body:\n{body}\n\nComments:\n{comments}" に結合

3. 実行可能性チェック (Verify Executability)
   └─ Status=="Ready" && IssueURL != "" ?

4. ステータス更新: Ready → InProgress
   └─ GraphQL mutation でStatusフィールド更新

5. Claude Code実行 (Execute with Claude)
   ├─ コマンド: claude --print --resume {SessionID?} "{Prompt}"
   ├─ stdout/stderrキャプチャ
   ├─ JSON出力からSessionID抽出
   └─ タイムアウト: 30分

6. プロジェクト更新 (Update Project)
   ├─ Status → InReview (成功/失敗問わず)
   ├─ Result → 実行結果サマリー (最大500文字)
   ├─ SessionID → 抽出したセッションID
   └─ ExecutedAt → 現在時刻

7. Issueコメント追加 (Comment on Issue)
   └─ HTMLコメント <!-- vibe-project-comment --> 付きで実行状態を記録
```

### B. 監視モード (`vibe watch`)

```
無限ループ (Polling Loop)
  ├─ 指定間隔でスリープ (デフォルト: 5分)
  ├─ 全Readyタスクを取得
  ├─ 実行可能タスクのフィルタ (IssueURL存在チェック)
  ├─ 各タスクに対してフローA (1-7) を実行
  ├─ 成功/失敗をログ出力
  ├─ macOS通知を送信 (成功: ✅, 失敗: ❌)
  └─ 繰り返し
```

### C. プロジェクト選択フロー

```
1. vibe project list <owner>
   ├─ UserかOrgかを自動判定
   ├─ GraphQLでプロジェクト一覧取得
   └─ タイトル、番号、URL表示

2. vibe project select <owner> <number>
   ├─ owner/numberを検証
   ├─ ~/.vibe/config.jsonに保存
   └─ 成功メッセージ表示
```

### D. 認証フロー

```
1. vibe auth login
   ├─ Personal Access Tokenを入力 (stdin)
   ├─ トークン検証 (viewer query)
   ├─ ~/.vibe/config.jsonに保存
   └─ ユーザー名表示

2. vibe auth status
   ├─ トークン存在チェック
   ├─ viewer query でユーザー情報取得
   └─ 認証状態とユーザー名表示

3. vibe auth logout
   ├─ ~/.vibe/config.jsonからトークン削除
   └─ 成功メッセージ
```

## 設計パターンと技術的決定事項 (Design Patterns & Technical Decisions)

### 1. サービスレイヤーパターン (Service Layer Pattern)
- **TaskService**がGitHub Project操作を全てカプセル化
- フィールドメタデータをキャッシュし、効率的なmutationを実現
- 単一責任: タスクのCRUD + GitHub関連操作

### 2. 関心の分離 (Separation of Concerns)
```
Domain Layer      → 純粋なモデル (Task, Execution, Status)
Integration Layer → GitHub, Claude, Config
CLI Layer         → オーケストレーションとユーザー対話
```

### 3. 設定の優先順位 (Configuration Precedence)
```
Global Config + Local Config Override
  ↓
Project URL解析 → owner/number OR 直接指定
  ↓
ClaudePath: デフォルト "claude" または明示指定
```

### 4. タスク実行可能性モデル (Task Executability Model)
```
実行可能条件:
  ✓ Status == "Ready"
  ✓ IssueURL != ""

プロンプトソース:
  × ProjectフィールドではなくIssueコメントから動的読み込み
  → タスク指示を柔軟に更新可能
```

### 5. 冪等なGitHub更新 (Idempotent Updates)
- MutationでフィールドIDとオプションIDを使用 (名前ではない)
- SessionIDを永続化し、`claude --resume`で継続可能
- コメントにHTMLマーカーで再帰防止

### 6. エラーハンドリング (Error Handling)
```
実行失敗時:
  × エラーをthrowしない
  ✓ "InReview"ステータスに設定し、人間のレビュー待ち

部分的失敗:
  ✓ ログ記録するがワークフロー継続
  ✓ macOS通知は成功/失敗に関わらず送信
```

### 7. 拡張性フック (Extensibility Hooks)
- カスタム`claude_path`サポート (非標準インストール対応)
- 実行タイムアウト設定可能
- Statusフィールドでのタスクフィルタリング
- Watch間隔のカスタマイズ

## データモデル詳細 (Data Models)

### タスクステートマシン (Task State Machine)
```
Ready → InProgress → InReview
  ↑                      |
  └──────────────────────┘
     (手動でReadyに戻す)
```

### GitHub Projectフィールド (Project Fields)
```
必須フィールド:
- Status (Single Select): Ready, In progress, In review
- Result (Text): 実行結果のサマリー
- SessionID (Text): claude --resumeで使用
- ExecutedAt (Date): 実行日時

オプション:
- Prompt (Text): プロジェクトフィールドとして保存可能だが、
                 実行時は常にIssueコメントから読み込む
```

### 実行ライフサイクル (Execution Lifecycle)
```
StartedAt (time.Time)
  ↓
Execute (claude CLI)
  ↓
EndedAt (time.Time) / Duration (time.Duration)
  ↓
Success/Error → Summary (500文字) → Project Update → Issue Comment
```

## 外部依存関係 (External Dependencies)

| パッケージ | 用途 |
|-----------|------|
| `github.com/spf13/cobra` | CLIフレームワーク (コマンド、フラグ、ヘルプ) |
| `github.com/shurcooL/githubv4` | GitHub GraphQL APIクライアント |
| `golang.org/x/oauth2` | OAuth2トークン処理 |
| `gopkg.in/yaml.v3` | YAML設定ファイル解析 |

## セキュリティ考慮事項 (Security Considerations)

1. **トークン管理**
   - Personal Access Tokenは`~/.vibe/config.json`に保存
   - `.vibe.yaml`にはトークンを含めない (リポジトリコミット防止)
   - 必要スコープ: `project`, `read:org`, `repo`

2. **コマンドインジェクション対策**
   - Claudeコマンド実行時、プロンプトを適切にエスケープ
   - `exec.Command`の引数を配列で渡す (シェルインジェクション防止)

3. **タイムアウト**
   - Claude実行に30分のタイムアウト設定
   - 無限ループ防止

## 今後の拡張可能性 (Future Extensibility)

1. **複数AI実行環境サポート**
   - Claudeだけでなく、他のAIツールにも対応可能
   - Executor interfaceの導入

2. **並列実行**
   - 複数Readyタスクの並列実行
   - Goroutineとチャネルで実装可能

3. **Webhook統合**
   - GitHub Webhookでタスク作成時に自動実行
   - Watch modeの代替

4. **詳細ログ**
   - 構造化ログ (JSON)
   - ログレベル設定

5. **メトリクス**
   - 実行時間、成功率などの統計
   - Prometheus形式でエクスポート

## まとめ (Summary)

Vibe-Projectは、シンプルかつ堅牢なアーキテクチャで以下を実現しています：

✅ **明確な責任分離**: Domain, Integration, CLIの3層構造  
✅ **柔軟な設定管理**: グローバル + プロジェクトローカルの2段階  
✅ **効率的なGitHub統合**: GraphQL + メタデータキャッシュ  
✅ **簡潔なエラーハンドリング**: 失敗時も状態を保存し、人間のレビューを促す  
✅ **拡張性**: 新しいAIツールやワークフローへの対応が容易  

このアーキテクチャにより、開発タスクの自動化とGitHub Projectsの統合が効率的に行えます。
