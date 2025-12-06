package driver

import (
	"go.uber.org/zap"
)

func NewLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	// 必要に応じて設定を変更
	return config.Build()
}
