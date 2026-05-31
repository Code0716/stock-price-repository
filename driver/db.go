//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../mock/$GOPACKAGE/$GOFILE
package driver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/Code0716/stock-price-repository/config"
)

const (
	// docker compose の depends_on(service_healthy) だけでは MySQL 初回起動の
	// 一時サーバや起動直後のクラッシュリカバリ中の谷間を吸収しきれず、
	// gorm.Open() の初回 Ping が失敗してプロセスが即死する。DB 側にだけ
	// リトライが無かったため、アプリ側で接続確立まで待つ。
	dbConnMaxPingRetries = 30              // 最大試行回数
	dbConnPingInterval   = 2 * time.Second // 試行間隔（30回 = 最大約60秒待ち。MySQL のリカバリ/init を吸収できる長さ）
	dbConnPingTimeout    = 3 * time.Second // 1回の Ping のタイムアウト（ハング防止）
)

// NewDBConn initializes DB connection.
func NewDBConn() (conn *sql.DB, cleanup func(), err error) {
	dsn, err := BuildMySQLConnectionString()
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, err
	}

	cleanup = func() {
		if err := sqlDB.Close(); err != nil {
			panic(err)
		}
	}

	// sql.Open は遅延接続のため、実接続が確立できるまで Ping をリトライする。
	// これにより後続の NewGorm()(gorm.Open) 時点で接続が生きていることを保証する。
	if err := pingWithRetry(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, nil, errors.Wrap(err, "NewDBConn.pingWithRetry retry exhausted")
	}

	return sqlDB, cleanup, nil
}

// pingWithRetry pings the DB until it succeeds or retries are exhausted.
func pingWithRetry(sqlDB *sql.DB) error {
	var err error
	for range dbConnMaxPingRetries {
		ctx, cancel := context.WithTimeout(context.Background(), dbConnPingTimeout)
		err = sqlDB.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		time.Sleep(dbConnPingInterval)
	}
	return err
}

// BuildMySQLConnectionString builds mysql connection string.
func BuildMySQLConnectionString() (string, error) {
	mysqlCfg := mysql.NewConfig()
	dbConfig := config.GetDatabase()

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
