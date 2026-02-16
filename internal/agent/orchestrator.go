package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aetherdev/aetherdev/internal/models"
)

// Orchestrator manages agent tasks, routing between Claude and AWS agents.
type Orchestrator struct {
	claude  *ClaudeAgent
	bedrock *AWSBedrockAgent
	store   *models.Store
}

// NewOrchestrator creates a new agent orchestrator.
func NewOrchestrator(claude *ClaudeAgent, bedrock *AWSBedrockAgent, store *models.Store) *Orchestrator {
	return &Orchestrator{
		claude:  claude,
		bedrock: bedrock,
		store:   store,
	}
}

// TaskRequest describes an agent task to execute.
type TaskRequest struct {
	RepoID      int64  `json:"repo_id"`
	UserID      int64  `json:"user_id"`
	Type        string `json:"type"`
	Prompt      string `json:"prompt"`
	Provider    string `json:"provider"` // claude, aws_bedrock, auto
	CodeContext string `json:"code_context"`
	BranchName  string `json:"branch_name"`
}

// ExecuteTask runs an agent task asynchronously.
func (o *Orchestrator) ExecuteTask(ctx context.Context, req TaskRequest) (*models.AgentTask, error) {
	task := &models.AgentTask{
		RepoID:     req.RepoID,
		UserID:     req.UserID,
		Type:       req.Type,
		Prompt:     req.Prompt,
		Provider:   req.Provider,
		BranchName: req.BranchName,
	}

	if err := o.store.CreateAgentTask(task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// Run task in background
	go o.runTask(context.Background(), task, req)

	return task, nil
}

func (o *Orchestrator) runTask(ctx context.Context, task *models.AgentTask, req TaskRequest) {
	now := time.Now()
	task.StartedAt = &now
	task.Status = "running"
	o.store.UpdateAgentTask(task)

	var result string
	var tokensUsed int
	var err error

	provider := req.Provider
	if provider == "auto" || provider == "" {
		provider = "claude"
	}

	switch provider {
	case "claude":
		result, tokensUsed, err = o.runClaudeTask(req)
	case "aws_bedrock":
		result, err = o.runBedrockTask(ctx, req)
	default:
		err = fmt.Errorf("unknown provider: %s", provider)
	}

	completed := time.Now()
	task.CompletedAt = &completed
	task.TokensUsed = tokensUsed

	if err != nil {
		task.Status = "failed"
		task.Result = fmt.Sprintf("Error: %v", err)
		log.Printf("Agent task %d failed: %v", task.ID, err)
	} else {
		task.Status = "completed"
		task.Result = result
	}

	if err := o.store.UpdateAgentTask(task); err != nil {
		log.Printf("Failed to update task %d: %v", task.ID, err)
	}
}

func (o *Orchestrator) runClaudeTask(req TaskRequest) (string, int, error) {
	if o.claude == nil {
		return "", 0, fmt.Errorf("claude agent not configured")
	}

	var resp *ChatResponse
	var err error

	switch req.Type {
	case "code_review":
		resp, err = o.claude.CodeReview(req.CodeContext, "", req.Prompt)
	case "generate", "document":
		resp, err = o.claude.GenerateDocument(req.Type, req.Prompt, req.CodeContext)
	case "mindmap":
		resp, err = o.claude.GenerateMindMap(req.Prompt, req.CodeContext)
	case "best_practices":
		resp, err = o.claude.SuggestBestPractices(req.CodeContext, "")
	default:
		// Generic task
		messages := []Message{{Role: "user", Content: req.Prompt}}
		resp, err = o.claude.Chat("You are AetherDev, an AI-powered development assistant.", messages, nil)
	}

	if err != nil {
		return "", 0, err
	}

	var result string
	for _, c := range resp.Content {
		if c.Type == "text" {
			result += c.Text
		}
	}
	tokens := resp.Usage.InputTokens + resp.Usage.OutputTokens
	return result, tokens, nil
}

func (o *Orchestrator) runBedrockTask(ctx context.Context, req TaskRequest) (string, error) {
	if o.bedrock == nil {
		return "", fmt.Errorf("AWS Bedrock agent not configured")
	}

	sessionID := fmt.Sprintf("task-%d-%d", req.RepoID, time.Now().Unix())
	resp, err := o.bedrock.InvokeAgent(ctx, sessionID, req.Prompt)
	if err != nil {
		return "", err
	}
	return resp.Output, nil
}
