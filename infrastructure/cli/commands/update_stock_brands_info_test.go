package commands

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestUpdateStockBrandsV1Command_Action(t *testing.T) {
	type fields struct {
		stockBrandInteractor func(ctrl *gomock.Controller) usecase.StockBrandInteractor
	}
	type args struct {
		ctx *cli.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "正常系",
			fields: fields{
				stockBrandInteractor: func(ctrl *gomock.Controller) usecase.StockBrandInteractor {
					mock := mock_usecase.NewMockStockBrandInteractor(ctrl)
					mock.EXPECT().UpdateStockBrands(gomock.Any(), gomock.Any()).Return(nil)
					return mock
				},
			},
			args: args{
				ctx: cli.NewContext(cli.NewApp(), flag.NewFlagSet("test", 0), nil),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := &UpdateStockBrandsV1Command{
				stockBrandInteractor: tt.fields.stockBrandInteractor(ctrl),
			}
			if err := c.Action(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("UpdateStockBrandsV1Command.Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
