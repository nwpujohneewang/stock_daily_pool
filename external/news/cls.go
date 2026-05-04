package news

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"stock/model/core_model"
	"time"

	"go.uber.org/zap"
)

type CLSClient struct {
	httpClient *http.Client
	logger     *zap.Logger
}

func NewCLSClient(logger *zap.Logger) *CLSClient {
	return &CLSClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

type clsResp struct {
	Data struct {
		RollData []clsItem `json:"roll_data"`
	} `json:"data"`
}

type clsItem struct {
	ID       int64        `json:"id"`
	CTime    int64        `json:"ctime"`
	Title    string       `json:"title"`
	Content  string       `json:"content"`
	Level    string       `json:"level"` // "A" = 重要
	Subjects []clsSubject `json:"subjects"`
}

type clsSubject struct {
	SubjectType string `json:"subject_type"` // "stock"
	SubjectCode string `json:"subject_code"` // "600519"
	SubjectName string `json:"subject_name"`
}

// FetchLatest 拉取最新电报
func (c *CLSClient) FetchLatest(ctx context.Context, pageSize int, minLevel string) ([]core_model.RawNews, error) {
	url := fmt.Sprintf(
		"https://www.cls.cn/nodeapi/telegraphList?app=CailianpressWeb&os=web&sv=8.4.6&rn=%d",
		pageSize,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://www.cls.cn/telegraph")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, preview)
	}

	var result clsResp
	if err := json.Unmarshal(body, &result); err != nil {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, preview)
	}

	newsList := make([]core_model.RawNews, 0, len(result.Data.RollData))
	for _, item := range result.Data.RollData {
		if !c.isAStockRelated(item) {
			continue
		}
		if !meetsMinLevel(item.Level, minLevel) {
			continue
		}

		publishTime := time.Unix(item.CTime, 0)

		importance := "low"
		switch item.Level {
		case "A":
			importance = "high"
		case "B":
			importance = "medium"
		}
		if judgeImportance(item.Title+" "+item.Content) == "high" {
			importance = "high"
		}

		// 提取提到的股票代码
		var mentionedStocks []string
		for _, s := range item.Subjects {
			if s.SubjectType == "stock" && s.SubjectCode != "" {
				mentionedStocks = append(mentionedStocks, s.SubjectCode)
			}
		}

		content := item.Content
		if content == "" {
			content = item.Title
		}

		newsList = append(newsList, core_model.RawNews{
			NewsID:          fmt.Sprintf("cls_%d", item.ID),
			Source:          "cls",
			Title:           item.Title,
			Content:         content,
			PublishTime:     publishTime,
			URL:             fmt.Sprintf("https://www.cls.cn/detail/%d", item.ID),
			Importance:      importance,
			MentionedStocks: mentionedStocks,
		})
	}

	c.logger.Info("cls fetch done",
		zap.Int("total", len(result.Data.RollData)),
		zap.Int("kept", len(newsList)))
	return newsList, nil
}

func (c *CLSClient) isAStockRelated(item clsItem) bool {
	for _, s := range item.Subjects {
		if s.SubjectType == "stock" {
			return true
		}
	}
	return judgeImportance(item.Title+" "+item.Content) != "low"
}

func meetsMinLevel(level, minLevel string) bool {
	if minLevel == "" || minLevel == "low" {
		return true
	}
	levelOrder := map[string]int{"A": 3, "B": 2, "C": 1}
	itemLevel := levelOrder[level]
	required := levelOrder[minLevel]
	if required == 0 {
		return true
	}
	return itemLevel >= required
}
