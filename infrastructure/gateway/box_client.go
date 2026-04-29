//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package gateway

import "context"

type BoxClient interface {
	UploadFile(ctx context.Context, localFilePath string) error
}
