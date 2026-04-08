package handler

import (
	"context"
	"net/http"
	"stock/dal/repo"
	"stock/internal/pkg/utils"
	"stock/internal/service/snapshot"
	"stock/model/api"
	"time"

	"github.com/gin-gonic/gin"
)

type SnapshotHandler struct{}

func NewSnapshotHandler() *SnapshotHandler {
	return &SnapshotHandler{}
}

type TriggerSnapshotRequest struct {
	Date string `json:"date"`
}

func (h *SnapshotHandler) TriggerSnapshot(c *gin.Context) {
	var req TriggerSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body provided, use today's date
		loc, _ := time.LoadLocation("Asia/Shanghai")
		req.Date = time.Now().In(loc).Format("2006-01-02")
	}

	if req.Date == "" {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		req.Date = time.Now().In(loc).Format("2006-01-02")
	}

	snapshotSvc := snapshot.NewSnapshotService()
	if err := snapshotSvc.TakeSnapshot(c.Request.Context(), req.Date); err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "snapshot failed: "+err.Error()))
		return
	}

	cleanupCutoff, cleanupErr := cleanupSnapshotsByBaseDate(c.Request.Context(), req.Date)

	c.JSON(http.StatusOK, api.OK(gin.H{
		"message": "snapshot completed",
		"date":    req.Date,
		"cleanup": gin.H{
			"cutoff_date": cleanupCutoff,
			"error":       cleanupErr,
		},
	}))
}

type TriggerSnapshotRangeRequest struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// TriggerSnapshotRange triggers snapshots for a date range (inclusive), skipping non-trading days.
// Returns a summary of processed, skipped and failed dates.
func (h *SnapshotHandler) TriggerSnapshotRange(c *gin.Context) {
	var req TriggerSnapshotRangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid request body"))
		return
	}
	if req.StartDate == "" || req.EndDate == "" {
		c.JSON(http.StatusBadRequest, api.Fail(400, "start_date and end_date are required"))
		return
	}
	loc, _ := time.LoadLocation("Asia/Shanghai")
	start, err := time.ParseInLocation("2006-01-02", req.StartDate, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid start_date format, expect YYYY-MM-DD"))
		return
	}
	end, err := time.ParseInLocation("2006-01-02", req.EndDate, loc)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Fail(400, "invalid end_date format, expect YYYY-MM-DD"))
		return
	}
	if end.Before(start) {
		c.JSON(http.StatusBadRequest, api.Fail(400, "end_date must be >= start_date"))
		return
	}

	snapshotSvc := snapshot.NewSnapshotService()
	processed := make([]string, 0)
	skipped := make([]string, 0)
	failed := make([]string, 0)

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if !utils.IsTradingDay(d) {
			skipped = append(skipped, d.Format("2006-01-02"))
			continue
		}
		dateStr := d.Format("2006-01-02")
		if err := snapshotSvc.TakeSnapshot(c.Request.Context(), dateStr); err != nil {
			failed = append(failed, dateStr)
			continue
		}
		processed = append(processed, dateStr)
	}

	cleanupCutoff, cleanupErr := cleanupSnapshotsByBaseDate(c.Request.Context(), req.EndDate)

	c.JSON(http.StatusOK, api.OK(gin.H{
		"requested_range": gin.H{
			"start_date": req.StartDate,
			"end_date":   req.EndDate,
		},
		"processed_dates": processed,
		"skipped_dates":   skipped,
		"failed_dates":    failed,
		"cleanup": gin.H{
			"cutoff_date": cleanupCutoff,
			"error":       cleanupErr,
		},
	}))
}

func cleanupSnapshotsByBaseDate(ctx context.Context, baseDate string) (string, string) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t, err := time.ParseInLocation("2006-01-02", baseDate, loc)
	if err != nil {
		return "", "invalid base date"
	}
	cutoff := t.AddDate(0, 0, -30).Format("2006-01-02")

	snapshotRepo := repo.NewSnapshotRepository()
	if err := snapshotRepo.DeleteBeforeOrEqualDate(ctx, cutoff); err != nil {
		return cutoff, err.Error()
	}
	return cutoff, ""
}
