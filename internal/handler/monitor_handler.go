package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"stock/internal/service/monitor"
	"stock/model/api"
)

type MonitorHandler struct{}

func NewMonitorHandler() *MonitorHandler {
	return &MonitorHandler{}
}

func (h *MonitorHandler) Start(c *gin.Context) {
	svc := monitor.GetInstance()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "monitor not initialized"))
		return
	}
	if svc.IsRunning() {
		c.JSON(http.StatusOK, api.OK("monitor already running"))
		return
	}
	go svc.Start(c.Request.Context())
	c.JSON(http.StatusOK, api.OK("monitor started"))
}

func (h *MonitorHandler) Stop(c *gin.Context) {
	svc := monitor.GetInstance()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "monitor not initialized"))
		return
	}
	if !svc.IsRunning() {
		c.JSON(http.StatusOK, api.OK("monitor not running"))
		return
	}
	svc.Stop()
	c.JSON(http.StatusOK, api.OK("monitor stopped"))
}

func (h *MonitorHandler) Tick(c *gin.Context) {
	svc := monitor.GetInstance()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "monitor not initialized"))
		return
	}
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	if err := svc.ManualTick(c.Request.Context(), date); err != nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, api.OK("tick completed"))
}

func (h *MonitorHandler) Status(c *gin.Context) {
	svc := monitor.GetInstance()
	if svc == nil {
		c.JSON(http.StatusInternalServerError, api.Fail(500, "monitor not initialized"))
		return
	}
	status := svc.GetStatus()
	c.JSON(http.StatusOK, api.OK(status))
}
