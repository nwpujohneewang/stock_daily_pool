package dal_model

import "time"

// JiuyanRawData 韭研公社原始爬取数据
type JiuyanRawData struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Date          time.Time `gorm:"column:date;not null"`
	TopicName     string    `gorm:"column:topic_name;not null"`
	ActionFieldID string    `gorm:"column:action_field_id"`
	StockCode     string    `gorm:"column:stock_code;not null"`
	StockName     string    `gorm:"column:stock_name;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (JiuyanRawData) TableName() string {
	return "jiuyan_raw_data"
}
