package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// respondJSON JSON レスポンスを返却する共通メソッド（ステータスコードは200固定）
func respondJSON(w http.ResponseWriter, logger *zap.Logger, data interface{}) {
	respondJSONStatus(w, logger, http.StatusOK, data)
}

// respondJSONStatus 指定したステータスコードでJSONレスポンスを返却する共通メソッド
func respondJSONStatus(w http.ResponseWriter, logger *zap.Logger, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
	}
}
