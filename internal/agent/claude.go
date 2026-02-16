package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ClaudeAgent interfaces with the Claude API / Claude Agent SDK.
type ClaudeAgent struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewClaudeAgent creates a new Claude agent client.
func NewClaudeAgent(apiKey, model string) *ClaudeAgent {
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &ClaudeAgent{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.anthropic.com/v1",
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// Message represents a conversation message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request payload for Claude.
type ChatRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
}

// Tool describes a tool available to the agent.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// ChatResponse is the response from Claude.
type ChatResponse struct {
	ID      string `json:"id"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	StopReason string `json:"stop_reason"`
}

// Chat sends a request to the Claude API.
func (c *ClaudeAgent) Chat(system string, messages []Message, tools []Tool) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    system,
		Messages:  messages,
		Tools:     tools,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("claude API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &chatResp, nil
}

// CodeReview asks the agent to review code and return suggestions.
func (c *ClaudeAgent) CodeReview(code, language, context string) (*ChatResponse, error) {
	system := `You are AetherDev's code review agent. You follow GitOps best practices, SDLC standards, and cloud-native patterns.
Review the code and provide:
1. Security issues (OWASP Top 10)
2. Performance concerns
3. Best practice violations
4. Suggested improvements
Format as structured markdown with severity levels.`

	messages := []Message{
		{Role: "user", Content: fmt.Sprintf("Review this %s code:\n\n```%s\n%s\n```\n\nContext: %s", language, language, code, context)},
	}
	return c.Chat(system, messages, nil)
}

// GenerateDocument asks the agent to generate documentation.
func (c *ClaudeAgent) GenerateDocument(docType, prompt, codeContext string) (*ChatResponse, error) {
	system := fmt.Sprintf(`You are AetherDev's documentation agent. Generate a %s document.
Follow best practices for technical documentation. Be thorough but concise.`, docType)

	messages := []Message{
		{Role: "user", Content: fmt.Sprintf("%s\n\nCode context:\n%s", prompt, codeContext)},
	}
	return c.Chat(system, messages, nil)
}

// GenerateMindMap asks the agent to create a mind map structure.
func (c *ClaudeAgent) GenerateMindMap(topic, codeContext string) (*ChatResponse, error) {
	system := `You are AetherDev's mind map agent. Generate a mind map as JSON.
Output a JSON object with this structure:
{"id":"root","label":"Topic","children":[{"id":"1","label":"Subtopic","children":[]}]}
Only output valid JSON, no markdown fences.`

	messages := []Message{
		{Role: "user", Content: fmt.Sprintf("Create a mind map for: %s\n\nContext:\n%s", topic, codeContext)},
	}
	return c.Chat(system, messages, nil)
}

// SuggestBestPractices analyzes a repo and suggests improvements.
func (c *ClaudeAgent) SuggestBestPractices(repoStructure, configFiles string) (*ChatResponse, error) {
	system := `You are AetherDev's best practices advisor. Analyze the repository and provide a structured report.
Check for:
- GitOps: branching strategy, PR templates, .gitignore, commit hooks
- SDLC: testing, CI/CD, code quality, documentation
- Cloud: IaC presence, security configs, cost optimization, compliance
- Security: secrets management, dependency scanning, SAST/DAST

Output as JSON:
{"categories":[{"name":"gitops","score":85,"items":[{"rule":"...","status":"pass|warn|fail","severity":"...","suggestion":"...","auto_fixable":true}]}]}`

	messages := []Message{
		{Role: "user", Content: fmt.Sprintf("Analyze this repository:\n\nStructure:\n%s\n\nConfig files:\n%s", repoStructure, configFiles)},
	}
	return c.Chat(system, messages, nil)
}
