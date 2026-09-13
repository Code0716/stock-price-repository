package handler

import (
	"net/http"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/usecase"
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

	fromParam, toParam, err := parseDateRange(r)
	if err != nil {
		return nil, err
	}

	to := today
	if toParam != nil {
		to = *toParam
	}

	from := to.AddDate(0, 0, -90)
	if fromParam != nil {
		from = *fromParam
	}

	if from.After(to) {
		return nil, &validationError{message: "fromはto以前の日付である必要があります"}
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
