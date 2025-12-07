package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Code0716/stock-price-repository/pb"
	"github.com/Code0716/stock-price-repository/usecase"
)

// StockServiceServer implements pb.StockServiceServer
type StockServiceServer struct {
	pb.UnimplementedStockServiceServer
	getHighVolumeStockBrandsUseCase usecase.GetHighVolumeStockBrandsUseCase
}

func NewStockServiceServer(
	getHighVolumeStockBrandsUseCase usecase.GetHighVolumeStockBrandsUseCase,
) *StockServiceServer {
	return &StockServiceServer{
		getHighVolumeStockBrandsUseCase: getHighVolumeStockBrandsUseCase,
	}
}

func (s *StockServiceServer) GetHighVolumeStockBrands(
	ctx context.Context,
	req *pb.GetHighVolumeStockBrandsRequest,
) (*pb.GetHighVolumeStockBrandsResponse, error) {
	// Use pagination-aware use case
	result, err := s.getHighVolumeStockBrandsUseCase.ExecuteWithPagination(
		ctx,
		req.GetSymbolFrom(),
		int(req.GetLimit()),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get high volume stock brands: %v", err)
	}

	// Convert domain models to protobuf messages
	pbBrands := make([]*pb.HighVolumeStockBrand, 0, len(result.Brands))
	for _, brand := range result.Brands {
		pbBrands = append(pbBrands, &pb.HighVolumeStockBrand{
			StockBrandId:  brand.StockBrandID,
			TickerSymbol:  brand.TickerSymbol,
			CompanyName:   brand.CompanyName,
			VolumeAverage: brand.VolumeAverage,
			CreatedAt:     brand.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Build pagination info if limit was specified
	var pagination *pb.PaginationInfo
	if result.Limit > 0 {
		nextCursor := ""
		if result.NextCursor != nil {
			nextCursor = *result.NextCursor
		}
		pagination = &pb.PaginationInfo{
			NextCursor: nextCursor,
			Limit:      int32(result.Limit),
		}
	}

	return &pb.GetHighVolumeStockBrandsResponse{
		Brands:     pbBrands,
		Pagination: pagination,
	}, nil
}
