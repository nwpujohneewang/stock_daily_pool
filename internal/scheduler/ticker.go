package scheduler

import (
	"context"
	"time"

	"stock/internal/pkg/logger"
	"stock/internal/pkg/utils"
	"stock/internal/service"

	"go.uber.org/zap"
)

type Ticker struct {
	cron           *cronWrapper
	monitorService service.MonitorServiceInterface
	intervalSec    int
}

type cronWrapper struct{}

func (c *cronWrapper) Now() time.Time {
	return time.Now()
}

func NewTicker(
	intervalSec int,
	monitorService service.MonitorServiceInterface,
) *Ticker {
	return &Ticker{
		monitorService: monitorService,
		intervalSec:    intervalSec,
	}
}

func (t *Ticker) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(t.intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if !t.isTradingTime() {
				continue
			}
			if err := t.monitorService.ProcessTick(ctx, time.Now().Format("2006-01-02")); err != nil {
				logger.Warn("process tick failed", zap.Error(err))
			}
		}
	}
}

func (t *Ticker) isTradingTime() bool {
	now := time.Now()
	hour := now.Hour()
	min := now.Minute()

	if !utils.IsTradingDay(now) {
		return false
	}

	if hour < 9 || hour >= 15 {
		return false
	}

	if hour == 9 && min < 25 {
		return false
	}

	return true
}

func (t *Ticker) Stop() error {
	return nil
}
