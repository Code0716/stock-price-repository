package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// バックテストのイグジット既定値
var (
	defaultTakeProfit  = decimal.NewFromFloat(0.10)
	defaultStopLoss    = decimal.NewFromFloat(0.05)
	defaultMaxHoldDays = 20
)

type getBacktestParams struct {
	symbol string
	from   *time.Time
	to     *time.Time
	params models.BacktestParams
}

type BacktestHandler struct {
	usecase    usecase.BacktestInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewBacktestHandler(u usecase.BacktestInteractor, h driver.HTTPServer, l *zap.Logger) *BacktestHandler {
	return &BacktestHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

// parsePositiveRate 0より大きく1以下のレート文字列をパースする。空なら既定値。
func parsePositiveRate(raw string, def decimal.Decimal) (decimal.Decimal, bool) {
	if raw == "" {
		return def, true
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || f <= 0 || f > 1 {
		return decimal.Zero, false
	}
	return decimal.NewFromFloat(f), true
}

// parseNonNegativeRate 0以上0.05以下のレート文字列をパースする。空なら0（コストなし）。
// commission / slippage などコストパラメータ用（ゼロ値は後方互換でコストなし）。
func parseNonNegativeRate(raw string) (decimal.Decimal, bool) {
	if raw == "" {
		return decimal.Zero, true
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || f < 0 || f > 0.05 {
		return decimal.Zero, false
	}
	return decimal.NewFromFloat(f), true
}

func (h *BacktestHandler) validateGetBacktestParams(r *http.Request) (*getBacktestParams, error) {
	p := &getBacktestParams{}

	p.symbol = h.httpServer.GetQueryParam(r, "symbol")
	if p.symbol == "" {
		return nil, &validationError{message: "シンボルは必須です"}
	}
	if len(p.symbol) > 10 {
		return nil, &validationError{message: "シンボルが長すぎます"}
	}
	if !alphanumericRequiredRegex.MatchString(p.symbol) {
		return nil, &validationError{message: "シンボルは英数字である必要があります"}
	}

	from, err := h.httpServer.GetQueryParamDate(r, "from", util.DateLayout)
	if err != nil {
		return nil, &validationError{message: "fromの日付形式が不正です"}
	}
	p.from = from

	to, err := h.httpServer.GetQueryParamDate(r, "to", util.DateLayout)
	if err != nil {
		return nil, &validationError{message: "toの日付形式が不正です"}
	}
	p.to = to

	takeProfit, ok := parsePositiveRate(h.httpServer.GetQueryParam(r, "takeProfit"), defaultTakeProfit)
	if !ok {
		return nil, &validationError{message: "takeProfitは0より大きく1以下である必要があります"}
	}
	stopLoss, ok := parsePositiveRate(h.httpServer.GetQueryParam(r, "stopLoss"), defaultStopLoss)
	if !ok {
		return nil, &validationError{message: "stopLossは0より大きく1以下である必要があります"}
	}

	maxHoldDays := defaultMaxHoldDays
	if raw := h.httpServer.GetQueryParam(r, "maxHoldDays"); raw != "" {
		d, err := strconv.Atoi(raw)
		if err != nil || d <= 0 || d > 250 {
			return nil, &validationError{message: "maxHoldDaysは1〜250である必要があります"}
		}
		maxHoldDays = d
	}

	commissionRate, ok := parseNonNegativeRate(h.httpServer.GetQueryParam(r, "commission"))
	if !ok {
		return nil, &validationError{message: "commissionは0以上0.05以下である必要があります"}
	}
	slippageRate, ok := parseNonNegativeRate(h.httpServer.GetQueryParam(r, "slippage"))
	if !ok {
		return nil, &validationError{message: "slippageは0以上0.05以下である必要があります"}
	}

	p.params = models.BacktestParams{
		TakeProfit:     takeProfit,
		StopLoss:       stopLoss,
		MaxHoldDays:    maxHoldDays,
		CommissionRate: commissionRate,
		SlippageRate:   slippageRate,
	}
	return p, nil
}

func (h *BacktestHandler) GetBacktest(w http.ResponseWriter, r *http.Request) {
	p, err := h.validateGetBacktestParams(r)
	if err != nil {
		if verr, ok := err.(*validationError); ok {
			http.Error(w, verr.message, http.StatusBadRequest)
			return
		}
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	result, err := h.usecase.GetBacktestComparison(r.Context(), p.symbol, p.from, p.to, p.params)
	if err != nil {
		h.logger.Error("failed to get backtest", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	respondJSON(w, h.logger, result)
}
