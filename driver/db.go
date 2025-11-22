//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/Code0716/stock-price-repository/config"
)

// NewDBConn initializes DB connection.
func NewDBConn() (conn *sql.DB, close func(), err error) {
	dsn, err := BuildMySQLConnectionString()
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, err
	}

	close = func() {
		if err := sqlDB.Close(); err != nil {
			panic(err)
		}
	}

	return sqlDB, close, nil
}

// BuildMySQLConnectionString builds mysql connection string.
func BuildMySQLConnectionString() (string, error) {
	mysqlCfg := mysql.NewConfig()
	dbConfig := config.Database()

	mysqlCfg.DBName = dbConfig.DBName
	mysqlCfg.Net = "tcp"
	mysqlCfg.Addr = fmt.Sprintf("%s:%s", dbConfig.Host, dbConfig.Port)
	mysqlCfg.User = dbConfig.User
	mysqlCfg.Passwd = dbConfig.Passwd
	mysqlCfg.ParseTime = true
	loc, err := time.LoadLocation(dbConfig.Timezone)
	if err != nil {
		return "", err
	}
	mysqlCfg.Loc = loc
	// mysqlCfg.Collation = env.DBCharset
	ret := mysqlCfg.FormatDSN()
	return ret, nil
}
