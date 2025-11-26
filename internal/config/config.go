package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config はアプリケーション設定
type Config struct {
	GitHubToken   string `json:"github_token"`
	ProjectOwner  string `json:"project_owner"`  // org or user
	ProjectNumber int    `json:"project_number"` // project number
	ClaudePath    string `json:"claude_path"`    // claude コマンドのパス
}

// DefaultClaudePath はデフォルトのclaudeコマンドパス
const DefaultClaudePath = "claude"

// configFileName は設定ファイル名
const configFileName = "config.json"

// configDir は設定ディレクトリ名
const configDirName = ".vive"

// Load は設定ファイルを読み込む
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
		return fmt.Errorf("github_token is required. Run: vive auth login")
	}
	if c.ProjectOwner == "" || c.ProjectNumber == 0 {
		return fmt.Errorf("project is not configured. Run: vive project select")
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
