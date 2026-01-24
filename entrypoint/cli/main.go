package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/di"

	"go.uber.org/zap"
)

func main() {
	config.LoadEnvConfig()
	ctx := context.Background()
	if loc, err := time.LoadLocation("Asia/Tokyo"); err == nil {
		time.Local = loc
	}

	cli, cleanup, err := di.InitializeCli(ctx)
	if err != nil {
		log.Fatal(ctx, "failed to initialize cli", zap.Error(err))
	}
	defer cleanup()

	if err := cli.Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
