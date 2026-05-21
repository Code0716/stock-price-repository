package models

import (
	pb "github.com/Code0716/stock-price-repository/pb"
)

func StockBrandToProto(m *StockBrand) *pb.StockBrand {
	return &pb.StockBrand{
		Id:               m.ID,
		TickerSymbol:     m.TickerSymbol,
		Name:             m.Name,
		MarketCode:       m.MarketCode,
		MarketName:       m.MarketName,
		Sector33Code:     m.Sector33Code,
		Sector33CodeName: m.Sector33CodeName,
		Sector17Code:     m.Sector17Code,
		Sector17CodeName: m.Sector17CodeName,
		CreatedAt:        m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func StockBrandFromProto(p *pb.StockBrand) *StockBrand {
	return &StockBrand{
		ID:               p.Id,
		TickerSymbol:     p.TickerSymbol,
		Name:             p.Name,
		MarketCode:       p.MarketCode,
		MarketName:       p.MarketName,
		Sector33Code:     p.Sector33Code,
		Sector33CodeName: p.Sector33CodeName,
		Sector17Code:     p.Sector17Code,
		Sector17CodeName: p.Sector17CodeName,
	}
}
