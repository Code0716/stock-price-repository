package handler

import (
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
)

const (
	dailyAvoidStockDatesDefaultLimit = 90
	dailyAvoidStockDatesMaxLimit     = 400
)

type DailyAvoidStockHandler struct {
	usecase    usecase.DailyAvoidStockInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewDailyAvoidStockHandler(u usecase.DailyAvoidStockInteractor, s driver.HTTPServer, l *zap.Logger) *DailyAvoidStockHandler {
	return &DailyAvoidStockHandler{usecase: u, httpServer: s, logger: l}
}

// GetDailyAvoidStocks GET /daily-avoid-stocks?date=YYYY-MM-DD
// date 省略時は最新の as_of_date にフォールバックする。該当日が無くても 200 で空を返す。
func (h *DailyAvoidStockHandler) GetDailyAvoidStocks(w http.ResponseWriter, r *http.Request) {
	date, err := h.httpServer.GetQueryParamDate(r, "date", util.DateLayout)
	if err != nil {
		http.Error(w, "dateの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	day, err := h.usecase.GetDay(r.Context(), date)
	if err != nil {
		writeError(w, h.logger, "daily avoid stocks get day failed", err)
		return
	}
	respondJSON(w, h.logger, day)
}

// GetDailyAvoidStockDates GET /daily-avoid-stocks/dates?limit=90
func (h *DailyAvoidStockHandler) GetDailyAvoidStockDates(w http.ResponseWriter, r *http.Request) {
	limit := dailyAvoidStockDatesDefaultLimit
	if ls := r.URL.Query().Get("limit"); ls != "" {
		v, err := strconv.Atoi(ls)
		if err != nil {
			http.Error(w, "limit は整数で指定してください", http.StatusBadRequest)
			return
		}
		if v <= 0 || v > dailyAvoidStockDatesMaxLimit {
			http.Error(w, "limit は 1 以上 400 以下で指定してください", http.StatusBadRequest)
			return
		}
		limit = v
	}

	dates, err := h.usecase.GetAsOfDates(r.Context(), limit)
	if err != nil {
		writeError(w, h.logger, "daily avoid stocks get dates failed", err)
		return
	}
	respondJSON(w, h.logger, dates)
}
