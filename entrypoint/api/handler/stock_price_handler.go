package handler

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
	"go.uber.org/zap"
)

var symbolRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

type StockPriceHandler struct {
	usecase    usecase.StockBrandsDailyPriceInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewStockPriceHandler(u usecase.StockBrandsDailyPriceInteractor, h driver.HTTPServer, l *zap.Logger) *StockPriceHandler {
	return &StockPriceHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

func (h *StockPriceHandler) GetDailyPrices(w http.ResponseWriter, r *http.Request) {
	symbol := h.httpServer.GetQueryParam(r, "symbol")
	if symbol == "" {
		http.Error(w, "シンボルは必須です", http.StatusBadRequest)
		return
	}

	if len(symbol) > 10 {
		http.Error(w, "シンボルが長すぎます", http.StatusBadRequest)
		return
	}

	if !symbolRegex.MatchString(symbol) {
		http.Error(w, "シンボルは英数字である必要があります", http.StatusBadRequest)
		return
	}

	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		http.Error(w, "fromの日付形式が不正です", http.StatusBadRequest)
		return
	}

	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		http.Error(w, "toの日付形式が不正です", http.StatusBadRequest)
		return
	}

	// ソート順の取得（デフォルトは昇順）
	var sortOrder *models.SortOrder
	orderParam := h.httpServer.GetQueryParam(r, "order")
	if orderParam != "" {
		if orderParam != string(models.SortOrderAsc) && orderParam != string(models.SortOrderDesc) {
			http.Error(w, "orderはascまたはdescである必要があります", http.StatusBadRequest)
			return
		}
		order := models.SortOrder(orderParam)
		sortOrder = &order
	}

	prices, err := h.usecase.GetDailyStockPricesWithOrder(r.Context(), symbol, from, to, sortOrder)
	if err != nil {
		h.logger.Error("failed to get daily stock prices", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(prices); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
	}
}
