package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config はアプリケーション設定
type Config struct {
	GitHubToken   string `json:"github_token" yaml:"github_token"`
	ProjectOwner  string `json:"project_owner" yaml:"project_owner"`   // org or user
	ProjectNumber int    `json:"project_number" yaml:"project_number"` // project number
	ClaudePath    string `json:"claude_path" yaml:"claude_path"`       // claude コマンドのパス
}

// ProjectConfig はYAMLファイル用のプロジェクト設定
type ProjectConfig struct {
	Project struct {
		URL    string `yaml:"url"`    // GitHub Project URL (優先)
		Owner  string `yaml:"owner"`  // 後方互換性のため残す
		Number int    `yaml:"number"` // 後方互換性のため残す
	} `yaml:"project"`
	ClaudePath string `yaml:"claude_path,omitempty"`
}

// DefaultClaudePath はデフォルトのclaudeコマンドパス
const DefaultClaudePath = "claude"

// configFileName は設定ファイル名
const configFileName = "config.json"

// configDirName は設定ディレクトリ名
const configDirName = ".vibe"

// yamlConfigFileName はYAML設定ファイル名
const yamlConfigFileName = ".vibe.yaml"

// Load は設定ファイルを読み込む（後方互換性のため残す）
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{ClaudePath: DefaultClaudePath}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.ClaudePath == "" {
		cfg.ClaudePath = DefaultClaudePath
	}

	return &cfg, nil
}

// LoadWithPrecedence は優先順位に従って設定を読み込む
// 1. カレントディレクトリの .vibe.yaml (最優先)
// 2. グローバル設定 ~/.vibe/config.json
func LoadWithPrecedence() (*Config, error) {
	// まずグローバル設定を読み込む
	globalCfg, err := Load()
	if err != nil {
		return nil, err
	}

	// プロジェクトローカルの設定を探す
	yamlPath, err := findProjectConfig()
	if err != nil {
		// YAML設定が見つからない場合はグローバル設定のみを使用
		return globalCfg, nil
	}

	// YAML設定を読み込んでマージ
	localCfg, err := loadYAML(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML config from %s: %w", yamlPath, err)
	}

	// ローカル設定で上書き（GitHubトークンはグローバル設定を優先）
	merged := globalCfg
	if localCfg.ProjectOwner != "" {
		merged.ProjectOwner = localCfg.ProjectOwner
	}
	if localCfg.ProjectNumber != 0 {
		merged.ProjectNumber = localCfg.ProjectNumber
	}
	if localCfg.ClaudePath != "" {
		merged.ClaudePath = localCfg.ClaudePath
	}

	return merged, nil
}

// loadYAML はYAMLファイルから設定を読み込む
func loadYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML config: %w", err)
	}

	var projectCfg ProjectConfig
	if err := yaml.Unmarshal(data, &projectCfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	var owner string
	var number int

	// URLが指定されている場合はURLからパース（優先）
	if projectCfg.Project.URL != "" {
		owner, number, err = ParseProjectURL(projectCfg.Project.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse project URL: %w", err)
		}
	} else {
		// 後方互換性: owner/number が直接指定されている場合
		owner = projectCfg.Project.Owner
		number = projectCfg.Project.Number
	}

	cfg := &Config{
		ProjectOwner:  owner,
		ProjectNumber: number,
		ClaudePath:    projectCfg.ClaudePath,
	}

	if cfg.ClaudePath == "" {
		cfg.ClaudePath = DefaultClaudePath
	}

	return cfg, nil
}

// ParseProjectURL はGitHub Project URLからowner と project number を抽出する
// 対応形式:
//   - https://github.com/users/{owner}/projects/{number}
//   - https://github.com/users/{owner}/projects/{number}/views/{view}
//   - https://github.com/orgs/{owner}/projects/{number}
//   - https://github.com/orgs/{owner}/projects/{number}/views/{view}
func ParseProjectURL(url string) (owner string, number int, err error) {
	// ユーザープロジェクト: https://github.com/users/{owner}/projects/{number}...
	userPattern := regexp.MustCompile(`^https://github\.com/users/([^/]+)/projects/(\d+)`)
	// 組織プロジェクト: https://github.com/orgs/{owner}/projects/{number}...
	orgPattern := regexp.MustCompile(`^https://github\.com/orgs/([^/]+)/projects/(\d+)`)

	if matches := userPattern.FindStringSubmatch(url); len(matches) == 3 {
		owner = matches[1]
		number, _ = strconv.Atoi(matches[2])
		return owner, number, nil
	}

	if matches := orgPattern.FindStringSubmatch(url); len(matches) == 3 {
		owner = matches[1]
		number, _ = strconv.Atoi(matches[2])
		return owner, number, nil
	}

	return "", 0, fmt.Errorf("invalid GitHub Project URL format: %s", url)
}

// findProjectConfig はカレントディレクトリから上位ディレクトリへ .vibe.yaml を探索
func findProjectConfig() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	dir := cwd
	for {
		yamlPath := filepath.Join(dir, yamlConfigFileName)
		if _, err := os.Stat(yamlPath); err == nil {
			return yamlPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// ルートディレクトリに到達
			return "", fmt.Errorf("no .vibe.yaml found")
		}
		dir = parent
	}
}

// Save は設定ファイルを保存する
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	// ディレクトリ作成
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Validate は設定が有効かどうかを検証する
func (c *Config) Validate() error {
	if c.GitHubToken == "" {
		return fmt.Errorf("github_token is required. Run: vibe auth login")
	}
	if c.ProjectOwner == "" || c.ProjectNumber == 0 {
		return fmt.Errorf("project is not configured. Run: vibe project select")
	}
	return nil
}

// IsConfigured はプロジェクトが設定済みかどうかを返す
func (c *Config) IsConfigured() bool {
	return c.GitHubToken != "" && c.ProjectOwner != "" && c.ProjectNumber > 0
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home dir: %w", err)
	}
	return filepath.Join(home, configDirName, configFileName), nil
}
