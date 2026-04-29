package driver

import (
	"bytes"
	"context"
	"log"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
)

type BoxAPIClient struct{}

func NewBoxAPIClient() gateway.BoxClient {
	return &BoxAPIClient{}
}

func (c *BoxAPIClient) UploadFile(ctx context.Context, localFilePath string) error {
	cfg := config.GetBOX()

	if cfg.RcloneFolderPath == "" {
		return errors.New("box upload skipped: BOX_RCLONE_FOLDER_PATH not set")
	}

	if _, err := exec.LookPath("rclone"); err != nil {
		return errors.New("box upload skipped: rclone not installed")
	}

	dest := cfg.RcloneRemoteName + ":" + cfg.RcloneFolderPath
	cmd := execCommandContext(ctx, "rclone", "copy", localFilePath, dest)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "rclone copy failed: %s", stderr.String())
	}

	log.Printf("box upload succeeded: %s -> %s", localFilePath, dest)
	return nil
}
