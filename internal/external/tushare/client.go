package tushare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"math"
	"net/http"
	"stock/internal/pkg/logger"
	"strings"
	"time"

	"stock/config"
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
		retryCfg: retryCfg,
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
	Area     string `json:"area"`
}

func (c *Client) StockBasic(ctx context.Context) ([]StockBasicItem, error) {
	req := &TushareRequest{
		APIName: "stock_basic",
		Params:  map[string]interface{}{"exchange": "", "list_status": "L"},
		Fields:  "ts_code,symbol,name,industry,list_date,exchange,is_hs,area",
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
			Area:     toString(item[7]),
		})
	}

	return results, nil
}

// StockST 返回当前处于 ST（包括*S、ST、SST、*ST 等特殊处理状态的股票
func (c *Client) StockST(ctx context.Context) (map[string]bool, error) {
	req := &TushareRequest{
		APIName: "stock_st",
		Params:  map[string]interface{}{},
		Fields:  "ts_code",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, nil
	}

	result := make(map[string]bool)
	for _, item := range resp.Data.Items {
		if len(item) < 1 {
			continue
		}
		result[toString(item[0])] = true
	}

	return result, nil
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

func (c *Client) RealtimeQuoteAll(ctx context.Context) ([]QuoteItem, error) {
	req := &TushareRequest{
		APIName: "rt_k",
		Params:  map[string]interface{}{"ts_code": "0*.SZ,3*.SZ,6*.SH,688*.SH"},
		Fields:  "ts_code,pre_close,close,vol,amount",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		logger.Info("empty response from tuShare", zap.Any("msg", resp.Msg))
		return nil, nil
	}

	var results []QuoteItem
	var over5 []QuoteItem
	for _, item := range resp.Data.Items {
		if len(item) < 5 {
			continue
		}
		quoteItem := QuoteItem{
			TsCode:   toString(item[0]),
			PreClose: toFloat64(item[1]),
			Price:    toFloat64(item[2]),
			PctChg:   calPct(toFloat64(item[1]), toFloat64(item[2])),
			Vol:      toInt64(item[3]),
			Amount:   toFloat64(item[4]),
		}
		results = append(results, quoteItem)
		if quoteItem.PctChg >= 5.0 {
			over5 = append(over5, quoteItem)
		}
	}

	return results, nil
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

func (c *Client) Daily(ctx context.Context, tradeDate string, tsCodes []string) ([]QuoteItem, error) {
	if tradeDate == "" {
		return nil, fmt.Errorf("trade_date is required")
	}
	if len(tsCodes) == 0 {
		return nil, fmt.Errorf("ts_codes is empty")
	}

	var allQuotes []QuoteItem
	for _, batch := range splitCodes(tsCodes, 50) {
		req := &TushareRequest{
			APIName: "daily",
			Params: map[string]interface{}{
				"ts_code":    strings.Join(batch, ","),
				"trade_date": tradeDate,
			},
			Fields: "ts_code,trade_date,open,high,low,close,pre_close,change,pct_chg,vol,amount",
		}

		resp, err := c.doRequest(ctx, req)
		if err != nil {
			return allQuotes, err
		}

		if resp.Data == nil {
			continue
		}

		for _, item := range resp.Data.Items {
			if len(item) < 11 {
				continue
			}
			allQuotes = append(allQuotes, QuoteItem{
				TsCode:   toString(item[0]),
				PreClose: toFloat64(item[6]),
				Price:    toFloat64(item[5]),
				PctChg:   toFloat64(item[8]),
				Vol:      toInt64(item[9]),
				Amount:   toFloat64(item[10]),
			})
		}
	}

	return allQuotes, nil
}

func (c *Client) DailyAll(ctx context.Context, tradeDate string) ([]*QuoteItem, error) {
	if tradeDate == "" {
		return nil, fmt.Errorf("trade_date is required")
	}

	req := &TushareRequest{
		APIName: "daily",
		Params: map[string]interface{}{
			"trade_date": tradeDate,
		},
		Fields: "ts_code,trade_date,open,high,low,close,pre_close,change,pct_chg,vol,amount",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, nil
	}

	var results []*QuoteItem
	for _, item := range resp.Data.Items {
		if len(item) < 11 {
			continue
		}
		results = append(results, &QuoteItem{
			TsCode:   toString(item[0]),
			PreClose: toFloat64(item[6]),
			Price:    toFloat64(item[5]),
			PctChg:   toFloat64(item[8]),
			Vol:      toInt64(item[9]),
			Amount:   toFloat64(item[10]),
		})
	}

	return results, nil
}

// LimitListItem holds limit-up/down detail info from tushare limit_list_d API
type LimitListItem struct {
	TsCode     string // 股票代码 "000001.SZ"
	Name       string // 股票名称
	FirstTime  string // 首次封板时间 "09:31:05"
	LastTime   string // 最后封板时间 "14:55:00"
	LimitTimes int    // 连板数
}

// LimitListD 获取当天涨停股列表
// tradeDate 格式: "20260329" (无连字符)
func (c *Client) LimitListD(ctx context.Context, tradeDate string) ([]LimitListItem, error) {
	req := &TushareRequest{
		APIName: "limit_list_d",
		Params: map[string]interface{}{
			"trade_date": tradeDate,
			"limit_type": "U",
		},
		Fields: "ts_code,name,first_time,last_time,limit_times",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, nil
	}

	var results []LimitListItem
	for _, item := range resp.Data.Items {
		if len(item) < 5 {
			continue
		}
		results = append(results, LimitListItem{
			TsCode:     toString(item[0]),
			Name:       toString(item[1]),
			FirstTime:  formatTushareTime(toString(item[2])),
			LastTime:   formatTushareTime(toString(item[3])),
			LimitTimes: int(toInt64(item[4])),
		})
	}

	return results, nil
}

// DailyBasicItem holds daily basic indicators from tushare daily_basic API
type DailyBasicItem struct {
	TsCode  string  // 股票代码 "000001.SZ"
	TotalMv float64 // 总市值（万元）
	CircMv  float64 // 流通市值（万元）
}

// DailyBasic 获取每日指标（市值等）
// tradeDate 格式: "20260329" (无连字符)
func (c *Client) DailyBasic(ctx context.Context, tradeDate string) ([]DailyBasicItem, error) {
	if tradeDate == "" {
		return nil, fmt.Errorf("trade_date is required")
	}

	req := &TushareRequest{
		APIName: "daily_basic",
		Params: map[string]interface{}{
			"trade_date": tradeDate,
		},
		Fields: "ts_code,trade_date,total_mv,circ_mv",
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Data == nil {
		return nil, nil
	}

	var results []DailyBasicItem
	for _, item := range resp.Data.Items {
		if len(item) < 4 {
			continue
		}
		results = append(results, DailyBasicItem{
			TsCode:  toString(item[0]),
			TotalMv: toFloat64(item[2]),
			CircMv:  toFloat64(item[3]),
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

// formatTushareTime converts tushare time format to HH:MM:SS
// Input: "130830" -> Output: "13:08:30"
// Input: "93339" -> Output: "09:33:39"
func formatTushareTime(s string) string {
	if s == "" {
		return ""
	}
	// Pad to 6 digits if needed (e.g., "93339" -> "093339")
	for len(s) < 6 {
		s = "0" + s
	}
	if len(s) != 6 {
		return s
	}
	return s[0:2] + ":" + s[2:4] + ":" + s[4:6]
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

func calPct(pre, now float64) float64 {
	raw := (now - pre) / pre * 100.0
	rounded := math.Round(raw*100) / 100 // 四舍五入到两位小数
	return rounded
}
