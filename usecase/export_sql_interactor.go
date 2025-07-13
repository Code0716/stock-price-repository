//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package usecase

import (
	"context"
	"time"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
)

type exportSQLInteractorImpl struct {
	mySQLDumpClient gateway.MySQLDumpClient
}

func NewExportSQLInteractor(
	mySQLDumpClient gateway.MySQLDumpClient,

) ExportSQLInteractor {
	return &exportSQLInteractorImpl{
		mySQLDumpClient: mySQLDumpClient,
	}
}

type ExportSQLInteractor interface {
	ExportSQLFiles(ctx context.Context, t time.Time) error
}
