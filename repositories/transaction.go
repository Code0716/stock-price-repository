package repositories

import "context"

//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE

// Transaction transaction interface
type Transaction interface {
	DoInTx(ctx context.Context, fn func(context.Context) error) error
}
