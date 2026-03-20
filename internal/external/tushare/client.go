package tushare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"stock/internal/config"
)

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func isRetryableFromStatus(statusCode int) bool {
	if statusCode >= 400 && statusCode < 500 {
		return statusCode == 429
	}
	return statusCode >= 500 || statusCode == 0
}

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	rateLimit  int
	retryCfg   config.RetryConfig
}

type TushareRequest struct {
	APIName string                 `json:"api_name"`
	Token   string                 `json:"token"`
	Params  map[string]interface{} `json:"params,omitempty"`
	Fields  string                 `json:"fields,omitempty"`
}

type TushareResponse struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data *TushareData `json:"data,omitempty"`
}

type TushareData struct {
	Fields []string        `json:"fields"`
	Items  [][]interface{} `json:"items"`
}

func NewClient(cfg *config.TushareConfig, retryCfg config.RetryConfig) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		token:   cfg.Token,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		rateLimit: cfg.RateLimitPerMin,
		retryCfg:  retryCfg,
	}
}

func (c *Client) doRequestOnce(ctx context.Context, req *TushareRequest) (*TushareResponse, error) {
	req.Token = c.token

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &HTTPError{StatusCode: 0, Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("status code %d", resp.StatusCode)}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result TushareResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("tushare error: %s", result.Msg)
	}

	return &result, nil
}

func (c *Client) doRequest(ctx context.Context, req *TushareRequest) (*TushareResponse, error) {
	var lastErr error
	delay := c.retryCfg.InitialDelay()

	for attempt := 0; attempt <= c.retryCfg.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
			delay = time.Duration(float64(delay) * c.retryCfg.Multiplier)
			if delay > c.retryCfg.MaxDelay() {
				delay = c.retryCfg.MaxDelay()
			}
		}

		resp, err := c.doRequestOnce(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			if !isRetryableFromStatus(httpErr.StatusCode) {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
}

type StockBasicItem struct {
	TsCode   string `json:"ts_code"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Industry string `json:"industry"`
	ListDate string `json:"list_date"`
	Exchange string `json:"exchange"`
	IsHS     string `json:"is_hs"`
	IsST     bool   `json:"is_st"`
}

func (c *Client) StockBasic(ctx context.Context) ([]StockBasicItem, error) {
	req := &TushareRequest{
		APIName: "stock_basic",
		Params:  map[string]interface{}{"exchange": "", "list_status": "L"},
		Fields:  "ts_code,symbol,name,industry,list_date,exchange,is_hs,is_st",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, nil
	}

	var results []StockBasicItem
	for _, item := range resp.Data.Items {
		if len(item) < 8 {
			continue
		}
		results = append(results, StockBasicItem{
			TsCode:   toString(item[0]),
			Symbol:   toString(item[1]),
			Name:     toString(item[2]),
			Industry: toString(item[3]),
			ListDate: toString(item[4]),
			Exchange: toString(item[5]),
			IsHS:     toString(item[6]),
			IsST:     item[7] == "1",
		})
	}

	return results, nil
}

type QuoteItem struct {
	TsCode       string  `json:"ts_code"`
	PreClose     float64 `json:"pre_close"`
	Price        float64 `json:"price"`
	PctChg       float64 `json:"pct_chg"`
	Vol          int64   `json:"vol"`
	Amount       float64 `json:"amount"`
	TurnoverRate float64 `json:"turnover_rate"`
}

func (c *Client) RealtimeQuote(ctx context.Context, tsCodes []string) ([]QuoteItem, error) {
	var allQuotes []QuoteItem

	for _, batch := range splitCodes(tsCodes, 50) {
		req := &TushareRequest{
			APIName: "realtime_quote",
			Params:  map[string]interface{}{"ts_code": strings.Join(batch, ",")},
			Fields:  "ts_code,pre_close,price,pct_chg,vol,amount,turnover_rate",
		}

		resp, err := c.doRequest(ctx, req)
		if err != nil {
			return allQuotes, err
		}

		if resp.Data == nil {
			continue
		}

		for _, item := range resp.Data.Items {
			if len(item) < 7 {
				continue
			}
			allQuotes = append(allQuotes, QuoteItem{
				TsCode:       toString(item[0]),
				PreClose:     toFloat64(item[1]),
				Price:        toFloat64(item[2]),
				PctChg:       toFloat64(item[3]),
				Vol:          toInt64(item[4]),
				Amount:       toFloat64(item[5]),
				TurnoverRate: toFloat64(item[6]),
			})
		}
	}

	return allQuotes, nil
}

type ConceptItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Src  string `json:"src"`
}

func (c *Client) ConceptList(ctx context.Context) ([]ConceptItem, error) {
	req := &TushareRequest{
		APIName: "concept",
		Params:  map[string]interface{}{},
		Fields:  "id,name,src",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, nil
	}

	var results []ConceptItem
	for _, item := range resp.Data.Items {
		if len(item) < 3 {
			continue
		}
		results = append(results, ConceptItem{
			ID:   toString(item[0]),
			Name: toString(item[1]),
			Src:  toString(item[2]),
		})
	}

	return results, nil
}

type ConceptDetailItem struct {
	ID          string `json:"id"`
	ConceptName string `json:"concept_name"`
	TsCode      string `json:"ts_code"`
	Name        string `json:"name"`
}

func (c *Client) ConceptDetail(ctx context.Context, conceptID string) ([]ConceptDetailItem, error) {
	req := &TushareRequest{
		APIName: "concept_detail",
		Params:  map[string]interface{}{"id": conceptID},
		Fields:  "id,concept_name,ts_code,name",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, nil
	}

	var results []ConceptDetailItem
	for _, item := range resp.Data.Items {
		if len(item) < 4 {
			continue
		}
		results = append(results, ConceptDetailItem{
			ID:          toString(item[0]),
			ConceptName: toString(item[1]),
			TsCode:      toString(item[2]),
			Name:        toString(item[3]),
		})
	}

	return results, nil
}

func splitCodes(codes []string, size int) [][]string {
	var result [][]string
	for i := 0; i < len(codes); i += size {
		end := i + size
		if end > len(codes) {
			end = len(codes)
		}
		result = append(result, codes[i:end])
	}
	return result
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}
