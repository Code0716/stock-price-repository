package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
)

type StockPriceHandler struct {
	usecase    usecase.StockBrandsDailyPriceInteractor
	httpServer driver.HTTPServer
}

func NewStockPriceHandler(u usecase.StockBrandsDailyPriceInteractor, h driver.HTTPServer) *StockPriceHandler {
	return &StockPriceHandler{
		usecase:    u,
		httpServer: h,
	}
}

func (h *StockPriceHandler) GetDailyPrices(w http.ResponseWriter, r *http.Request) {
	symbol := h.httpServer.GetQueryParam(r, "symbol")
	if symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		http.Error(w, "invalid from date format", http.StatusBadRequest)
		return
	}

	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		http.Error(w, "invalid to date format", http.StatusBadRequest)
		return
	}

	prices, err := h.usecase.GetDailyStockPrices(r.Context(), symbol, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(prices); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
