package llmnode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"brook-agent/internal/model"
)

type openAIClient struct {
	cfg    Config
	client *http.Client
}

// NewOpenAIClient 创建 OpenAI Chat Completions 协议客户端（可对接 Ollama 兼容接口）。
func NewOpenAIClient(cfg Config) Client {
	return &openAIClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *openAIClient) Generate(ctx context.Context, messages []model.Message) (model.Message, error) {
	reqBody := openAIChatRequest{
		Model:       c.cfg.Model,
		Temperature: c.cfg.Temperature,
		Messages:    toOpenAIMessages(c.cfg.SystemPrompt, messages),
		Tools:       defaultTools(),
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return model.Message{}, err
	}

	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return model.Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return model.Message{}, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return model.Message{}, fmt.Errorf("llm request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return model.Message{}, err
	}
	if len(parsed.Choices) == 0 {
		return model.Message{}, fmt.Errorf("llm returned empty choices")
	}

	choice := parsed.Choices[0].Message
	out := model.Message{
		Role:    model.RoleAssistant,
		Content: choice.Content,
	}
	for _, tc := range choice.ToolCalls {
		args, err := convertArguments(tc.Function.Arguments)
		if err != nil {
			return model.Message{}, err
		}
		out.ToolCalls = append(out.ToolCalls, model.ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		})
	}
	return out, nil
}

func toOpenAIMessages(systemPrompt string, src []model.Message) []openAIMessage {
	out := make([]openAIMessage, 0, len(src)+1)
	if systemPrompt != "" {
		out = append(out, openAIMessage{
			Role:    model.RoleSystem,
			Content: systemPrompt,
		})
	}
	for _, m := range src {
		item := openAIMessage{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			argBytes, _ := json.Marshal(tc.Args)
			item.ToolCalls = append(item.ToolCalls, openAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: openAIFunctionCall{
					Name:      tc.Name,
					Arguments: string(argBytes),
				},
			})
		}
		out = append(out, item)
	}
	return out
}

func convertArguments(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}, nil
	}
	anyMap := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &anyMap); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(anyMap))
	for k, v := range anyMap {
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}

func defaultTools() []openAITool {
	return []openAITool{
		{
			Type: "function",
			Function: openAIFunction{
				Name:        "bash",
				Description: "Execute shell command on host machine",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]string{
							"type":        "string",
							"description": "Shell command text",
						},
					},
					"required": []string{"command"},
				},
			},
		},
		{
			Type: "function",
			Function: openAIFunction{
				Name:        "file",
				Description: "Read write or list files",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"op": map[string]string{
							"type":        "string",
							"description": "read write or list",
						},
						"path": map[string]string{
							"type":        "string",
							"description": "Target file or directory path",
						},
						"content": map[string]string{
							"type":        "string",
							"description": "File content when op=write",
						},
					},
					"required": []string{"op", "path"},
				},
			},
		},
		{
			Type: "function",
			Function: openAIFunction{
				Name:        "network",
				Description: "Send http requests",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"method": map[string]string{
							"type":        "string",
							"description": "HTTP method GET or POST",
						},
						"url": map[string]string{
							"type":        "string",
							"description": "HTTP URL",
						},
						"body": map[string]string{
							"type":        "string",
							"description": "Request body for POST",
						},
					},
					"required": []string{"url"},
				},
			},
		},
	}
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIChatResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openAIChoice       `json:"choices"`
	Usage   openAIResponseUsages `json:"usage"`
}

type openAIChoice struct {
	Index   int           `json:"index"`
	Message openAIMessage `json:"message"`
}

type openAIResponseUsages struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
