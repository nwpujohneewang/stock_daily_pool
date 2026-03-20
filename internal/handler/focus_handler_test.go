package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestFocusHandler_Get_NoDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/focus", nil)

	handler := &FocusHandler{}
	handler.Get(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFocusHandler_Get_WithDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/focus?date=2026-03-19", nil)

	handler := &FocusHandler{}
	handler.Get(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFocusHandler_Set_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("POST", "/focus", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	handler := &FocusHandler{}
	handler.Set(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFocusHandler_Delete_NoTopicID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("DELETE", "/focus/abc", nil)

	handler := &FocusHandler{}
	handler.Delete(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFocusHandler_Delete_ValidTopicID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("DELETE", "/focus/1", nil)

	handler := &FocusHandler{}
	handler.Delete(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSynonymHandler_List_NoTopicID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{}

	c.Request, _ = http.NewRequest("GET", "/topics/abc/synonyms", nil)

	handler := &SynonymHandler{}
	handler.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSynonymHandler_List_ValidTopicID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "id", Value: "1"}}

	c.Request, _ = http.NewRequest("GET", "/topics/1/synonyms", nil)

	handler := &SynonymHandler{}
	handler.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSynonymHandler_Create_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request, _ = http.NewRequest("POST", "/topics/1/synonyms", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	handler := &SynonymHandler{}
	handler.Create(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSynonymHandler_Delete_NoID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{}

	c.Request, _ = http.NewRequest("DELETE", "/topics/synonyms/abc", nil)

	handler := &SynonymHandler{}
	handler.Delete(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
