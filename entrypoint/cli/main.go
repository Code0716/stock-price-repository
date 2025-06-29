package main

import (
	"context"
	"log"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/di"

	"go.uber.org/zap"
)

func main() {
	config.LoadEnvConfig()
	ctx := context.Background()
	cli, cleanup, err := di.InitializeCli(ctx)
	if err != nil {
		log.Fatal(ctx, "failed to initialize cli", zap.Error(err))
	}
	defer cleanup()

	cli.Run()
}
