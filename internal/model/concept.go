package model

import (
	"time"
)

type TushareConcept struct {
	ID          int64     `gorm:"column:id"            json:"id"`
	ConceptCode string    `gorm:"column:concept_code"  json:"concept_code"`
	ConceptName string    `gorm:"column:concept_name"  json:"concept_name"`
	Source      string    `gorm:"column:source"        json:"source"`
	CreatedAt   time.Time `gorm:"column:created_at"    json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"    json:"updated_at"`
}

func (TushareConcept) TableName() string { return "tushare_concepts" }

type TopicConcept struct {
	ID          int64     `gorm:"column:id"            json:"id"`
	ConceptName string    `gorm:"column:concept_name"  json:"concept_name"`
	ConceptCode string    `gorm:"column:concept_code"  json:"concept_code"`
	TopicID     int64     `gorm:"column:topic_id"      json:"topic_id"`
	MatchType   string    `gorm:"column:match_type"    json:"match_type"`
	CreatedAt   time.Time `gorm:"column:created_at"    json:"created_at"`
}

func (TopicConcept) TableName() string { return "topic_concepts" }

type TushareConceptDetail struct {
	ID          int64     `gorm:"column:id"            json:"id"`
	TsCode      string    `gorm:"column:ts_code"       json:"ts_code"`
	ConceptName string    `gorm:"column:concept_name"  json:"concept_name"`
	ConceptCode string    `gorm:"column:concept_code"  json:"concept_code"`
	Source      string    `gorm:"column:source"        json:"source"`
	CreatedAt   time.Time `gorm:"column:created_at"    json:"created_at"`
}

func (TushareConceptDetail) TableName() string { return "tushare_concept_details" }
