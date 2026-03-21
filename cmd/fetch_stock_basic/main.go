package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"stock/dal/db"
	"stock/internal/service"

	"stock/config"
	"stock/internal/external/tushare"
)

func main() {
	cfg := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	if err := config.Init(*cfg); err != nil {
		fmt.Fprintf(os.Stderr, "init config failed: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	client := tushare.NewClient(&config.GlobalConfig.Tushare, config.GlobalConfig.Retry)

	db.Init()
	err := service.NewStockService(client).SyncStockBasic(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync stock basic failed: %v\n", err)
		os.Exit(1)
	}

}
