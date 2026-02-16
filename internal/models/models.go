package models

import "time"

// User represents a platform user.
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	AvatarURL string    `json:"avatar_url"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Repository represents a git repository with extended metadata.
type Repository struct {
	ID          int64     `json:"id"`
	OwnerID     int64     `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPrivate   bool      `json:"is_private"`
	DefaultBr   string    `json:"default_branch"`
	Language    string    `json:"language"`
	Stars       int       `json:"stars"`
	Forks       int       `json:"forks"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AgentTask represents a task executed by an AI agent.
type AgentTask struct {
	ID          int64      `json:"id"`
	RepoID      int64      `json:"repo_id"`
	UserID      int64      `json:"user_id"`
	Type        string     `json:"type"` // code_review, fix, generate, document, mindmap, design
	Status      string     `json:"status"` // pending, running, completed, failed
	Prompt      string     `json:"prompt"`
	Result      string     `json:"result"`
	Provider    string     `json:"provider"` // claude, aws_bedrock
	Model       string     `json:"model"`
	TokensUsed  int        `json:"tokens_used"`
	BranchName  string     `json:"branch_name"`
	CommitSHA   string     `json:"commit_sha"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Document represents an agent-generated document.
type Document struct {
	ID        int64     `json:"id"`
	RepoID    int64     `json:"repo_id"`
	TaskID    int64     `json:"task_id"`
	Title     string    `json:"title"`
	Type      string    `json:"type"` // markdown, design, mindmap, diagram, runbook
	Content   string    `json:"content"`
	Path      string    `json:"path"` // path in repo
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MindMap represents a mind map structure.
type MindMap struct {
	ID        int64     `json:"id"`
	RepoID    int64     `json:"repo_id"`
	Title     string    `json:"title"`
	RootNode  *MapNode  `json:"root_node"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MapNode represents a node in a mind map.
type MapNode struct {
	ID       string     `json:"id"`
	Label    string     `json:"label"`
	Notes    string     `json:"notes,omitempty"`
	Color    string     `json:"color,omitempty"`
	Icon     string     `json:"icon,omitempty"`
	Children []*MapNode `json:"children,omitempty"`
}

// Pipeline represents a CI/CD pipeline definition.
type Pipeline struct {
	ID        int64          `json:"id"`
	RepoID    int64          `json:"repo_id"`
	Name      string         `json:"name"`
	Status    string         `json:"status"` // idle, running, passed, failed
	Stages    []PipelineStage `json:"stages"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// PipelineStage is a stage within a pipeline.
type PipelineStage struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration int    `json:"duration_seconds"`
	LogURL   string `json:"log_url"`
}

// BestPracticeReport is the output of the best practices engine.
type BestPracticeReport struct {
	ID          int64               `json:"id"`
	RepoID      int64               `json:"repo_id"`
	OverallScore int                `json:"overall_score"` // 0-100
	Categories  []PracticeCategory  `json:"categories"`
	GeneratedAt time.Time           `json:"generated_at"`
}

// PracticeCategory groups related best practice checks.
type PracticeCategory struct {
	Name   string          `json:"name"` // gitops, sdlc, cloud, security
	Score  int             `json:"score"`
	Items  []PracticeItem  `json:"items"`
}

// PracticeItem is a single best practice check result.
type PracticeItem struct {
	Rule        string `json:"rule"`
	Description string `json:"description"`
	Status      string `json:"status"` // pass, warn, fail
	Severity    string `json:"severity"` // info, low, medium, high, critical
	Suggestion  string `json:"suggestion"`
	AutoFixable bool   `json:"auto_fixable"`
}
