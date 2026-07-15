package handler

import (
	"net/http"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

// getReturnAnalysisParams GetReturnAnalysis のリクエストパラメータ
type getReturnAnalysisParams struct {
	symbol    string
	from      *time.Time
	to        *time.Time
	benchmark string
}

type ReturnAnalysisHandler struct {
	usecase    usecase.ReturnAnalysisInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewReturnAnalysisHandler(u usecase.ReturnAnalysisInteractor, h driver.HTTPServer, l *zap.Logger) *ReturnAnalysisHandler {
	return &ReturnAnalysisHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

// validateGetReturnAnalysisParams GetReturnAnalysis のリクエストパラメータをバリデーションする
func (h *ReturnAnalysisHandler) validateGetReturnAnalysisParams(r *http.Request) (*getReturnAnalysisParams, error) {
	params := &getReturnAnalysisParams{}

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

	from, to, err := parseDateRange(r)
	if err != nil {
		return nil, err
	}
	params.from = from
	params.to = to

	// benchmark クエリパラメータ: "nikkei"（デフォルト）または "topix"
	benchmark := h.httpServer.GetQueryParam(r, "benchmark")
	if benchmark == "" {
		benchmark = models.BenchmarkNikkei
	}
	if benchmark != models.BenchmarkNikkei && benchmark != models.BenchmarkTopix {
		return nil, &validationError{message: "benchmarkは\"nikkei\"または\"topix\"である必要があります"}
	}
	params.benchmark = benchmark

	return params, nil
}

func (h *ReturnAnalysisHandler) GetReturnAnalysis(w http.ResponseWriter, r *http.Request) {
	params, err := h.validateGetReturnAnalysisParams(r)
	if err != nil {
		writeError(w, h.logger, "failed to validate get return analysis params", err)
		return
	}

	result, err := h.usecase.GetReturnAnalysis(r.Context(), params.symbol, params.from, params.to, params.benchmark)
	if err != nil {
		writeError(w, h.logger, "failed to get return analysis", err)
		return
	}

	respondJSON(w, h.logger, result)
}
