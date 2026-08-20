package handler

import (
	"net/http"
	"time"

	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/usecase"
	"go.uber.org/zap"
)

const (
	defaultNotificationHistoryLimit = 50
	maxNotificationHistoryLimit     = 200
)

type NotificationHistoryPaginationInfo struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type GetNotificationHistoriesResponse struct {
	Notifications []*models.NotificationHistory      `json:"notifications"`
	Pagination    *NotificationHistoryPaginationInfo `json:"pagination"`
}

type NotificationHandler struct {
	usecase    usecase.NotificationHistoryInteractor
	httpServer driver.HTTPServer
	logger     *zap.Logger
}

func NewNotificationHandler(u usecase.NotificationHistoryInteractor, h driver.HTTPServer, l *zap.Logger) *NotificationHandler {
	return &NotificationHandler{
		usecase:    u,
		httpServer: h,
		logger:     l,
	}
}

type getNotificationHistoriesParams struct {
	channelID string
	query     string
	page      int
	limit     int
}

func (h *NotificationHandler) validateGetNotificationHistoriesParams(r *http.Request) (*getNotificationHistoriesParams, error) {
	page, err := parseBoundedInt(h.httpServer, r, "page", 1, 0)
	if err != nil {
		return nil, err
	}

	limit, err := parseBoundedInt(h.httpServer, r, "limit", defaultNotificationHistoryLimit, maxNotificationHistoryLimit)
	if err != nil {
		return nil, err
	}

	return &getNotificationHistoriesParams{
		channelID: h.httpServer.GetQueryParam(r, "channel"),
		query:     h.httpServer.GetQueryParam(r, "q"),
		page:      page,
		limit:     limit,
	}, nil
}

func (h *NotificationHandler) GetNotificationHistories(w http.ResponseWriter, r *http.Request) {
	params, err := h.validateGetNotificationHistoriesParams(r)
	if err != nil {
		writeError(w, h.logger, "failed to validate get notification histories params", err)
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		writeError(w, h.logger, "failed to parse date range", err)
		return
	}
	// sent_at は DATETIME のため、to は日付の終端（翌日00:00の直前）まで含める
	if to != nil {
		endOfDay := to.AddDate(0, 0, 1).Add(-time.Nanosecond)
		to = &endOfDay
	}

	result, err := h.usecase.GetNotificationHistories(r.Context(), &models.NotificationHistoryFilter{
		ChannelID: params.channelID,
		From:      from,
		To:        to,
		Query:     params.query,
		Page:      params.page,
		Limit:     params.limit,
	})
	if err != nil {
		writeError(w, h.logger, "failed to get notification histories", err)
		return
	}

	respondJSON(w, h.logger, &GetNotificationHistoriesResponse{
		Notifications: result.Notifications,
		Pagination: &NotificationHistoryPaginationInfo{
			Page:       result.Page,
			Limit:      result.Limit,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}
