package server

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/models"
	pb "github.com/Code0716/stock-price-repository/pb"
)

func TestStockServiceServer_GetHighVolumeStockBrands(t *testing.T) {
	now := time.Now()

	type fields struct {
		getHighVolumeStockBrandsUseCase func(ctrl *gomock.Controller) *mock_usecase.MockGetHighVolumeStockBrandsUseCase
	}
	type args struct {
		ctx context.Context
		req *pb.GetHighVolumeStockBrandsRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.GetHighVolumeStockBrandsResponse
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "正常系: 全件取得成功",
			fields: fields{
				getHighVolumeStockBrandsUseCase: func(ctrl *gomock.Controller) *mock_usecase.MockGetHighVolumeStockBrandsUseCase {
					mock := mock_usecase.NewMockGetHighVolumeStockBrandsUseCase(ctrl)
					mock.EXPECT().
						ExecuteWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(int(0))).
						Return(&models.PaginatedHighVolumeStockBrands{
							Brands: []*models.HighVolumeStockBrand{
								models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
								models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
							},
							NextCursor: nil,
							Limit:      0,
						}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: &pb.GetHighVolumeStockBrandsRequest{},
			},
			want: &pb.GetHighVolumeStockBrandsResponse{
				Brands: []*pb.HighVolumeStockBrand{
					{
						StockBrandId:  "uuid-1",
						TickerSymbol:  "1001",
						CompanyName:   "Test Brand 1",
						VolumeAverage: 1000000,
						CreatedAt:     now.Format("2006-01-02T15:04:05Z07:00"),
					},
					{
						StockBrandId:  "uuid-2",
						TickerSymbol:  "1002",
						CompanyName:   "Test Brand 2",
						VolumeAverage: 2000000,
						CreatedAt:     now.Format("2006-01-02T15:04:05Z07:00"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: データが存在しない場合は空配列を返す",
			fields: fields{
				getHighVolumeStockBrandsUseCase: func(ctrl *gomock.Controller) *mock_usecase.MockGetHighVolumeStockBrandsUseCase {
					mock := mock_usecase.NewMockGetHighVolumeStockBrandsUseCase(ctrl)
					mock.EXPECT().
						ExecuteWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(int(0))).
						Return(&models.PaginatedHighVolumeStockBrands{Brands: []*models.HighVolumeStockBrand{}, NextCursor: nil, Limit: 0}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: &pb.GetHighVolumeStockBrandsRequest{},
			},
			want: &pb.GetHighVolumeStockBrandsResponse{
				Brands: []*pb.HighVolumeStockBrand{}, Pagination: nil,
			},
			wantErr: false,
		},
		{
			name: "正常系: limit指定でページネーション情報が返される（次ページあり）",
			fields: fields{
				getHighVolumeStockBrandsUseCase: func(ctrl *gomock.Controller) *mock_usecase.MockGetHighVolumeStockBrandsUseCase {
					mock := mock_usecase.NewMockGetHighVolumeStockBrandsUseCase(ctrl)
					nextCursor := "1002"
					mock.EXPECT().
						ExecuteWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(int(2))).
						Return(&models.PaginatedHighVolumeStockBrands{
							Brands: []*models.HighVolumeStockBrand{
								models.NewHighVolumeStockBrand("uuid-1", "1001", "Test Brand 1", 1000000, now),
								models.NewHighVolumeStockBrand("uuid-2", "1002", "Test Brand 2", 2000000, now),
							},
							NextCursor: &nextCursor,
							Limit:      2,
						}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: &pb.GetHighVolumeStockBrandsRequest{
					Limit: 2,
				},
			},
			want: &pb.GetHighVolumeStockBrandsResponse{
				Brands: []*pb.HighVolumeStockBrand{
					{
						StockBrandId:  "uuid-1",
						TickerSymbol:  "1001",
						CompanyName:   "Test Brand 1",
						VolumeAverage: 1000000,
						CreatedAt:     now.Format("2006-01-02T15:04:05Z07:00"),
					},
					{
						StockBrandId:  "uuid-2",
						TickerSymbol:  "1002",
						CompanyName:   "Test Brand 2",
						VolumeAverage: 2000000,
						CreatedAt:     now.Format("2006-01-02T15:04:05Z07:00"),
					},
				},
				Pagination: &pb.PaginationInfo{
					NextCursor: "1002",
					Limit:      2,
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: symbol_from指定でカーソルベースページネーションが動作",
			fields: fields{
				getHighVolumeStockBrandsUseCase: func(ctrl *gomock.Controller) *mock_usecase.MockGetHighVolumeStockBrandsUseCase {
					mock := mock_usecase.NewMockGetHighVolumeStockBrandsUseCase(ctrl)
					nextCursor := "1004"
					mock.EXPECT().
						ExecuteWithPagination(gomock.Any(), gomock.Eq("1002"), gomock.Eq(int(2))).
						Return(&models.PaginatedHighVolumeStockBrands{
							Brands: []*models.HighVolumeStockBrand{
								models.NewHighVolumeStockBrand("uuid-3", "1003", "Test Brand 3", 3000000, now),
								models.NewHighVolumeStockBrand("uuid-4", "1004", "Test Brand 4", 4000000, now),
							},
							NextCursor: &nextCursor,
							Limit:      2,
						}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: &pb.GetHighVolumeStockBrandsRequest{
					SymbolFrom: "1002",
					Limit:      2,
				},
			},
			want: &pb.GetHighVolumeStockBrandsResponse{
				Brands: []*pb.HighVolumeStockBrand{
					{
						StockBrandId:  "uuid-3",
						TickerSymbol:  "1003",
						CompanyName:   "Test Brand 3",
						VolumeAverage: 3000000,
						CreatedAt:     now.Format("2006-01-02T15:04:05Z07:00"),
					},
					{
						StockBrandId:  "uuid-4",
						TickerSymbol:  "1004",
						CompanyName:   "Test Brand 4",
						VolumeAverage: 4000000,
						CreatedAt:     now.Format("2006-01-02T15:04:05Z07:00"),
					},
				},
				Pagination: &pb.PaginationInfo{
					NextCursor: "1004",
					Limit:      2,
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: 最後のページでNextCursorが空文字列",
			fields: fields{
				getHighVolumeStockBrandsUseCase: func(ctrl *gomock.Controller) *mock_usecase.MockGetHighVolumeStockBrandsUseCase {
					mock := mock_usecase.NewMockGetHighVolumeStockBrandsUseCase(ctrl)
					mock.EXPECT().
						ExecuteWithPagination(gomock.Any(), gomock.Eq("1002"), gomock.Eq(int(2))).
						Return(&models.PaginatedHighVolumeStockBrands{
							Brands: []*models.HighVolumeStockBrand{
								models.NewHighVolumeStockBrand("uuid-3", "1003", "Test Brand 3", 3000000, now),
							},
							NextCursor: nil, // 最後のページなのでnil
							Limit:      2,
						}, nil)
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: &pb.GetHighVolumeStockBrandsRequest{
					SymbolFrom: "1002",
					Limit:      2,
				},
			},
			want: &pb.GetHighVolumeStockBrandsResponse{
				Brands: []*pb.HighVolumeStockBrand{
					{
						StockBrandId:  "uuid-3",
						TickerSymbol:  "1003",
						CompanyName:   "Test Brand 3",
						VolumeAverage: 3000000,
						CreatedAt:     now.Format("2006-01-02T15:04:05Z07:00"),
					},
				},
				Pagination: &pb.PaginationInfo{
					NextCursor: "", // 空文字列
					Limit:      2,
				},
			},
			wantErr: false,
		},
		{
			name: "異常系: ユースケースエラー",
			fields: fields{
				getHighVolumeStockBrandsUseCase: func(ctrl *gomock.Controller) *mock_usecase.MockGetHighVolumeStockBrandsUseCase {
					mock := mock_usecase.NewMockGetHighVolumeStockBrandsUseCase(ctrl)
					mock.EXPECT().
						ExecuteWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(int(0))).
						Return(nil, errors.New("usecase error"))
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: &pb.GetHighVolumeStockBrandsRequest{},
			},
			want:    nil,
			wantErr: true,
			errCode: codes.Internal,
		},
		{
			name: "異常系: limitが負の値の場合はエラー",
			fields: fields{
				getHighVolumeStockBrandsUseCase: func(ctrl *gomock.Controller) *mock_usecase.MockGetHighVolumeStockBrandsUseCase {
					mock := mock_usecase.NewMockGetHighVolumeStockBrandsUseCase(ctrl)
					mock.EXPECT().
						ExecuteWithPagination(gomock.Any(), gomock.Eq(""), gomock.Eq(int(-1))).
						Return(nil, errors.New("limit must be non-negative"))
					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				req: &pb.GetHighVolumeStockBrandsRequest{
					Limit: -1,
				},
			},
			want:    nil,
			wantErr: true,
			errCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var getHighVolumeStockBrandsUseCase *mock_usecase.MockGetHighVolumeStockBrandsUseCase
			if tt.fields.getHighVolumeStockBrandsUseCase != nil {
				getHighVolumeStockBrandsUseCase = tt.fields.getHighVolumeStockBrandsUseCase(ctrl)
			}

			s := &StockServiceServer{
				getHighVolumeStockBrandsUseCase: getHighVolumeStockBrandsUseCase,
			}

			got, err := s.GetHighVolumeStockBrands(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetHighVolumeStockBrands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("GetHighVolumeStockBrands() error is not gRPC status error")
					return
				}
				if st.Code() != tt.errCode {
					t.Errorf("GetHighVolumeStockBrands() error code = %v, want %v", st.Code(), tt.errCode)
				}
				return
			}

			if len(got.Brands) != len(tt.want.Brands) {
				t.Errorf("GetHighVolumeStockBrands() got %d brands, want %d", len(got.Brands), len(tt.want.Brands))
				return
			}

			for i, brand := range got.Brands {
				if brand.StockBrandId != tt.want.Brands[i].StockBrandId ||
					brand.TickerSymbol != tt.want.Brands[i].TickerSymbol ||
					brand.CompanyName != tt.want.Brands[i].CompanyName ||
					brand.VolumeAverage != tt.want.Brands[i].VolumeAverage ||
					brand.CreatedAt != tt.want.Brands[i].CreatedAt {
					t.Errorf("GetHighVolumeStockBrands() brand[%d] = %v, want %v", i, brand, tt.want.Brands[i])
				}
			}

			// Paginationフィールドの検証
			if tt.want.Pagination == nil {
				if got.Pagination != nil {
					t.Errorf("GetHighVolumeStockBrands() pagination = %v, want nil", got.Pagination)
				}
			} else {
				if got.Pagination == nil {
					t.Errorf("GetHighVolumeStockBrands() pagination = nil, want %v", tt.want.Pagination)
					return
				}
				if got.Pagination.NextCursor != tt.want.Pagination.NextCursor {
					t.Errorf("GetHighVolumeStockBrands() pagination.NextCursor = %v, want %v", got.Pagination.NextCursor, tt.want.Pagination.NextCursor)
				}
				if got.Pagination.Limit != tt.want.Pagination.Limit {
					t.Errorf("GetHighVolumeStockBrands() pagination.Limit = %v, want %v", got.Pagination.Limit, tt.want.Pagination.Limit)
				}
			}
		})
	}
}
