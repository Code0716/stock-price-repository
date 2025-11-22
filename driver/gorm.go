//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"database/sql"
	"log"
	"os"

	gormMySQLDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/Code0716/stock-price-repository/config"
)

// NewGorm initializes db
func NewGorm(conn *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(
		gormMySQLDriver.New(
			gormMySQLDriver.Config{Conn: conn}),
		&gorm.Config{
			Logger: gormLogger.New(
				log.New(os.Stdout, "\r\n", log.LstdFlags),
				gormLogger.Config{
					LogLevel: logLevelToGormLogLevel(config.App().AppLogLevel),
				}),
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return gormDB, nil
}

// logLevelToGormLogLevel level
func logLevelToGormLogLevel(logLevel string) gormLogger.LogLevel {
	switch logLevel {
	case "debug":
		return gormLogger.Info
	case "warn":
		return gormLogger.Warn
	case "error":
		return gormLogger.Error
	case "raspberrypi":
		return gormLogger.Error
	default:
		return gormLogger.Info
	}
}
