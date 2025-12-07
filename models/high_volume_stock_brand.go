package models

import "time"

// HighVolumeStockBrand represents a stock brand with high trading volume
type HighVolumeStockBrand struct {
	StockBrandID  string    `json:"stockBrandId"`
	TickerSymbol  string    `json:"tickerSymbol"`
	CompanyName   string    `json:"companyName"`
	VolumeAverage uint64    `json:"volumeAverage"`
	CreatedAt     time.Time `json:"createdAt"`
}

func NewHighVolumeStockBrand(
	stockBrandID string,
	tickerSymbol string,
	companyName string,
	volumeAverage uint64,
	createdAt time.Time,
) *HighVolumeStockBrand {
	return &HighVolumeStockBrand{
		StockBrandID:  stockBrandID,
		TickerSymbol:  tickerSymbol,
		CompanyName:   companyName,
		VolumeAverage: volumeAverage,
		CreatedAt:     createdAt,
	}
}

// PaginatedHighVolumeStockBrands represents paginated high volume stock brands
type PaginatedHighVolumeStockBrands struct {
	Brands     []*HighVolumeStockBrand
	NextCursor *string // nil means this is the last page
	Limit      int
}
