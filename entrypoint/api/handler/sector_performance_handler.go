package handler

import (
	"net/http"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/usecase"
	"github.com/Code0716/stock-price-repository/util"
	"go.uber.org/zap"
)

// SectorPerformanceHandler GET /sector-performance のハンドラー
type SectorPerformanceHandler struct {
	usecase    usecase.SectorPerformanceInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewSectorPerformanceHandler(u usecase.SectorPerformanceInteractor, h driver.HTTPServer, l *zap.Logger) *SectorPerformanceHandler {
	return &SectorPerformanceHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

type sectorPerformanceParams struct {
	from        time.Time
	to          time.Time
	granularity string
}

func (h *SectorPerformanceHandler) validateParams(r *http.Request) (*sectorPerformanceParams, error) {
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

	granularity := r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "33"
	}
	if granularity != "33" && granularity != "17" {
		return nil, &validationError{message: "granularityは\"33\"または\"17\"を指定してください"}
	}

	return &sectorPerformanceParams{from: from, to: to, granularity: granularity}, nil
}

// GetSectorPerformance GET /sector-performance
func (h *SectorPerformanceHandler) GetSectorPerformance(w http.ResponseWriter, r *http.Request) {
	params, err := h.validateParams(r)
	if err != nil {
		writeError(w, h.logger, "failed to validate sector performance params", err)
		return
	}

	result, err := h.usecase.GetSectorPerformance(r.Context(), params.from, params.to, params.granularity)
	if err != nil {
		writeError(w, h.logger, "failed to get sector performance", err)
		return
	}

	respondJSON(w, h.logger, result)
}
