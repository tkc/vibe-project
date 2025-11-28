package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/tkc/vibe-project/internal/domain"
)

// フィールド名定数
const (
	FieldStatus     = "Status"
	FieldPrompt     = "Prompt"
	FieldResult     = "Result"
	FieldSessionID  = "SessionID"
	FieldExecutedAt = "ExecutedAt"
)

// TaskService はタスク操作を提供する
type TaskService struct {
	client        *Client
	projectID     string
	projectNumber int
	fields        map[string]ProjectField // フィールド名 -> フィールド情報
}

// NewTaskService は新しいTaskServiceを作成する
func NewTaskService(client *Client, projectNumber int) *TaskService {
	return &TaskService{
		client:        client,
		projectNumber: projectNumber,
		fields:        make(map[string]ProjectField),
	}
}

// Initialize はProjectの情報を取得してサービスを初期化する
func (s *TaskService) Initialize(ctx context.Context) error {
	// Project IDを取得
	project, err := s.client.GetProjectByNumber(ctx, s.projectNumber)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	s.projectID = project.ID

	// フィールド情報を取得
	if err := s.loadFields(ctx); err != nil {
		return fmt.Errorf("failed to load fields: %w", err)
	}

	return nil
}

func (s *TaskService) loadFields(ctx context.Context) error {
	var query struct {
		Node struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []struct {
						TypeName    string `graphql:"__typename"`
						FieldCommon struct {
							ID   string
							Name string
						} `graphql:"... on ProjectV2FieldCommon"`
						SingleSelect struct {
							Options []struct {
								ID   string
								Name string
							}
						} `graphql:"... on ProjectV2SingleSelectField"`
					}
				} `graphql:"fields(first: 30)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectId)"`
	}

	variables := map[string]interface{}{
		"projectId": githubv4.ID(s.projectID),
	}

	if err := s.client.gql.Query(ctx, &query, variables); err != nil {
		return err
	}

	for _, f := range query.Node.ProjectV2.Fields.Nodes {
		field := ProjectField{
			ID:   f.FieldCommon.ID,
			Name: f.FieldCommon.Name,
		}
		if f.TypeName == "ProjectV2SingleSelectField" {
			for _, opt := range f.SingleSelect.Options {
				field.Options = append(field.Options, FieldOption{
					ID:   opt.ID,
					Name: opt.Name,
				})
			}
		}
		s.fields[f.FieldCommon.Name] = field
	}

	return nil
}

// GetTasks はProjectのタスク一覧を取得する
func (s *TaskService) GetTasks(ctx context.Context, filter *domain.TaskFilter) ([]*domain.Task, error) {
	var query struct {
		Node struct {
			ProjectV2 struct {
				Items struct {
					Nodes []struct {
						ID      string
						Content struct {
							Issue struct {
								Title string
								URL   string
							} `graphql:"... on Issue"`
							DraftIssue struct {
								Title string
							} `graphql:"... on DraftIssue"`
						}
						FieldValues struct {
							Nodes []struct {
								TypeName  string `graphql:"__typename"`
								TextField struct {
									Text  string
									Field struct {
										FieldCommon struct {
											Name string
										} `graphql:"... on ProjectV2FieldCommon"`
									} `graphql:"field"`
								} `graphql:"... on ProjectV2ItemFieldTextValue"`
								SingleSelect struct {
									Name  string
									Field struct {
										FieldCommon struct {
											Name string
										} `graphql:"... on ProjectV2FieldCommon"`
									} `graphql:"field"`
								} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
								DateField struct {
									Date  string
									Field struct {
										FieldCommon struct {
											Name string
										} `graphql:"... on ProjectV2FieldCommon"`
									} `graphql:"field"`
								} `graphql:"... on ProjectV2ItemFieldDateValue"`
							}
						} `graphql:"fieldValues(first: 20)"`
					}
				} `graphql:"items(first: 100)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectId)"`
	}

	variables := map[string]interface{}{
		"projectId": githubv4.ID(s.projectID),
	}

	if err := s.client.gql.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}

	tasks := make([]*domain.Task, 0)
	for _, item := range query.Node.ProjectV2.Items.Nodes {
		task := &domain.Task{
			ID: item.ID,
		}

		// タイトルを取得
		if item.Content.Issue.Title != "" {
			task.Title = item.Content.Issue.Title
			task.IssueURL = item.Content.Issue.URL
		} else {
			task.Title = item.Content.DraftIssue.Title
		}

		// フィールド値を取得
		for _, fv := range item.FieldValues.Nodes {
			var fieldName string
			switch fv.TypeName {
			case "ProjectV2ItemFieldTextValue":
				fieldName = fv.TextField.Field.FieldCommon.Name
			case "ProjectV2ItemFieldSingleSelectValue":
				fieldName = fv.SingleSelect.Field.FieldCommon.Name
			case "ProjectV2ItemFieldDateValue":
				fieldName = fv.DateField.Field.FieldCommon.Name
			}

			switch fieldName {
			case FieldStatus:
				task.Status = domain.Status(fv.SingleSelect.Name)
			case FieldPrompt:
				task.Prompt = fv.TextField.Text
			case FieldResult:
				task.Result = fv.TextField.Text
			case FieldSessionID:
				task.SessionID = fv.TextField.Text
			case FieldExecutedAt:
				if fv.DateField.Date != "" {
					t, _ := time.Parse("2006-01-02", fv.DateField.Date)
					task.ExecutedAt = &t
				}
			}
		}

		// フィルタ適用
		if filter != nil && filter.Status != nil {
			if task.Status != *filter.Status {
				continue
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTask は指定IDのタスクを取得する
func (s *TaskService) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	tasks, err := s.GetTasks(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, t := range tasks {
		if t.ID == taskID {
			return t, nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// UpdateTask はタスクのフィールドを更新する
func (s *TaskService) UpdateTask(ctx context.Context, task *domain.Task, exec *domain.Execution) error {
	// Statusを更新
	if err := s.updateSingleSelectField(ctx, task.ID, FieldStatus, string(exec.NewStatus())); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Resultを更新
	if err := s.updateTextField(ctx, task.ID, FieldResult, exec.Summary()); err != nil {
		return fmt.Errorf("failed to update result: %w", err)
	}

	// SessionIDを更新
	if exec.SessionID != "" {
		if err := s.updateTextField(ctx, task.ID, FieldSessionID, exec.SessionID); err != nil {
			return fmt.Errorf("failed to update session id: %w", err)
		}
	}

	// ExecutedAtを更新
	if err := s.updateDateField(ctx, task.ID, FieldExecutedAt, exec.EndedAt); err != nil {
		return fmt.Errorf("failed to update executed at: %w", err)
	}

	return nil
}

// SetTaskInProgress はタスクをInProgressに設定する
func (s *TaskService) SetTaskInProgress(ctx context.Context, taskID string) error {
	return s.updateSingleSelectField(ctx, taskID, FieldStatus, string(domain.StatusInProgress))
}

func (s *TaskService) updateTextField(ctx context.Context, itemID, fieldName, value string) error {
	field, ok := s.fields[fieldName]
	if !ok {
		return fmt.Errorf("field not found: %s", fieldName)
	}

	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID string
			} `graphql:"projectV2Item"`
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: githubv4.ID(s.projectID),
		ItemID:    githubv4.ID(itemID),
		FieldID:   githubv4.ID(field.ID),
		Value: githubv4.ProjectV2FieldValue{
			Text: githubv4.NewString(githubv4.String(value)),
		},
	}

	return s.client.gql.Mutate(ctx, &mutation, input, nil)
}

func (s *TaskService) updateSingleSelectField(ctx context.Context, itemID, fieldName, optionName string) error {
	field, ok := s.fields[fieldName]
	if !ok {
		return fmt.Errorf("field not found: %s", fieldName)
	}

	// オプションIDを検索
	var optionID string
	for _, opt := range field.Options {
		if opt.Name == optionName {
			optionID = opt.ID
			break
		}
	}
	if optionID == "" {
		return fmt.Errorf("option not found: %s in field %s", optionName, fieldName)
	}

	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID string
			} `graphql:"projectV2Item"`
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: githubv4.ID(s.projectID),
		ItemID:    githubv4.ID(itemID),
		FieldID:   githubv4.ID(field.ID),
		Value: githubv4.ProjectV2FieldValue{
			SingleSelectOptionID: githubv4.NewString(githubv4.String(optionID)),
		},
	}

	return s.client.gql.Mutate(ctx, &mutation, input, nil)
}

func (s *TaskService) updateDateField(ctx context.Context, itemID, fieldName string, date time.Time) error {
	field, ok := s.fields[fieldName]
	if !ok {
		return fmt.Errorf("field not found: %s", fieldName)
	}

	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ProjectV2Item struct {
				ID string
			} `graphql:"projectV2Item"`
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	dateStr := date.Format("2006-01-02")
	dateValue := githubv4.Date{Time: date}
	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: githubv4.ID(s.projectID),
		ItemID:    githubv4.ID(itemID),
		FieldID:   githubv4.ID(field.ID),
		Value: githubv4.ProjectV2FieldValue{
			Date: &dateValue,
		},
	}
	_ = dateStr // suppress unused variable warning

	return s.client.gql.Mutate(ctx, &mutation, input, nil)
}

// AddIssueComment はタスクに紐づくIssueにコメントを追加する
func (s *TaskService) AddIssueComment(ctx context.Context, task *domain.Task, body string) error {
	if task.IssueURL == "" {
		return fmt.Errorf("task has no associated issue")
	}
	return s.client.AddIssueComment(ctx, task.IssueURL, body)
}

// GetFirstReadyTask はReadyステータスの最初のタスクを取得する
func (s *TaskService) GetFirstReadyTask(ctx context.Context) (*domain.Task, error) {
	status := domain.StatusReady
	filter := &domain.TaskFilter{Status: &status}
	tasks, err := s.GetTasks(ctx, filter)
	if err != nil {
		return nil, err
	}

	for _, t := range tasks {
		if t.IsExecutable() {
			return t, nil
		}
	}

	return nil, nil // 実行可能なタスクがない
}

// LoadTaskPrompt はIssueのコメントからプロンプトを読み込む
func (s *TaskService) LoadTaskPrompt(ctx context.Context, task *domain.Task) error {
	if task.IssueURL == "" {
		return fmt.Errorf("task has no associated issue")
	}

	comments, err := s.client.GetIssueComments(ctx, task.IssueURL)
	if err != nil {
		return fmt.Errorf("failed to get issue comments: %w", err)
	}

	if len(comments) == 0 {
		return fmt.Errorf("no comments found in issue")
	}

	// 全コメントを改行で結合してプロンプトとする
	task.Prompt = strings.Join(comments, "\n\n---\n\n")
	return nil
}

// GetStatusOptions はStatusフィールドの選択肢一覧を返す
func (s *TaskService) GetStatusOptions() []FieldOption {
	if field, ok := s.fields[FieldStatus]; ok {
		return field.Options
	}
	return nil
}

// GetFields は全フィールド情報を返す
func (s *TaskService) GetFields() map[string]ProjectField {
	return s.fields
}
