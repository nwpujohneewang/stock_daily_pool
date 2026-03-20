package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	resp := Response{
		Code:    0,
		Message: "ok",
		Data:    nil,
	}
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ok", resp.Message)
}

func TestOK(t *testing.T) {
	data := map[string]string{"key": "value"}
	resp := OK(data)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "ok", resp.Message)
	assert.Equal(t, data, resp.Data)
}

func TestFail(t *testing.T) {
	resp := Fail(500, "internal error")
	assert.Equal(t, 500, resp.Code)
	assert.Equal(t, "internal error", resp.Message)
}

func TestPoolHandler_GetLimitUp_NoDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/pool/limit-up", nil)

	handler := &PoolHandler{}
	handler.GetLimitUp(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPoolHandler_GetLimitUp_WithDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/pool/limit-up?date=2026-03-19", nil)

	handler := &PoolHandler{}
	handler.GetLimitUp(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPoolHandler_GetAbove5_NoDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/pool/above5", nil)

	handler := &PoolHandler{}
	handler.GetAbove5(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPoolHandler_GetAbove5_WithDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/pool/above5?date=2026-03-19", nil)

	handler := &PoolHandler{}
	handler.GetAbove5(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAlertHandler_GetTodayAlerts_NoDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/alerts", nil)

	handler := &AlertHandler{}
	handler.GetTodayAlerts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAlertHandler_GetTodayAlerts_WithDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/alerts?date=2026-03-19", nil)

	handler := &AlertHandler{}
	handler.GetTodayAlerts(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTopicHandler_List_NoParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/topics", nil)

	handler := &TopicHandler{}
	handler.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTopicHandler_List_WithKeyword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/topics?keyword=AI", nil)

	handler := &TopicHandler{}
	handler.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestTopicHandler_List_WithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/topics?page=2&page_size=25", nil)

	handler := &TopicHandler{}
	handler.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
