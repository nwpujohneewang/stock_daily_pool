package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"stock/config"
)

// anthropicChatModel implements einomodel.BaseChatModel via direct Anthropic HTTP API.
type anthropicChatModel struct {
	model      string
	apiURL     string
	apiKey     string
	maxTokens  int
	httpClient *http.Client
}

// Anthropic Messages API request/response types

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Thinking  anthropicThinking  `json:"thinking"`
}

type anthropicThinking struct {
	Type string `json:"type"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
	Error   *anthropicError    `json:"error,omitempty"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (m *anthropicChatModel) Generate(ctx context.Context, input []*schema.Message, _ ...einomodel.Option) (*schema.Message, error) {
	var systemPrompt string
	var userParts []string
	for _, msg := range input {
		switch msg.Role {
		case schema.System:
			systemPrompt = msg.Content
		case schema.User:
			userParts = append(userParts, msg.Content)
		}
	}
	userPrompt := strings.Join(userParts, "\n")

	req := anthropicRequest{
		Model:     m.model,
		MaxTokens: m.maxTokens,
		System:    systemPrompt,
		Thinking:  anthropicThinking{Type: "disabled"},
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.apiURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", m.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(errBody))
	}

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("api error %s: %s", result.Error.Type, result.Error.Message)
	}

	for _, block := range result.Content {
		if block.Type == "text" {
			return schema.AssistantMessage(block.Text, nil), nil
		}
	}
	return nil, fmt.Errorf("no text content in response")
}

// Stream is not used; satisfies the BaseChatModel interface.
func (m *anthropicChatModel) Stream(_ context.Context, _ []*schema.Message, _ ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("streaming not supported")
}

// Client wraps an Eino BaseChatModel and exposes a convenience ChatCompletion method.
type Client struct {
	chatModel einomodel.BaseChatModel
}

type ClassifyResponse struct {
	TopicName  string  `json:"topic_name"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

type BatchClassifyItem struct {
	TsCode     string  `json:"ts_code"`
	TopicName  string  `json:"topic_name"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

func NewClient(cfg *config.LLMConfig) *Client {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" || strings.HasPrefix(apiKey, "${") {
		return nil
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		chatModel: &anthropicChatModel{
			model:      cfg.Model,
			apiURL:     cfg.APIURL,
			apiKey:     apiKey,
			maxTokens:  cfg.MaxTokens,
			httpClient: &http.Client{Timeout: timeout},
		},
	}
}

func (c *Client) ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	msgs := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}
	resp, err := c.chatModel.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (c *Client) Classify(ctx context.Context, stock, industry string, activeTopics []string) (*ClassifyResponse, error) {
	systemPrompt := `你是A股市场热点分析专家。给定一只股票的名称、行业信息和当日活跃的热点列表，判断该股票最可能属于哪个热点。返回 JSON 格式：{"topic_name": "热点名称", "confidence": 0.0-1.0, "reasoning": "判断理由"}`
	userMsg := fmt.Sprintf("股票：%s，行业：%s。当日活跃热点：%v。请判断该股票今日涨停最可能属于哪个热点。", stock, industry, activeTopics)

	content, err := c.ChatCompletion(ctx, systemPrompt, userMsg)
	if err != nil {
		return nil, err
	}

	var classifyResp ClassifyResponse
	if err := json.Unmarshal([]byte(content), &classifyResp); err != nil {
		return &ClassifyResponse{Reasoning: content}, fmt.Errorf("parse JSON: %w", err)
	}
	return &classifyResp, nil
}
