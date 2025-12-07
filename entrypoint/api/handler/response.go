package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// respondJSON JSON レスポンスを返却する共通メソッド
func respondJSON(w http.ResponseWriter, logger *zap.Logger, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
	}
}
