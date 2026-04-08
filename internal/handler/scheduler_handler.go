package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"stock/internal/scheduler"
	"stock/model/api"
)

type SchedulerHandler struct {
	scheduler *scheduler.Scheduler
}

func NewSchedulerHandler(s *scheduler.Scheduler) *SchedulerHandler {
	return &SchedulerHandler{scheduler: s}
}

func (h *SchedulerHandler) ensureScheduler(c *gin.Context) *scheduler.Scheduler {
	if h.scheduler == nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "scheduler not initialized"))
		return nil
	}
	return h.scheduler
}

func (h *SchedulerHandler) TriggerPreMarketInit(c *gin.Context) {
	s := h.ensureScheduler(c)
	if s == nil {
		return
	}
	s.TriggerPreMarketInit()
	c.JSON(http.StatusOK, api.OK("pre-market init triggered"))
}

func (h *SchedulerHandler) TriggerLoadYesterdayStrongPool(c *gin.Context) {
	s := h.ensureScheduler(c)
	if s == nil {
		return
	}
	s.TriggerLoadYesterdayStrongPool()
	c.JSON(http.StatusOK, api.OK("load yesterday strong pool triggered"))
}

func (h *SchedulerHandler) TriggerClosingSnapshot(c *gin.Context) {
	s := h.ensureScheduler(c)
	if s == nil {
		return
	}
	s.TriggerClosingSnapshot()
	c.JSON(http.StatusOK, api.OK("closing snapshot triggered"))
}

func (h *SchedulerHandler) TriggerSyncStockBasic(c *gin.Context) {
	s := h.ensureScheduler(c)
	if s == nil {
		return
	}
	s.TriggerSyncStockBasic()
	c.JSON(http.StatusOK, api.OK("sync stock basic triggered"))
}
