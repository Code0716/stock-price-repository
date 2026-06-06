package handler

import (
	"net/http"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
	"go.uber.org/zap"
)

type getTechnicalIndicatorsParams struct {
	symbol string
	from   *time.Time
	to     *time.Time
}

type TechnicalIndicatorsHandler struct {
	usecase    usecase.TechnicalIndicatorsInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewTechnicalIndicatorsHandler(u usecase.TechnicalIndicatorsInteractor, h driver.HTTPServer, l *zap.Logger) *TechnicalIndicatorsHandler {
	return &TechnicalIndicatorsHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

func (h *TechnicalIndicatorsHandler) validateGetTechnicalIndicatorsParams(r *http.Request) (*getTechnicalIndicatorsParams, error) {
	params := &getTechnicalIndicatorsParams{}

	params.symbol = h.httpServer.GetQueryParam(r, "symbol")
	if params.symbol == "" {
		return nil, &validationError{message: "シンボルは必須です"}
	}
	if len(params.symbol) > 10 {
		return nil, &validationError{message: "シンボルが長すぎます"}
	}
	if !alphanumericRequiredRegex.MatchString(params.symbol) {
		return nil, &validationError{message: "シンボルは英数字である必要があります"}
	}

	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		return nil, &validationError{message: "fromの日付形式が不正です"}
	}
	params.from = from

	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		return nil, &validationError{message: "toの日付形式が不正です"}
	}
	params.to = to

	return params, nil
}

func (h *TechnicalIndicatorsHandler) GetTechnicalIndicators(w http.ResponseWriter, r *http.Request) {
	params, err := h.validateGetTechnicalIndicatorsParams(r)
	if err != nil {
		if verr, ok := err.(*validationError); ok {
			http.Error(w, verr.message, http.StatusBadRequest)
			return
		}
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	result, err := h.usecase.GetTechnicalIndicators(r.Context(), params.symbol, params.from, params.to)
	if err != nil {
		h.logger.Error("failed to get technical indicators", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	respondJSON(w, h.logger, result)
}
