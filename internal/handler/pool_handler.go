package handler

import (
	"errors"
	"fmt"
	"net/http"
	"stock/dal/cache"
	"stock/dal/repo"
	"stock/internal/service/limitdetail"
	"stock/internal/service/pool"
	"stock/model/api"
	"stock/model/api/response"
	"stock/model/dal_model"
	"time"

	"github.com/gin-gonic/gin"
)

type PoolHandler struct{}

func NewPoolHandler() *PoolHandler {
	return &PoolHandler{}
}

func (h *PoolHandler) GetLimitUp(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	svc := pool.NewPoolService()
	results, err := svc.GetLimitUp(ctx, pool.PoolQueryParams{Date: date})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "failed to get limit-up pool"))
		return
	}

	// Get limit details from cache
	var limitDetails map[string]*cache.LimitDetail
	limitDetailSvc := limitdetail.GetInstance()
	if limitDetailSvc != nil {
		limitDetails, _ = limitDetailSvc.GetAllLimitDetails(ctx, date)
	}

	items := make([]response.PoolItem, 0, len(results))
	for _, r := range results {
		item := response.PoolItem{
			TsCode:       r.TsCode,
			Name:         r.Name,
			Price:        r.Price,
			PreClose:     r.PreClose,
			ChangePct:    r.ChangePct,
			PoolType:     int(r.PoolType),
			LimitUpPrice: r.LimitUpPrice,
			Date:         r.Date,
		}

		// Merge limit detail if available
		if detail, ok := limitDetails[r.TsCode]; ok && detail != nil {
			item.FirstTime = detail.FirstTime
			item.LastTime = detail.LastTime
			item.LimitTimes = detail.LimitTimes
			if detail.LimitTimes > 0 {
				item.LimitTimesDisplay = fmt.Sprintf("%d天%d板", detail.LimitTimes, detail.LimitTimes)
			}
		}

		items = append(items, item)
	}

	c.JSON(http.StatusOK, api.OK(items))
}

func (h *PoolHandler) GetAbove5(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", "2026-03-18")

	svc := pool.NewPoolService()
	results, err := svc.GetAbove5(ctx, pool.PoolQueryParams{Date: date})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "failed to get above-5 pool"))
		return
	}

	items := make([]response.PoolItem, 0, len(results))
	for _, r := range results {
		items = append(items, response.PoolItem{
			TsCode:       r.TsCode,
			Name:         r.Name,
			Price:        r.Price,
			PreClose:     r.PreClose,
			ChangePct:    r.ChangePct,
			PoolType:     int(r.PoolType),
			LimitUpPrice: r.LimitUpPrice,
			Date:         r.Date,
		})
	}

	c.JSON(http.StatusOK, api.OK(items))
}

func (h *PoolHandler) ReclassifyByDate(c *gin.Context) {
	ctx := c.Request.Context()
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	svc := pool.NewPoolService()
	result, err := svc.Reclassify(ctx, date)
	if err != nil {
		if errors.Is(err, pool.ErrDataNotReady) {
			c.JSON(http.StatusOK, api.Fail(4001, "实时数据尚未就绪，请稍后重试"))
			return
		}
		if errors.Is(err, pool.ErrHistoricalSnapshotNotGenerated) {
			c.JSON(http.StatusOK, api.Fail(4002, "该日期未生成分类快照"))
			return
		}
		c.JSON(http.StatusInternalServerError, api.Fail(500, "reclassify failed"))
		return
	}

	// Get limit details from cache
	var limitDetails map[string]*cache.LimitDetail
	limitDetailSvc := limitdetail.GetInstance()
	if limitDetailSvc != nil {
		limitDetails, _ = limitDetailSvc.GetAllLimitDetails(ctx, date)
	}

	// Collect all ts_codes for batch query
	allCodes := make([]string, 0, len(result.Items)+len(result.YesterdayStrongItems))
	for _, r := range result.Items {
		allCodes = append(allCodes, r.TsCode)
	}
	for _, r := range result.YesterdayStrongItems {
		allCodes = append(allCodes, r.TsCode)
	}

	isHistorical := date != time.Now().Format("2006-01-02")

	// Query all topic relations for display only for today-mode results.
	var allRelations map[string][]dal_model.StockTopicRelation
	if len(allCodes) > 0 {
		relationRepo := repo.NewStockTopicRelationRepository()
		allRelations, _ = relationRepo.GetByTsCodeBatch(ctx, allCodes)
	}

	items := make([]response.ReclassifyItem, 0, len(result.Items))
	for _, r := range result.Items {
		item := response.ReclassifyItem{
			TsCode:                r.TsCode,
			Name:                  r.Name,
			Price:                 r.Price,
			PreClose:              r.PreClose,
			ChangePct:             r.ChangePct,
			PoolType:              r.PoolType,
			LimitUpPrice:          r.LimitUpPrice,
			IsLimitUp:             r.IsLimitUp,
			IsAbove5Pct:           r.IsAbove5Pct,
			BoardCode:             r.BoardCode,
			Topics:                r.Topics,
			ClassifyLayer:         r.ClassifyLayer,
			Confidence:            r.Confidence,
			Skipped:               r.Skipped,
			SkipReason:            r.SkipReason,
			ConsecutiveStrongDays: int(r.ConsecutiveStrongDays),
			TotalMv:               r.TotalMv,
			Vol:                   r.Vol,
			Amount:                r.Amount,
		}

		// Merge limit detail if available (only for limit-up stocks)
		if r.IsLimitUp {
			if r.LimitTimes > 0 {
				item.LimitTimes = int(r.LimitTimes)
				item.LimitTimesDisplay = fmt.Sprintf("%d天%d板", r.LimitTimes, r.LimitTimes)
			} else if detail, ok := limitDetails[r.TsCode]; ok && detail != nil {
				item.FirstTime = detail.FirstTime
				item.LastTime = detail.LastTime
				item.LimitTimes = detail.LimitTimes
				if detail.LimitTimes > 0 {
					item.LimitTimesDisplay = fmt.Sprintf("%d天%d板", detail.LimitTimes, detail.LimitTimes)
				}
			}
		}

		// Add all topic relations

		if relations, ok := allRelations[r.TsCode]; ok && len(relations) > 0 {
			allTopics := make([]dal_model.TopicRelation, 0, len(relations))
			for _, rel := range relations {
				allTopics = append(allTopics, dal_model.TopicRelation{
					TopicID:      rel.TopicID,
					TopicName:    rel.TopicName,
					Category:     rel.Category,
					Source:       rel.Source,
					HitCount:     rel.HitCount,
					LastSeenDate: rel.LastSeenDate.Format("2006-01-02"),
				})
			}
			item.AllTopics = allTopics
		}

		items = append(items, item)
	}

	yesterdayStrongItems := make([]response.ReclassifyItem, 0, len(result.YesterdayStrongItems))
	for _, r := range result.YesterdayStrongItems {
		item := response.ReclassifyItem{
			TsCode:                r.TsCode,
			Name:                  r.Name,
			Price:                 r.Price,
			PreClose:              r.PreClose,
			ChangePct:             r.ChangePct,
			PoolType:              r.PoolType,
			BoardCode:             r.BoardCode,
			YesterdayChangePct:    r.YesterdayChangePct,
			IsYesterdayStrong:     r.IsYesterdayStrong,
			ConsecutiveStrongDays: int(r.ConsecutiveStrongDays),
			LimitTimes:            int(r.LimitTimes),
			TotalMv:               r.TotalMv,
		}

		// Add all topic relations for yesterday strong items too
		if !isHistorical {
			if relations, ok := allRelations[r.TsCode]; ok && len(relations) > 0 {
				allTopics := make([]dal_model.TopicRelation, 0, len(relations))
				for _, rel := range relations {
					allTopics = append(allTopics, dal_model.TopicRelation{
						TopicID:      rel.TopicID,
						TopicName:    rel.TopicName,
						Category:     rel.Category,
						Source:       rel.Source,
						HitCount:     rel.HitCount,
						LastSeenDate: rel.LastSeenDate.Format("2006-01-02"),
						UpdatedAt:    rel.UpdatedAt.Format(time.RFC3339),
					})
				}
				item.AllTopics = allTopics
			}
		}

		yesterdayStrongItems = append(yesterdayStrongItems, item)
	}

	c.JSON(http.StatusOK, api.OK(response.ReclassifyResp{
		Items:                items,
		YesterdayStrongItems: yesterdayStrongItems,
		IsTrading:            result.IsTrading,
	}))
}
