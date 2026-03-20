package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestEvidenceHandler_List_NoTsCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/stock//evidence", nil)

	handler := &EvidenceHandler{}
	handler.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEvidenceHandler_List_ValidTsCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "ts_code", Value: "000001.SZ"}}

	c.Request, _ = http.NewRequest("GET", "/stock/000001.SZ/evidence", nil)

	handler := &EvidenceHandler{}
	handler.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEvidenceHandler_List_WithDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "ts_code", Value: "000001.SZ"}}
	c.Request, _ = http.NewRequest("GET", "/stock/000001.SZ/evidence?date=2026-03-19", nil)

	handler := &EvidenceHandler{}
	handler.List(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEvidenceHandler_Correct_NoTsCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{}

	c.Request, _ = http.NewRequest("PUT", "/stock//evidence/1/correct", nil)

	handler := &EvidenceHandler{}
	handler.Correct(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEvidenceHandler_Correct_InvalidEvidenceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "ts_code", Value: "000001.SZ"}}

	c.Request, _ = http.NewRequest("PUT", "/stock/000001.SZ/evidence/abc/correct", nil)

	handler := &EvidenceHandler{}
	handler.Correct(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConceptHandler_GetTopicConcepts_NoTopicID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request, _ = http.NewRequest("GET", "/topics/abc/concepts", nil)

	handler := &ConceptHandler{}
	handler.ListByTopic(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestConceptHandler_GetTopicConcepts_ValidTopicID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = gin.Params{{Key: "id", Value: "1"}}

	c.Request, _ = http.NewRequest("GET", "/topics/1/concepts", nil)

	handler := &ConceptHandler{}
	handler.ListByTopic(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
