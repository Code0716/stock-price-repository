package handler

import (
	"net/http"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

// ValuationHandler GET /valuation のハンドラ。
type ValuationHandler struct {
	usecase    usecase.ValuationInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewValuationHandler(u usecase.ValuationInteractor, h driver.HTTPServer, l *zap.Logger) *ValuationHandler {
	return &ValuationHandler{usecase: u, httpServer: h, logger: l}
}

func (h *ValuationHandler) GetValuation(w http.ResponseWriter, r *http.Request) {
	symbol := h.httpServer.GetQueryParam(r, "symbol")
	if symbol == "" {
		http.Error(w, "シンボルは必須です", http.StatusBadRequest)
		return
	}
	if len(symbol) > 10 {
		http.Error(w, "シンボルが長すぎます", http.StatusBadRequest)
		return
	}
	if !alphanumericRequiredRegex.MatchString(symbol) {
		http.Error(w, "シンボルは英数字である必要があります", http.StatusBadRequest)
		return
	}

	result, err := h.usecase.GetValuation(r.Context(), symbol)
	if err != nil {
		writeError(w, h.logger, "failed to get valuation", err)
		return
	}
	respondJSON(w, h.logger, result)
}
