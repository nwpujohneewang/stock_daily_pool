package dal_model

import (
	"encoding/json"
	"time"
)

type ClassificationAuditLog struct {
	ID               int64           `gorm:"column:id"                  json:"id"`
	Date             time.Time       `gorm:"column:date"                json:"date"`
	TsCode           string          `gorm:"column:ts_code"             json:"ts_code"`
	TopicID          *int64          `gorm:"column:topic_id"            json:"topic_id,omitempty"`
	ClassifyLayer    string          `gorm:"column:classify_layer"      json:"classify_layer"`
	Strategy         string          `gorm:"column:strategy"            json:"strategy"`
	CandidateScores  json.RawMessage `gorm:"column:candidate_scores"    json:"candidate_scores,omitempty"`
	EvidenceText     *string         `gorm:"column:evidence_text"       json:"evidence_text,omitempty"`
	Confidence       *float64        `gorm:"column:confidence"         json:"confidence,omitempty"`
	CorrectedTopicID *int64          `gorm:"column:corrected_topic_id"  json:"corrected_topic_id,omitempty"`
	CreatedAt        time.Time       `gorm:"column:created_at"         json:"created_at"`
}

func (ClassificationAuditLog) TableName() string { return "classification_audit_log" }

type MarketSnapshot struct {
	ID         int64           `gorm:"column:id"          json:"id"`
	Date       time.Time       `gorm:"column:date"        json:"date"`
	Status     int16           `gorm:"column:status"      json:"status"`
	RawData    json.RawMessage `gorm:"column:raw_data"    json:"raw_data,omitempty"`
	TopicCount int             `gorm:"column:topic_count" json:"topic_count"`
	StockCount int             `gorm:"column:stock_count" json:"stock_count"`
	RetryCount int             `gorm:"column:retry_count" json:"retry_count"`
	ErrorMsg   *string         `gorm:"column:error_msg"   json:"error_msg,omitempty"`
	CreatedAt  time.Time       `gorm:"column:created_at"  json:"created_at"`
	UpdatedAt  time.Time       `gorm:"column:updated_at"  json:"updated_at"`
}

func (MarketSnapshot) TableName() string { return "market_snapshots" }

const (
	SnapshotStatusPending = 0
	SnapshotStatusSuccess = 1
	SnapshotStatusFailed  = 2
)

type ClassifyLayer string

const (
	LayerL1Redis     ClassifyLayer = "L1_REDIS"
	LayerL2PGJiuyan  ClassifyLayer = "L2_PG_JIUYAN"
	LayerL3PGConcept ClassifyLayer = "L3_PG_CONCEPT"
	LayerL4LLM       ClassifyLayer = "L4_LLM"
)

type ClassifyStrategy string

const (
	StrategyManual             ClassifyStrategy = "MANUAL"
	StrategyJiuyanAttribution  ClassifyStrategy = "JIUYAN_ATTRIBUTION"
	StrategyConceptAttribution ClassifyStrategy = "CONCEPT_ATTRIBUTION"
	StrategyLLMV1              ClassifyStrategy = "LLM_V1"
)
