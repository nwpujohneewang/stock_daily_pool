package handler

import (
	"net/http"
	"stock/config"
	"stock/dal/db"
	"stock/internal/external/tushare"
	"stock/internal/pkg/limiter"
	"stock/internal/service"
	"stock/model/dal_model"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"stock/internal/ws"
)

type Handlers struct {
	Hub    *ws.Hub
	Config *config.AppConfig

	PoolHandler
	TopicHandler
	AlertHandler
	SynonymHandler
	FocusHandler
	EvidenceHandler
	ConceptHandler
	WSHandler
}

func NewHandlers(
	hub *ws.Hub,
	cfg *config.AppConfig,
) *Handlers {
	h := &Handlers{
		Hub:    hub,
		Config: cfg,
	}

	h.PoolHandler = *NewPoolHandler()
	h.TopicHandler = *NewTopicHandler()
	h.AlertHandler = *NewAlertHandler()
	h.SynonymHandler = *NewSynonymHandler()
	h.FocusHandler = *NewFocusHandler()
	h.EvidenceHandler = *NewEvidenceHandler()
	h.ConceptHandler = *NewConceptHandler()
	h.WSHandler = *NewWSHandler(hub)

	return h
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func OK(data interface{}) Response {
	return Response{Code: 0, Message: "ok", Data: data}
}

func Fail(code int, message string) Response {
	return Response{Code: code, Message: message}
}

type PoolHandler struct{}

func NewPoolHandler() *PoolHandler {
	return &PoolHandler{}
}

func (h *PoolHandler) GetLimitUp(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", "2026-03-18")

	pools, err := db.NewPoolRepository().GetByDate(ctx, date, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(pools))
}

func (h *PoolHandler) GetAbove5(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", "2026-03-18")

	pools, err := db.NewPoolRepository().GetByDate(ctx, date, 2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(pools))
}

type ReclassifyResult struct {
	TsCode        string                   `json:"ts_code"`
	Name          string                   `json:"name"`
	Price         float64                  `json:"price"`
	PreClose      float64                  `json:"pre_close"`
	ChangePct     float64                  `json:"change_pct"`
	PoolType      int                      `json:"pool_type"`
	LimitUpPrice  float64                  `json:"limit_up_price,omitempty"`
	IsLimitUp     bool                     `json:"is_limit_up"`
	IsAbove5Pct   bool                     `json:"is_above_5pct"`
	BoardCode     string                   `json:"board_code"`
	Topics        []dal_model.TopicMapping `json:"topics,omitempty"`
	ClassifyLayer string                   `json:"classify_layer,omitempty"`
	Confidence    float64                  `json:"confidence,omitempty"`
	Skipped       bool                     `json:"skipped"`
	SkipReason    string                   `json:"skip_reason,omitempty"`
}

func (h *PoolHandler) ReclassifyByDate(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	tradeDate := strings.ReplaceAll(date, "-", "")
	tushareClient := tushare.NewClient(&config.GlobalConfig.Tushare, config.GlobalConfig.Retry)
	classifySvc := service.NewClassifyService()

	quotes, err := tushareClient.DailyAll(ctx, tradeDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, "fetch daily data: "+err.Error()))
		return
	}
	if len(quotes) == 0 {
		c.JSON(http.StatusOK, OK([]ReclassifyResult{}))
		return
	}

	boardRules := map[dal_model.BoardCode]*dal_model.BoardRule{
		dal_model.BoardMain: {BoardCode: dal_model.BoardMain, BoardName: "主板", LimitUpRatio: 0.10},
		dal_model.BoardGEM:  {BoardCode: dal_model.BoardGEM, BoardName: "创业板", LimitUpRatio: 0.20},
		dal_model.BoardSTAR: {BoardCode: dal_model.BoardSTAR, BoardName: "科创板", LimitUpRatio: 0.20},
		dal_model.BoardBSE:  {BoardCode: dal_model.BoardBSE, BoardName: "北交所", LimitUpRatio: 0.30},
	}

	stockRepo := db.NewStockRepository()
	results := make([]ReclassifyResult, 0, len(quotes))

	for _, quote := range quotes {
		stock, err := stockRepo.GetByTsCode(ctx, quote.TsCode)
		if err != nil {
			continue
		}

		board := limiter.DetectBoard(quote.TsCode[:6])
		rule := boardRules[board]

		input := dal_model.DetectInput{
			TsCode:       quote.TsCode,
			StockName:    stock.Name,
			CurrentPrice: quote.Price,
			PreClose:     quote.PreClose,
			ChangePct:    quote.PctChg,
			BoardCode:    board,
			IsST:         stock.IsST,
			PrevState:    dal_model.LimitStateNone,
		}
		output := limiter.DetectLimitUp(input, rule)

		item := ReclassifyResult{
			TsCode:       quote.TsCode,
			Name:         stock.Name,
			Price:        quote.Price,
			PreClose:     quote.PreClose,
			ChangePct:    quote.PctChg,
			BoardCode:    string(board),
			IsLimitUp:    output.IsLimitUp,
			IsAbove5Pct:  output.IsAbove5Pct,
			LimitUpPrice: output.LimitUpPrice,
			PoolType:     0,
			Skipped:      output.Skipped,
			SkipReason:   output.SkipReason,
		}

		if output.Skipped {
			results = append(results, item)
			continue
		}

		if output.IsLimitUp {
			item.PoolType = 1
			topics, err := classifySvc.ClassifyStock(ctx, quote.TsCode, date, time.Now())
			if err == nil && len(topics) > 0 {
				item.Topics = topics
			}
		} else if output.IsAbove5Pct {
			item.PoolType = 2
		}

		results = append(results, item)
	}

	c.JSON(http.StatusOK, OK(results))
}

type TopicHandler struct{}

func NewTopicHandler() *TopicHandler {
	return &TopicHandler{}
}

func (h *TopicHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	topics, total, err := db.NewTopicRepository().List(ctx, keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(map[string]interface{}{
		"items": topics,
		"total": total,
		"page":  page,
		"pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}))
}

type AlertHandler struct{}

func NewAlertHandler() *AlertHandler {
	return &AlertHandler{}
}

func (h *AlertHandler) GetTodayAlerts(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", "2026-03-18")

	alerts, err := db.NewAlertRepository().GetTodayAlerts(ctx, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Fail(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, OK(alerts))
}
