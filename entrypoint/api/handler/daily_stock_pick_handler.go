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
	dailyStockPickDatesDefaultLimit = 90
	dailyStockPickDatesMaxLimit     = 400
)

type DailyStockPickHandler struct {
	usecase    usecase.DailyStockPickInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewDailyStockPickHandler(u usecase.DailyStockPickInteractor, s driver.HTTPServer, l *zap.Logger) *DailyStockPickHandler {
	return &DailyStockPickHandler{usecase: u, httpServer: s, logger: l}
}

// GetDailyStockPicks GET /daily-stock-picks?date=YYYY-MM-DD
// date 省略時は最新の pick_date にフォールバックする。該当日が無くても 200 で空を返す。
func (h *DailyStockPickHandler) GetDailyStockPicks(w http.ResponseWriter, r *http.Request) {
	date, err := h.httpServer.GetQueryParamDate(r, "date", util.DateLayout)
	if err != nil {
		http.Error(w, "dateの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	day, err := h.usecase.GetDay(r.Context(), date)
	if err != nil {
		writeError(w, h.logger, "daily stock picks get day failed", err)
		return
	}
	respondJSON(w, h.logger, day)
}

// GetDailyStockPickDates GET /daily-stock-picks/dates?limit=90
func (h *DailyStockPickHandler) GetDailyStockPickDates(w http.ResponseWriter, r *http.Request) {
	limit := dailyStockPickDatesDefaultLimit
	if ls := r.URL.Query().Get("limit"); ls != "" {
		v, err := strconv.Atoi(ls)
		if err != nil {
			http.Error(w, "limit は整数で指定してください", http.StatusBadRequest)
			return
		}
		if v <= 0 || v > dailyStockPickDatesMaxLimit {
			http.Error(w, "limit は 1 以上 400 以下で指定してください", http.StatusBadRequest)
			return
		}
		limit = v
	}

	dates, err := h.usecase.GetPickDates(r.Context(), limit)
	if err != nil {
		writeError(w, h.logger, "daily stock picks get dates failed", err)
		return
	}
	respondJSON(w, h.logger, dates)
}

// GetDailyStockPickStats GET /daily-stock-picks/stats?from=&to=&score_version=
// score_version 省略時は現行のスコア定義バージョンで絞る（定義の異なる推奨を混ぜて集計しないため）。
func (h *DailyStockPickHandler) GetDailyStockPickStats(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		writeError(w, h.logger, "daily stock picks stats invalid date range", err)
		return
	}

	scoreVersion := h.httpServer.GetQueryParam(r, "score_version")

	stats, err := h.usecase.GetStats(r.Context(), from, to, scoreVersion)
	if err != nil {
		writeError(w, h.logger, "daily stock picks get stats failed", err)
		return
	}
	respondJSON(w, h.logger, stats)
}
