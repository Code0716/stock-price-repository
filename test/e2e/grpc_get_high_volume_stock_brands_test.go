package e2e

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/Code0716/stock-price-repository/entrypoint/grpc/server"
	"github.com/Code0716/stock-price-repository/infrastructure/database"
	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	"github.com/Code0716/stock-price-repository/pb"
	"github.com/Code0716/stock-price-repository/test/helper"
	"github.com/Code0716/stock-price-repository/usecase"
)

const bufSize = 1024 * 1024

func TestE2E_GetHighVolumeStockBrands(t *testing.T) {
	// 1. Setup DB
	db, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	// 2. Setup Server Components
	repo := database.NewHighVolumeStockBrandRepositoryImpl(db)
	uc := usecase.NewGetHighVolumeStockBrandsUseCase(repo)
	srv := server.NewStockServiceServer(uc)

	// 3. Setup gRPC Server with bufconn
	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	pb.RegisterStockServiceServer(grpcServer, srv)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	defer grpcServer.Stop()

	// 4. Setup gRPC Client
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("passthrough://bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewStockServiceClient(conn)

	// 5. Test Cases
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		req     *pb.GetHighVolumeStockBrandsRequest
		check   func(t *testing.T, resp *pb.GetHighVolumeStockBrandsResponse, err error)
	}{
		{
			name: "正常系: データなし",
			setup: func(t *testing.T) {
				// 何もしない（テーブルは空）
			},
			req: &pb.GetHighVolumeStockBrandsRequest{
				Limit: 0,
			},
			check: func(t *testing.T, resp *pb.GetHighVolumeStockBrandsResponse, err error) {
				require.NoError(t, err)
				assert.Empty(t, resp.Brands)
				assert.Nil(t, resp.Pagination)
			},
		},
		{
			name: "正常系: 全件取得",
			setup: func(t *testing.T) {
				// StockBrandデータ作成
				sb1 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "1001",
					Name:         "テスト銘柄1",
					MarketCode:   "111",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				sb2 := &genModel.StockBrand{
					ID:           uuid.New().String(),
					TickerSymbol: "1002",
					Name:         "テスト銘柄2",
					MarketCode:   "111",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				err := db.Create(sb1).Error
				require.NoError(t, err)
				err = db.Create(sb2).Error
				require.NoError(t, err)

				// HighVolumeStockBrandデータ作成
				hvb1 := &genModel.HighVolumeStockBrand{
					StockBrandID:  sb1.ID,
					TickerSymbol:  sb1.TickerSymbol,
					VolumeAverage: 10000,
					CreatedAt:     time.Now(),
				}
				hvb2 := &genModel.HighVolumeStockBrand{
					StockBrandID:  sb2.ID,
					TickerSymbol:  sb2.TickerSymbol,
					VolumeAverage: 20000,
					CreatedAt:     time.Now(),
				}
				err = db.Create(hvb1).Error
				require.NoError(t, err)
				err = db.Create(hvb2).Error
				require.NoError(t, err)
			},
			req: &pb.GetHighVolumeStockBrandsRequest{
				Limit: 0,
			},
			check: func(t *testing.T, resp *pb.GetHighVolumeStockBrandsResponse, err error) {
				require.NoError(t, err)
				require.Len(t, resp.Brands, 2)
				// 銘柄コード順にソートされていることを確認
				assert.Equal(t, "1001", resp.Brands[0].TickerSymbol)
				assert.Equal(t, "テスト銘柄1", resp.Brands[0].CompanyName)
				assert.Equal(t, "1002", resp.Brands[1].TickerSymbol)
				assert.Equal(t, "テスト銘柄2", resp.Brands[1].CompanyName)
				assert.Nil(t, resp.Pagination)
			},
		},
		{
			name: "正常系: ページネーション (Limit指定)",
			setup: func(t *testing.T) {
				// データ作成 (3件)
				for i, symbol := range []string{"2001", "2002", "2003"} {
					sb := &genModel.StockBrand{
						ID:           uuid.New().String(),
						TickerSymbol: symbol,
						Name:         "テスト銘柄" + symbol,
						MarketCode:   "111",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					require.NoError(t, db.Create(sb).Error)

					hvb := &genModel.HighVolumeStockBrand{
						StockBrandID:  sb.ID,
						TickerSymbol:  sb.TickerSymbol,
						VolumeAverage: uint64(1000 * (i + 1)),
						CreatedAt:     time.Now(),
					}
					require.NoError(t, db.Create(hvb).Error)
				}
			},
			req: &pb.GetHighVolumeStockBrandsRequest{
				Limit: 2,
			},
			check: func(t *testing.T, resp *pb.GetHighVolumeStockBrandsResponse, err error) {
				require.NoError(t, err)
				require.Len(t, resp.Brands, 2)
				assert.Equal(t, "2001", resp.Brands[0].TickerSymbol)
				assert.Equal(t, "2002", resp.Brands[1].TickerSymbol)
				
				require.NotNil(t, resp.Pagination)
				assert.Equal(t, int32(2), resp.Pagination.Limit)
				// 次のカーソルは2件目のTickerSymbol
				assert.Equal(t, "2002", resp.Pagination.NextCursor)
			},
		},
		{
			name: "正常系: ページネーション (NextCursor指定)",
			setup: func(t *testing.T) {
				// データ作成 (3件)
				for i, symbol := range []string{"3001", "3002", "3003"} {
					sb := &genModel.StockBrand{
						ID:           uuid.New().String(),
						TickerSymbol: symbol,
						Name:         "テスト銘柄" + symbol,
						MarketCode:   "111",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					require.NoError(t, db.Create(sb).Error)

					hvb := &genModel.HighVolumeStockBrand{
						StockBrandID:  sb.ID,
						TickerSymbol:  sb.TickerSymbol,
						VolumeAverage: uint64(1000 * (i + 1)),
						CreatedAt:     time.Now(),
					}
					require.NoError(t, db.Create(hvb).Error)
				}
			},
			req: &pb.GetHighVolumeStockBrandsRequest{
				SymbolFrom: "3001",
				Limit:      2,
			},
			check: func(t *testing.T, resp *pb.GetHighVolumeStockBrandsResponse, err error) {
				require.NoError(t, err)
				require.Len(t, resp.Brands, 2)
				// 3001より大きいものから取得開始 -> 3002, 3003
				assert.Equal(t, "3002", resp.Brands[0].TickerSymbol)
				assert.Equal(t, "3003", resp.Brands[1].TickerSymbol)

				require.NotNil(t, resp.Pagination)
				// 最後まで取得したのでNextCursorは空文字かnil (実装依存だがprotoではstringなので空文字)
				// usecaseの実装を見ると、limit+1件取れない場合はNextCursorを設定しない
				assert.Empty(t, resp.Pagination.NextCursor)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up tables
			helper.TruncateAllTables(t, db)
			
			if tt.setup != nil {
				tt.setup(t)
			}

			resp, err := client.GetHighVolumeStockBrands(context.Background(), tt.req)
			if tt.check != nil {
				tt.check(t, resp, err)
			}
		})
	}
}
