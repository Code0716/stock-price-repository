package handler

import (
	"net/http"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

// StrategyRankingHandler GET /strategy-ranking のハンドラ。
type StrategyRankingHandler struct {
	usecase    usecase.StrategyRankingInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewStrategyRankingHandler(u usecase.StrategyRankingInteractor, h driver.HTTPServer, l *zap.Logger) *StrategyRankingHandler {
	return &StrategyRankingHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

func (h *StrategyRankingHandler) GetStrategyRanking(w http.ResponseWriter, r *http.Request) {
	result, err := h.usecase.GetStrategyRanking(r.Context())
	if err != nil {
		h.logger.Error("failed to get strategy ranking", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, result)
}
