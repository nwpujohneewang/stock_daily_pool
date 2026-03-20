package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"stock/config"
)

type Client struct {
	model       string
	apiURL      string
	apiKey      string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
}

type ChatCompletionRequest struct {
	Model       string                  `json:"model"`
	Temperature float64                 `json:"temperature"`
	MaxTokens   int                     `json:"max_tokens"`
	Messages    []ChatCompletionMessage `json:"messages"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message ChatMessage `json:"message"`
}

type ChatMessage struct {
	Content string `json:"content"`
}

type ClassifyResponse struct {
	TopicName  string  `json:"topic_name"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

func NewClient(cfg *config.LLMConfig) *Client {
	return &Client{
		model:       cfg.Model,
		apiURL:      cfg.APIURL,
		apiKey:      cfg.APIKey,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

func (c *Client) Classify(ctx context.Context, stock, industry string, activeTopics []string) (*ClassifyResponse, error) {
	systemPrompt := `你是A股市场热点分析专家。给定一只股票的名称、行业信息和当日活跃的热点列表，判断该股票最可能属于哪个热点。返回 JSON 格式：{"topic_name": "热点名称", "confidence": 0.0-1.0, "reasoning": "判断理由"}`

	userMsg := fmt.Sprintf("股票：%s，行业：%s。当日活跃热点：%v。请判断该股票今日涨停最可能属于哪个热点。", stock, industry, activeTopics)

	req := ChatCompletionRequest{
		Model:       c.model,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
		Messages: []ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	var classifyResp ClassifyResponse
	content := result.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), &classifyResp); err != nil {
		return &ClassifyResponse{
			Reasoning: content,
		}, fmt.Errorf("parse JSON: %w", err)
	}

	return &classifyResp, nil
}
