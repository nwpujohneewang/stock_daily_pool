package scheduler

import (
	"context"
	"log"
	"stock/internal/pkg/utils"
	"stock/internal/service/snapshot"
	"time"
)

type DailyTask struct {
	logger *log.Logger
}

func NewDailyTask() *DailyTask {
	return &DailyTask{
		logger: log.Default(),
	}
}

func (d *DailyTask) PreMarketInit(ctx context.Context) error {
	d.logger.Println("running pre-market init")
	return nil
}

func (d *DailyTask) CacheWarmup(ctx context.Context) error {
	d.logger.Println("running cache warmup")
	return nil
}

func (d *DailyTask) JiuyanSync(ctx context.Context) error {
	d.logger.Println("running jiuyan sync")
	return nil
}

func (d *DailyTask) ConceptSync(ctx context.Context) error {
	d.logger.Println("running concept sync")
	return nil
}

func (d *DailyTask) ClosingSnapshot(ctx context.Context) error {
	d.logger.Println("running closing snapshot")
	date := time.Now().Format("2006-01-02")
	svc := snapshot.NewSnapshotService()
	return svc.TakeSnapshot(ctx, date)
}

func (d *DailyTask) IsTradingDay() bool {
	return utils.IsTradingDay(time.Now())
}
