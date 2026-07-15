package handler

import (
	"net/http"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
	"go.uber.org/zap"
)

type SignalPerformanceHandler struct {
	usecase    usecase.SignalPerformanceInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewSignalPerformanceHandler(u usecase.SignalPerformanceInteractor, h driver.HTTPServer, l *zap.Logger) *SignalPerformanceHandler {
	return &SignalPerformanceHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

func (h *SignalPerformanceHandler) validateParams(r *http.Request) (*models.SignalPerformanceFilter, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	toParam, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		return nil, &validationError{message: "toの日付形式が不正です（YYYY-MM-DD）"}
	}
	to := today
	if toParam != nil {
		to = *toParam
	}

	fromParam, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		return nil, &validationError{message: "fromの日付形式が不正です（YYYY-MM-DD）"}
	}
	from := to.AddDate(0, 0, -90)
	if fromParam != nil {
		from = *fromParam
	}

	if from.After(to) {
		return nil, &validationError{message: "fromはtoより前の日付を指定してください"}
	}
	if to.Sub(from).Hours()/24 > 366 {
		return nil, &validationError{message: "期間は最大366日以内で指定してください"}
	}

	return &models.SignalPerformanceFilter{
		From:   from,
		To:     to,
		Method: h.httpServer.GetQueryParam(r, "method"),
		Action: h.httpServer.GetQueryParam(r, "action"),
	}, nil
}

// GetSignalPerformance GET /signal-performance
func (h *SignalPerformanceHandler) GetSignalPerformance(w http.ResponseWriter, r *http.Request) {
	filter, err := h.validateParams(r)
	if err != nil {
		writeError(w, h.logger, "failed to validate signal performance params", err)
		return
	}

	result, err := h.usecase.GetSignalPerformance(r.Context(), filter)
	if err != nil {
		writeError(w, h.logger, "failed to get signal performance", err)
		return
	}

	respondJSON(w, h.logger, result)
}
