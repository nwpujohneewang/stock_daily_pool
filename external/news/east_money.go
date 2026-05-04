package news

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stock/model/core_model"
	"strings"
	"time"

	"go.uber.org/zap"
)

// EastMoneyClient 东方财富7x24快讯采集客户端
type EastMoneyClient struct {
	httpClient *http.Client
	logger     *zap.Logger
}

func NewEastMoneyClient(logger *zap.Logger) *EastMoneyClient {
	return &EastMoneyClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

// eastMoneyResp 东财API响应结构
type eastMoneyResp struct {
	Data struct {
		List []eastMoneyItem `json:"list"`
	} `json:"data"`
}

type eastMoneyItem struct {
	ArtCode  string `json:"art_code"`
	Title    string `json:"title"`
	Digest   string `json:"digest"`
	Content  string `json:"content"`
	ShowTime string `json:"showtime"` // "2026-04-25 09:30:00"
	URLW     string `json:"url_w"`
}

// FetchLatest 拉取最新快讯
func (c *EastMoneyClient) FetchLatest(ctx context.Context, pageSize int) ([]core_model.RawNews, error) {
	url := fmt.Sprintf(
		"https://np-listapi.eastmoney.com/comm/wap/getListInfo?client=wap&biz=im_all&type=im&page_index=1&page_size=%d",
		pageSize,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result eastMoneyResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	newsList := make([]core_model.RawNews, 0, len(result.Data.List))
	for _, item := range result.Data.List {
		publishTime, _ := time.ParseInLocation("2006-01-02 15:04:05", item.ShowTime, time.Local)

		content := item.Digest
		if content == "" {
			content = item.Title
		}

		importance := judgeImportance(item.Title + " " + content)
		if importance == "low" {
			continue
		}

		newsList = append(newsList, core_model.RawNews{
			NewsID:      fmt.Sprintf("em_%s", item.ArtCode),
			Source:      "eastmoney",
			Title:       item.Title,
			Content:     content,
			PublishTime: publishTime,
			URL:         item.URLW,
			Importance:  importance,
		})
	}

	c.logger.Info("eastmoney fetch done",
		zap.Int("total", len(result.Data.List)),
		zap.Int("kept", len(newsList)))
	return newsList, nil
}

// judgeImportance 规则预筛新闻重要性（减少LLM压力）
func judgeImportance(text string) string {
	highKeywords := []string{
		"国务院", "发改委", "工信部", "证监会", "央行", "财政部",
		"涨停", "封板", "大涨", "暴涨", "突破", "创新高",
		"利好", "重大", "紧急", "突发",
		"降准", "降息", "加息", "LPR",
	}
	sectorKeywords := []string{
		"AI", "人工智能", "算力", "芯片", "半导体", "GPU",
		"新能源", "光伏", "储能", "锂电", "充电桩",
		"低空", "无人机", "eVTOL",
		"军工", "国防", "航天",
		"医药", "创新药", "CXO",
		"华为", "特斯拉", "苹果",
		"机器人", "人形机器人",
	}
	filterKeywords := []string{
		"午间公告", "晚间公告", "盘后公告",
		"股东减持", "限售股解禁",
		"美股收盘", "欧股收盘",
	}

	for _, kw := range filterKeywords {
		if strings.Contains(text, kw) {
			return "low"
		}
	}
	for _, kw := range highKeywords {
		if strings.Contains(text, kw) {
			return "high"
		}
	}
	for _, kw := range sectorKeywords {
		if strings.Contains(text, kw) {
			return "medium"
		}
	}
	return "low"
}
