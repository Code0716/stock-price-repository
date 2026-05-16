package models

import "time"

type FinAnnouncement struct {
	ID               string
	TickerSymbol     string
	StockBrandID     *string
	AnnouncementDate time.Time
	FiscalYear       string
	FiscalQuarter    string
	Sector17Code     string
	Sector33Code     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type FinAnnouncementFilter struct {
	TickerSymbol string
	From         time.Time
	To           time.Time
	Page         int
	Limit        int
}

type PaginatedFinAnnouncements struct {
	Announcements []*FinAnnouncement
	Page          int
	Limit         int
	Total         int64
	TotalPages    int
}
