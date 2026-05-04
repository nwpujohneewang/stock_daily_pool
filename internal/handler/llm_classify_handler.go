package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"stock/config"
	"stock/dal/cache"
	"stock/dal/repo"
	extushare "stock/external/tushare"
	"stock/internal/pkg/logger"
	"stock/internal/service/monitor"
	"stock/model/api"
	"stock/model/api/response"
	"stock/model/dal_model"
)

type LLMClassifyHandler struct{}

func NewLLMClassifyHandler() *LLMClassifyHandler {
	return &LLMClassifyHandler{}
}

func noDataResp(date string) response.LLMRunHistoryResp {
	return response.LLMRunHistoryResp{Status: "no_data", Date: date, Results: []response.LLMClassifyItem{}}
}

// RunHistory fetches historical quotes from Tushare and runs LLM classification
// synchronously without requiring a pre-generated snapshot.
// Only limit-up stocks and stocks with >5% gain are classified.
func (h *LLMClassifyHandler) RunHistory(c *gin.Context) {
	ctx := c.Request.Context()

	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, api.Fail(400, "date is required (format: YYYY-MM-DD)"))
		return
	}

	monitorSvc := monitor.GetInstance()
	if monitorSvc == nil || monitorSvc.GetLLMClassifyService() == nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "LLM service not configured"))
		return
	}
	llmSvc := monitorSvc.GetLLMClassifyService()

	cfg := config.GlobalConfig
	tushareClient := extushare.NewClient(&cfg.Tushare, cfg.Retry)
	tradeDate := strings.ReplaceAll(date, "-", "")
	start := time.Now()

	// Step 1: All daily closing quotes
	rawQuotes, err := tushareClient.DailyAll(ctx, tradeDate)
	if err != nil {
		logger.Warn("llm run-history: fetch daily failed", zap.String("date", date), zap.Error(err))
		c.JSON(http.StatusInternalServerError, api.Fail(500, "fetch daily quotes failed: "+err.Error()))
		return
	}
	if len(rawQuotes) == 0 {
		c.JSON(http.StatusOK, api.OK(noDataResp(date)))
		return
	}

	// Step 2: Limit-up set
	limitItems, err := tushareClient.LimitListD(ctx, tradeDate)
	if err != nil {
		logger.Warn("llm run-history: fetch limit list failed", zap.Error(err))
	}
	limitUpSet := make(map[string]bool, len(limitItems))
	for _, item := range limitItems {
		limitUpSet[item.TsCode] = true
	}

	// Step 3: Pre-filter to limit-up stocks and stocks with >5% gain
	// Also filter out ST stocks and Beijing Stock Exchange stocks.
	quotes := make([]*dal_model.StockQuote, 0, len(rawQuotes))
	codes := make([]string, 0, len(rawQuotes))
	for _, q := range rawQuotes {
		if !limitUpSet[q.TsCode] && q.PctChg < 5.0 {
			continue
		}
		if len(q.TsCode) >= 1 {
			c := q.TsCode[0]
			if c == '8' || c == '4' || c == '9' {
				continue
			}
		}
		quotes = append(quotes, &dal_model.StockQuote{
			TsCode:   q.TsCode,
			PreClose: q.PreClose,
			Price:    q.Price,
			PctChg:   q.PctChg,
			Vol:      q.Vol,
			Amount:   q.Amount,
		})
		codes = append(codes, q.TsCode)
	}
	if len(quotes) == 0 {
		c.JSON(http.StatusOK, api.OK(noDataResp(date)))
		return
	}

	// Step 4: Stock basic info only for filtered stocks
	stockRepo := repo.NewStockRepository()
	stocks, err := stockRepo.GetByTsCodes(ctx, codes)
	if err != nil {
		logger.Warn("llm run-history: fetch stocks failed", zap.Error(err))
	}
	stockMap := make(map[string]*dal_model.StockBasicInfo, len(stocks))
	for i := range stocks {
		stockMap[stocks[i].TsCode] = &stocks[i]
	}

	// Filter out ST stocks
	filteredQuotes := make([]*dal_model.StockQuote, 0, len(quotes))
	for _, q := range quotes {
		if info, ok := stockMap[q.TsCode]; ok && info.IsST {
			continue
		}
		filteredQuotes = append(filteredQuotes, q)
	}
	quotes = filteredQuotes

	// Step 5: Warm up stock-topic relation cache from DB for filtered stocks
	topicRelRepo := repo.NewStockTopicRelationRepository()
	if err := topicRelRepo.WarmupByTsCodes(ctx, codes); err != nil {
		logger.Warn("llm run-history: warmup stock topics failed", zap.Error(err))
		// non-fatal: classification proceeds with empty topics
	}

	// Step 6: Synchronous LLM classification on filtered quotes
	if err := llmSvc.RunClassification(ctx, date, quotes, stockMap, limitUpSet); err != nil {
		logger.Warn("llm run-history: classification error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, api.Fail(500, "llm classification failed: "+err.Error()))
		return
	}
	elapsedMs := time.Since(start).Milliseconds()

	// Step 7: Read results from LLM cache and build response
	llmCache := cache.NewLLMClassificationCache()
	results, hit, _ := llmCache.GetClassificationResult(ctx, date)
	if !hit || len(results) == 0 {
		c.JSON(http.StatusOK, api.OK(response.LLMRunHistoryResp{
			Status: "completed", Date: date, ElapsedMs: elapsedMs, Total: 0, Results: []response.LLMClassifyItem{},
		}))
		return
	}

	items := make([]response.LLMClassifyItem, 0, len(results))
	for tsCode, topics := range results {
		if len(topics) == 0 {
			continue
		}
		name := tsCode
		if info, ok := stockMap[tsCode]; ok {
			name = info.Name
		}
		items = append(items, response.LLMClassifyItem{
			TsCode:     tsCode,
			Name:       name,
			TopicName:  topics[0].TopicName,
			TopicID:    topics[0].TopicID,
			Category:   topics[0].Category,
			Confidence: topics[0].Confidence,
		})
	}

	c.JSON(http.StatusOK, api.OK(response.LLMRunHistoryResp{
		Status:    "completed",
		Date:      date,
		ElapsedMs: elapsedMs,
		Total:     len(items),
		Results:   items,
	}))
}
