package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlertService_New(t *testing.T) {
	svc := &AlertService{}
	assert.NotNil(t, svc)
}

func TestStockService_New(t *testing.T) {
	svc := &StockService{}
	assert.NotNil(t, svc)
}

func TestSnapshotService_New(t *testing.T) {
	svc := &SnapshotService{}
	assert.NotNil(t, svc)
}

func TestCrawlerService_New(t *testing.T) {
	svc := &CrawlerService{}
	assert.NotNil(t, svc)
}

func TestConceptSyncService_New(t *testing.T) {
	svc := &ConceptSyncService{}
	assert.NotNil(t, svc)
}

func TestMonitorService_New(t *testing.T) {
	svc := &MonitorService{}
	assert.NotNil(t, svc)
}

func TestClassifyService_New(t *testing.T) {
	svc := &ClassifyService{}
	assert.NotNil(t, svc)
}
