package handler

import (
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/usecase/daytrade"
	"github.com/Code0716/stock-price-repository/util"
)

type DaytradeHandler struct {
	usecase    usecase.DaytradeInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewDaytradeHandler(u usecase.DaytradeInteractor, s driver.HTTPServer, l *zap.Logger) *DaytradeHandler {
	return &DaytradeHandler{usecase: u, httpServer: s, logger: l}
}

func (h *DaytradeHandler) ImportSBICsv(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "無効なマルチパートフォームです", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `"file"フィールドが見つかりません`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	result, err := h.usecase.ImportSBICsv(r.Context(), file)
	if err != nil {
		if errors.Is(err, daytrade.ErrParse) {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		h.logger.Error("daytrade import failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, result)
}

type daytradeSummaryResponse struct {
	Granularity string                       `json:"granularity"`
	From        *string                      `json:"from"`
	To          *string                      `json:"to"`
	Buckets     []*models.DaytradeSummaryBucket `json:"buckets"`
}

func (h *DaytradeHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	granularityStr := h.httpServer.GetQueryParam(r, "granularity")
	if granularityStr == "" {
		granularityStr = "daily"
	}
	g := models.DaytradeSummaryGranularity(granularityStr)
	if !g.Valid() {
		http.Error(w, "granularityはdaily/monthly/yearly/allのいずれかである必要があります", http.StatusBadRequest)
		return
	}

	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		http.Error(w, "fromの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		http.Error(w, "toの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if from != nil && to != nil && from.After(*to) {
		http.Error(w, "fromはto以前の日付である必要があります", http.StatusBadRequest)
		return
	}

	buckets, err := h.usecase.GetSummary(r.Context(), from, to, g)
	if err != nil {
		h.logger.Error("daytrade summary failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	resp := &daytradeSummaryResponse{
		Granularity: granularityStr,
		Buckets:     buckets,
	}
	if from != nil {
		s := from.Format(util.DateLayout)
		resp.From = &s
	}
	if to != nil {
		s := to.Format(util.DateLayout)
		resp.To = &s
	}
	respondJSON(w, h.logger, resp)
}

type daytradeExecutionsResponse struct {
	Date       string                    `json:"date"`
	Executions []*models.DaytradeExecution `json:"executions"`
}

func (h *DaytradeHandler) GetExecutionsByDate(w http.ResponseWriter, r *http.Request) {
	dateStr := h.httpServer.GetQueryParam(r, "date")
	if dateStr == "" {
		http.Error(w, "dateパラメータは必須です", http.StatusBadRequest)
		return
	}
	date, err := time.ParseInLocation(util.DateLayout, dateStr, time.Local)
	if err != nil {
		http.Error(w, "dateの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	executions, err := h.usecase.GetExecutionsByDate(r.Context(), date)
	if err != nil {
		h.logger.Error("daytrade executions by date failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	respondJSON(w, h.logger, &daytradeExecutionsResponse{
		Date:       dateStr,
		Executions: executions,
	})
}

type daytradeSymbolSummaryResponse struct {
	From  *string                        `json:"from"`
	To    *string                        `json:"to"`
	Items []*models.DaytradeSymbolSummary `json:"items"`
}

func (h *DaytradeHandler) GetSummaryByTickerSymbol(w http.ResponseWriter, r *http.Request) {
	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		http.Error(w, "fromの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		http.Error(w, "toの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if from != nil && to != nil && from.After(*to) {
		http.Error(w, "fromはto以前の日付である必要があります", http.StatusBadRequest)
		return
	}

	items, err := h.usecase.GetSummaryByTickerSymbol(r.Context(), from, to)
	if err != nil {
		h.logger.Error("daytrade summary by ticker symbol failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	resp := &daytradeSymbolSummaryResponse{Items: items}
	if from != nil {
		s := from.Format(util.DateLayout)
		resp.From = &s
	}
	if to != nil {
		s := to.Format(util.DateLayout)
		resp.To = &s
	}
	respondJSON(w, h.logger, resp)
}

func (h *DaytradeHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		http.Error(w, "fromの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		http.Error(w, "toの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if from != nil && to != nil && from.After(*to) {
		http.Error(w, "fromはto以前の日付である必要があります", http.StatusBadRequest)
		return
	}

	stats, err := h.usecase.GetPeriodStats(r.Context(), from, to)
	if err != nil {
		h.logger.Error("daytrade stats failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, stats)
}

func (h *DaytradeHandler) GetInsights(w http.ResponseWriter, r *http.Request) {
	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		http.Error(w, "fromの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		http.Error(w, "toの日付形式が不正です (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if from != nil && to != nil && from.After(*to) {
		http.Error(w, "fromはto以前の日付である必要があります", http.StatusBadRequest)
		return
	}

	insights, err := h.usecase.GetInsights(r.Context(), from, to)
	if err != nil {
		h.logger.Error("daytrade insights failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	respondJSON(w, h.logger, insights)
}

type daytradeRangeResponse struct {
	Min *string `json:"min"`
	Max *string `json:"max"`
}

func (h *DaytradeHandler) GetCoveredRange(w http.ResponseWriter, r *http.Request) {
	minDate, maxDate, err := h.usecase.GetCoveredRange(r.Context())
	if err != nil {
		h.logger.Error("daytrade covered range failed", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	resp := &daytradeRangeResponse{}
	if minDate != nil {
		s := minDate.Format(util.DateLayout)
		resp.Min = &s
	}
	if maxDate != nil {
		s := maxDate.Format(util.DateLayout)
		resp.Max = &s
	}
	respondJSON(w, h.logger, resp)
}
