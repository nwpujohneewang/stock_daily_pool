package service

import (
	"context"
	"fmt"
	"stock/dal/redis"
	"time"

	"stock/config"
	"stock/dal/db"
	"stock/internal/model"
)

type AlertService struct {
	cfg    *config.MonitorConfig
	logger *logger
}

type logger struct{}

func (l logger) Error(msg string, keys ...interface{}) { fmt.Println("[ERROR]", msg, keys) }
func (l logger) Warn(msg string, keys ...interface{})  { fmt.Println("[WARN]", msg, keys) }
func (l logger) Info(msg string, keys ...interface{})  { fmt.Println("[INFO]", msg, keys) }

func NewAlertService(cfg *config.MonitorConfig) *AlertService {
	return &AlertService{
		cfg:    cfg,
		logger: &logger{},
	}
}

type AlertCheckInput struct {
	TsCode         string
	StockName      string
	TopicIDs       []int64
	TopicNames     []string
	TriggerPrice   float64
	TriggerTime    time.Time
	Date           string
	ChangePct      float64
	PrevDayPct     float64
	PrevDayLimitUp bool
	PrevDayAbove5  bool
}

type AlertCheckOutput struct {
	ShouldAlert    bool
	AlertTopicID   int64
	AlertTopicName string
	RejectReason   string
	Message        string
}

func (s *AlertService) CheckAndAlert(ctx context.Context, input AlertCheckInput) (*AlertCheckOutput, error) {
	focusedTopicID, focusedTopicName := s.checkFocus(ctx, input)
	if focusedTopicID == 0 {
		return &AlertCheckOutput{ShouldAlert: false, RejectReason: "topic_not_focused"}, nil
	}

	if !input.PrevDayAbove5 {
		return &AlertCheckOutput{ShouldAlert: false, RejectReason: "not_in_prev_day_5pct_pool"}, nil
	}

	if input.PrevDayLimitUp {
		return &AlertCheckOutput{ShouldAlert: false, RejectReason: "was_limit_up_prev_day"}, nil
	}

	triggerMin := input.TriggerTime.Hour()*60 + input.TriggerTime.Minute()
	if triggerMin < 9*60+25 || triggerMin > 15*60 {
		return &AlertCheckOutput{ShouldAlert: false, RejectReason: "outside_trading_hours"}, nil
	}

	alertRepo := db.NewAlertRepository()
	alertKey := fmt.Sprintf("%s:%d", input.TsCode, focusedTopicID)
	alreadyAlerted, err := alertRepo.IsAlerted(ctx, input.Date, alertKey)
	if err != nil {
		return nil, fmt.Errorf("check alerted: %w", err)
	}
	if alreadyAlerted {
		return &AlertCheckOutput{ShouldAlert: false, RejectReason: "already_alerted_today"}, nil
	}

	if err := alertRepo.MarkAlerted(ctx, input.Date, alertKey); err != nil {
		return nil, fmt.Errorf("mark alerted: %w", err)
	}

	alert := &model.StrategyAlert{
		Date:        time.Now(),
		TsCode:      input.TsCode,
		StockName:   input.StockName,
		TopicID:     &focusedTopicID,
		TopicName:   &focusedTopicName,
		AlertType:   1,
		TriggerTime: &input.TriggerTime,
		PrevDayPct:  &input.PrevDayPct,
	}

	if _, err := alertRepo.Create(ctx, alert); err != nil {
		return nil, fmt.Errorf("create alert: %w", err)
	}

	message := fmt.Sprintf("%s 首次涨停！属于今日关注热点【%s】，昨日涨幅 %.1f%%", input.StockName, focusedTopicName, input.PrevDayPct)
	return &AlertCheckOutput{
		ShouldAlert:    true,
		AlertTopicID:   focusedTopicID,
		AlertTopicName: focusedTopicName,
		Message:        message,
	}, nil
}

func (s *AlertService) checkFocus(ctx context.Context, input AlertCheckInput) (int64, string) {
	focusCache := redis.NewFocusCache()
	for i, tid := range input.TopicIDs {
		isFocused, err := focusCache.IsFocused(ctx, input.Date, tid)
		if err == nil && isFocused {
			return tid, input.TopicNames[i]
		}
	}
	return 0, ""
}

func (s *AlertService) GetTodayAlerts(ctx context.Context, date string) ([]model.StrategyAlert, error) {
	alertRepo := db.NewAlertRepository()
	return alertRepo.GetByDate(ctx, date)
}

func (s *AlertService) GetHistoryAlerts(ctx context.Context, startDate, endDate string, topicID *int64, page, pageSize int) ([]model.StrategyAlert, int64, error) {
	alertRepo := db.NewAlertRepository()
	return alertRepo.GetHistory(ctx, startDate, endDate, topicID, page, pageSize)
}
