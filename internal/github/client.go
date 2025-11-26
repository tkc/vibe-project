package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Client はGitHub GraphQL APIクライアント
type Client struct {
	gql   *githubv4.Client
	owner string
}

// NewClient は新しいClientを作成する
func NewClient(token, owner string) *Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return &Client{
		gql:   githubv4.NewClient(httpClient),
		owner: owner,
	}
}

// Project はGitHub Project V2の情報
type Project struct {
	ID     string
	Number int
	Title  string
	URL    string
}

// ProjectField はProjectのカスタムフィールド
type ProjectField struct {
	ID      string
	Name    string
	Options []FieldOption // Single Select用
}

// FieldOption はSingle Selectのオプション
type FieldOption struct {
	ID   string
	Name string
}

// GetProjects はユーザー/組織のProject一覧を取得する
func (c *Client) GetProjects(ctx context.Context) ([]Project, error) {
	// まずユーザーのプロジェクトを試す
	projects, userErr := c.getUserProjects(ctx)
	if userErr == nil {
		return projects, nil
	}

	// ユーザーで失敗したら組織のプロジェクトを試す
	projects, orgErr := c.getOrgProjects(ctx)
	if orgErr == nil {
		return projects, nil
	}

	// 両方失敗した場合
	if userErr != nil && orgErr != nil {
		// 権限エラーの場合は明確なメッセージを返す
		if strings.Contains(userErr.Error(), "not accessible by personal access token") {
			return nil, fmt.Errorf("token lacks 'project' scope. Please regenerate your token with the 'project' permission at https://github.com/settings/tokens")
		}
		return nil, fmt.Errorf("user: %v, org: %v", userErr, orgErr)
	}
	return nil, fmt.Errorf("unknown error")
}

func (c *Client) getUserProjects(ctx context.Context) ([]Project, error) {
	var query struct {
		User struct {
			ProjectsV2 struct {
				Nodes []struct {
					ID     string
					Number int
					Title  string
					URL    string `graphql:"url"`
				}
			} `graphql:"projectsV2(first: 20)"`
		} `graphql:"user(login: $owner)"`
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(c.owner),
	}

	if err := c.gql.Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	projects := make([]Project, 0, len(query.User.ProjectsV2.Nodes))
	for _, n := range query.User.ProjectsV2.Nodes {
		projects = append(projects, Project{
			ID:     n.ID,
			Number: n.Number,
			Title:  n.Title,
			URL:    n.URL,
		})
	}
	return projects, nil
}

func (c *Client) getOrgProjects(ctx context.Context) ([]Project, error) {
	var query struct {
		Organization struct {
			ProjectsV2 struct {
				Nodes []struct {
					ID     string
					Number int
					Title  string
					URL    string `graphql:"url"`
				}
			} `graphql:"projectsV2(first: 20)"`
		} `graphql:"organization(login: $owner)"`
	}

	variables := map[string]interface{}{
		"owner": githubv4.String(c.owner),
	}

	if err := c.gql.Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	projects := make([]Project, 0, len(query.Organization.ProjectsV2.Nodes))
	for _, n := range query.Organization.ProjectsV2.Nodes {
		projects = append(projects, Project{
			ID:     n.ID,
			Number: n.Number,
			Title:  n.Title,
			URL:    n.URL,
		})
	}
	return projects, nil
}

// GetProjectByNumber は指定番号のProjectを取得する
func (c *Client) GetProjectByNumber(ctx context.Context, number int) (*Project, error) {
	projects, err := c.GetProjects(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if p.Number == number {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("project #%d not found", number)
}

// AddIssueComment はIssueにコメントを追加する
func (c *Client) AddIssueComment(ctx context.Context, issueURL, body string) error {
	// Issue URLからIssue IDを取得
	issueID, err := c.getIssueID(ctx, issueURL)
	if err != nil {
		return fmt.Errorf("failed to get issue ID: %w", err)
	}

	var mutation struct {
		AddComment struct {
			CommentEdge struct {
				Node struct {
					ID string
				}
			}
		} `graphql:"addComment(input: $input)"`
	}

	input := githubv4.AddCommentInput{
		SubjectID: githubv4.ID(issueID),
		Body:      githubv4.String(body),
	}

	if err := c.gql.Mutate(ctx, &mutation, input, nil); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	return nil
}

// getIssueID はIssue URLからIssueのNode IDを取得する
func (c *Client) getIssueID(ctx context.Context, issueURL string) (string, error) {
	// URLからowner, repo, numberを抽出
	// 例: https://github.com/tkc/vibe-project/issues/1
	parts := strings.Split(issueURL, "/")
	if len(parts) < 7 {
		return "", fmt.Errorf("invalid issue URL: %s", issueURL)
	}

	owner := parts[3]
	repo := parts[4]
	numberStr := parts[6]

	var number int
	fmt.Sscanf(numberStr, "%d", &number)

	var query struct {
		Repository struct {
			Issue struct {
				ID string
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"repo":   githubv4.String(repo),
		"number": githubv4.Int(number),
	}

	if err := c.gql.Query(ctx, &query, variables); err != nil {
		return "", err
	}

	return query.Repository.Issue.ID, nil
}

// GetIssueComments はIssueのコメント一覧を取得する
func (c *Client) GetIssueComments(ctx context.Context, issueURL string) ([]string, error) {
	// URLからowner, repo, numberを抽出
	parts := strings.Split(issueURL, "/")
	if len(parts) < 7 {
		return nil, fmt.Errorf("invalid issue URL: %s", issueURL)
	}

	owner := parts[3]
	repo := parts[4]
	numberStr := parts[6]

	var number int
	fmt.Sscanf(numberStr, "%d", &number)

	var query struct {
		Repository struct {
			Issue struct {
				BodyText string
				Comments struct {
					Nodes []struct {
						BodyText string
						Author   struct {
							Login string
						}
					}
				} `graphql:"comments(first: 100)"`
			} `graphql:"issue(number: $number)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"repo":   githubv4.String(repo),
		"number": githubv4.Int(number),
	}

	if err := c.gql.Query(ctx, &query, variables); err != nil {
		return nil, err
	}

	// Issue本文 + 全コメントを結合（vibe project commentは除外）
	var comments []string
	if query.Repository.Issue.BodyText != "" {
		comments = append(comments, query.Repository.Issue.BodyText)
	}
	for _, comment := range query.Repository.Issue.Comments.Nodes {
		if comment.BodyText != "" {
			// "vibe project comment" で始まるコメントは除外
			if !strings.HasPrefix(comment.BodyText, "vibe project comment") {
				comments = append(comments, comment.BodyText)
			}
		}
	}

	return comments, nil
}
