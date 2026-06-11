package handler

import (
	"net/http"
	"strconv"

	"github.com/Code0716/stock-price-repository/domain_service"
	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

const (
	strategyRankingStocksDefaultLimit = 100
	strategyRankingStocksMaxLimit     = 2000
)

// StrategyRankingHandler GET /strategy-ranking および GET /strategy-ranking-stocks のハンドラ。
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

// GetStrategyRankingStocks GET /strategy-ranking-stocks?strategy=<id>&limit=<n>
func (h *StrategyRankingHandler) GetStrategyRankingStocks(w http.ResponseWriter, r *http.Request) {
	strategy := r.URL.Query().Get("strategy")
	if strategy == "" {
		http.Error(w, "strategy パラメータは必須です", http.StatusBadRequest)
		return
	}
	if !isValidStrategy(strategy) {
		http.Error(w, "strategy が不正です", http.StatusBadRequest)
		return
	}

	limit := strategyRankingStocksDefaultLimit
	if ls := r.URL.Query().Get("limit"); ls != "" {
		v, err := strconv.Atoi(ls)
		if err != nil {
			http.Error(w, "limit は整数で指定してください", http.StatusBadRequest)
			return
		}
		if v <= 0 || v > strategyRankingStocksMaxLimit {
			http.Error(w, "limit は 1 以上 2000 以下で指定してください", http.StatusBadRequest)
			return
		}
		limit = v
	}

	result, err := h.usecase.GetStrategyRankingStocks(r.Context(), strategy, limit)
	if err != nil {
		h.logger.Error("failed to get strategy ranking stocks", zap.Error(err), zap.String("strategy", strategy))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, result)
}

// isValidStrategy 戦略 ID が既知の5定数のいずれかか確認する。
func isValidStrategy(strategy string) bool {
	for _, s := range domain_service.StrategyOrder {
		if s == strategy {
			return true
		}
	}
	return false
}
