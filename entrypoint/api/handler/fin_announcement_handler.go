package handler

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
)

type FinAnnouncementHandler struct {
	usecase    usecase.StockBrandInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewFinAnnouncementHandler(u usecase.StockBrandInteractor, h driver.HTTPServer, l *zap.Logger) *FinAnnouncementHandler {
	return &FinAnnouncementHandler{usecase: u, httpServer: h, logger: l}
}

type FinAnnouncementResponse struct {
	ID               string `json:"id"`
	TickerSymbol     string `json:"tickerSymbol"`
	AnnouncementDate string `json:"announcementDate"`
	FiscalYear       string `json:"fiscalYear"`
	FiscalQuarter    string `json:"fiscalQuarter"`
	Sector17Code     string `json:"sector17Code"`
	Sector33Code     string `json:"sector33Code"`
}

type GetFinAnnouncementsResponse struct {
	Announcements []*FinAnnouncementResponse `json:"announcements"`
	Pagination    *AnalyzeHistoryPaginationInfo `json:"pagination"`
}

func toFinAnnouncementResponse(a *models.FinAnnouncement) *FinAnnouncementResponse {
	return &FinAnnouncementResponse{
		ID:               a.ID,
		TickerSymbol:     a.TickerSymbol,
		AnnouncementDate: a.AnnouncementDate.Format("2006-01-02"),
		FiscalYear:       a.FiscalYear,
		FiscalQuarter:    a.FiscalQuarter,
		Sector17Code:     a.Sector17Code,
		Sector33Code:     a.Sector33Code,
	}
}

// GetFinAnnouncements GET /fin-announcements
func (h *FinAnnouncementHandler) GetFinAnnouncements(w http.ResponseWriter, r *http.Request) {
	filter := &models.FinAnnouncementFilter{}

	symbol := h.httpServer.GetQueryParam(r, "symbol")
	if symbol != "" {
		if len(symbol) > 10 {
			http.Error(w, "symbolが長すぎます", http.StatusBadRequest)
			return
		}
		filter.TickerSymbol = symbol
	}

	fromStr := h.httpServer.GetQueryParam(r, "from")
	if fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			http.Error(w, "fromはYYYY-MM-DD形式で指定してください", http.StatusBadRequest)
			return
		}
		filter.From = t
	}

	toStr := h.httpServer.GetQueryParam(r, "to")
	if toStr != "" {
		t, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			http.Error(w, "toはYYYY-MM-DD形式で指定してください", http.StatusBadRequest)
			return
		}
		filter.To = t
	}

	page, err := parseBoundedInt(h.httpServer, r, "page", 1, 0)
	if err != nil {
		if verr, ok := err.(*validationError); ok {
			http.Error(w, verr.message, http.StatusBadRequest)
			return
		}
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	filter.Page = page

	limit, err := parseBoundedInt(h.httpServer, r, "limit", 100, 1000)
	if err != nil {
		if verr, ok := err.(*validationError); ok {
			http.Error(w, verr.message, http.StatusBadRequest)
			return
		}
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}
	filter.Limit = limit

	result, err := h.usecase.GetFinAnnouncements(r.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get fin announcements", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	announcements := make([]*FinAnnouncementResponse, 0, len(result.Announcements))
	for _, a := range result.Announcements {
		announcements = append(announcements, toFinAnnouncementResponse(a))
	}

	respondJSON(w, h.logger, &GetFinAnnouncementsResponse{
		Announcements: announcements,
		Pagination: &AnalyzeHistoryPaginationInfo{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// GetNextFinAnnouncement GET /fin-announcements/next?symbol=XXXX
func (h *FinAnnouncementHandler) GetNextFinAnnouncement(w http.ResponseWriter, r *http.Request) {
	symbol := h.httpServer.GetQueryParam(r, "symbol")
	if symbol == "" {
		http.Error(w, "symbolは必須です", http.StatusBadRequest)
		return
	}

	result, err := h.usecase.GetNextFinAnnouncement(r.Context(), symbol)
	if err != nil {
		h.logger.Error("failed to get next fin announcement", zap.Error(err))
		http.Error(w, "内部サーバーエラー", http.StatusInternalServerError)
		return
	}

	if result == nil {
		respondJSON(w, h.logger, map[string]interface{}{"announcement": nil})
		return
	}
	respondJSON(w, h.logger, map[string]interface{}{"announcement": toFinAnnouncementResponse(result)})
}
