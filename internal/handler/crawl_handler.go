package handler

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"stock/internal/service/crawler"
	"strings"

	"stock/model/api"
	"stock/model/api/request"
	"stock/model/api/response"

	"github.com/gin-gonic/gin"
)

type CrawlHandler struct{}

func NewCrawlHandler() *CrawlHandler {
	return &CrawlHandler{}
}

// parseCurl 从 curl 命令解析出认证参数
func parseCurl(curlCmd string) (*crawler.CrawlParams, error) {
	curlCmd = strings.TrimSpace(curlCmd)

	// Extract Cookie: -b '...' or -b "..."
	cookieRegex := regexp.MustCompile(`-b\s+['"]([^'"]+)['"]`)
	cookieMatch := cookieRegex.FindStringSubmatch(curlCmd)
	if len(cookieMatch) < 2 {
		return nil, fmt.Errorf("未找到 Cookie，请确认 curl 命令包含 -b 参数")
	}

	// Extract Token: -H 'token: xxx'
	tokenRegex := regexp.MustCompile(`-H\s+['"]token:\s*([^'"]+)['"]`)
	tokenMatch := tokenRegex.FindStringSubmatch(curlCmd)
	if len(tokenMatch) < 2 {
		return nil, fmt.Errorf("未找到 Token，请确认 curl 命令包含 -H 'token: ...'")
	}

	// Extract Timestamp: -H 'timestamp: xxx'
	tsRegex := regexp.MustCompile(`-H\s+['"]timestamp:\s*([^'"]+)['"]`)
	tsMatch := tsRegex.FindStringSubmatch(curlCmd)
	if len(tsMatch) < 2 {
		return nil, fmt.Errorf("未找到 Timestamp，请确认 curl 命令包含 -H 'timestamp: ...'")
	}

	// Extract Date: --data-raw '{"date":"YYYY-MM-DD",...}'
	dateRegex := regexp.MustCompile(`--data-raw\s+['"]\{[^}]*"date"\s*:\s*"(\d{4}-\d{2}-\d{2})"`)
	dateMatch := dateRegex.FindStringSubmatch(curlCmd)
	date := ""
	if len(dateMatch) >= 2 {
		date = dateMatch[1]
	}

	return &crawler.CrawlParams{
		Token:     tokenMatch[1],
		Cookie:    cookieMatch[1],
		Timestamp: tsMatch[1],
		Date:      date,
	}, nil
}

// CrawlJiuyan 爬取韭研数据
func (h *CrawlHandler) CrawlJiuyan(c *gin.Context) {
	var req request.CrawlJiuyanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid request: "+err.Error()))
		return
	}

	params, err := parseCurl(req.Curl)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "parse curl failed: "+err.Error()))
		return
	}

	ctx := c.Request.Context()
	crawlerSvc := crawler.NewCrawlerService()

	result, err := crawlerSvc.CrawlWithParams(ctx, *params)
	if err != nil {
		// 处理特定错误
		if errors.Is(err, crawler.ErrEmptyDate) {
			c.JSON(http.StatusBadRequest, api.Fail(400, "curl command does not contain date"))
			return
		}
		if errors.Is(err, crawler.ErrInvalidDate) {
			c.JSON(http.StatusBadRequest, api.Fail(400, "invalid date format"))
			return
		}

		var missingErr *crawler.MissingTopicsError
		if errors.As(err, &missingErr) {
			c.JSON(http.StatusBadRequest, api.FailWithData(400, "some topics are missing from dictionary", missingErr.Topics))
			return
		}

		c.JSON(http.StatusInternalServerError, api.Fail(500, "crawl failed: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.OK(response.CrawlJiuyanResp{
		Date:        result.Date,
		TopicsCount: result.TopicsCount,
		StocksCount: result.StocksCount,
	}))
}

// RebuildTopicRelations 从已存储的 jiuyan_raw_data 重建 topic relations
func (h *CrawlHandler) RebuildTopicRelations(c *gin.Context) {
	var req request.RebuildTopicRelationsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid request: "+err.Error()))
		return
	}

	ctx := c.Request.Context()
	crawlerSvc := crawler.NewCrawlerService()

	result, err := crawlerSvc.RebuildTopicRelations(ctx, req.StartDate, req.EndDate)
	if err != nil {
		// 处理特定错误
		if errors.Is(err, crawler.ErrInvalidDate) {
			c.JSON(http.StatusBadRequest, api.Fail(400, "invalid date format, expected YYYY-MM-DD"))
			return
		}
		if errors.Is(err, crawler.ErrInvalidDateRange) {
			c.JSON(http.StatusBadRequest, api.Fail(400, "start_date must not be after end_date"))
			return
		}

		var missingErr *crawler.MissingTopicsError
		if errors.As(err, &missingErr) {
			c.JSON(http.StatusBadRequest, api.FailWithData(400, "some topics are missing from dictionary", missingErr.Topics))
			return
		}

		c.JSON(http.StatusInternalServerError, api.Fail(500, "rebuild failed: "+err.Error()))
		return
	}

	// 转换为响应类型
	topics := make([]response.RebuildTopicInfoResp, 0, len(result.Topics))
	for _, t := range result.Topics {
		topics = append(topics, response.RebuildTopicInfoResp{
			Name:           t.Name,
			NormalizedName: t.NormalizedName,
			StocksCount:    t.StocksCount,
		})
	}

	c.JSON(http.StatusOK, api.OK(response.RebuildTopicRelationsResp{
		StartDate:      result.StartDate,
		EndDate:        result.EndDate,
		TopicsCount:    result.TopicsCount,
		RelationsCount: result.RelationsCount,
		Topics:         topics,
	}))
}
