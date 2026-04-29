package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type BOX struct {
	RcloneRemoteName string `envconfig:"box_rclone_remote_name" default:"box"`
	RcloneFolderPath string `envconfig:"box_rclone_folder_path" default:""`
}

var box BOX

func LoadConfigBOX() {
	prefix := ""
	err := envconfig.Process(prefix, &box)
	if err != nil {
		log.Fatalf("failed to init config: %v", err)
	}
}

func GetBOX() *BOX {
	return &box
}
