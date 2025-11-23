package helper

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB sets up a test database and returns the gorm.DB connection and a cleanup function.
func SetupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	dbUser := os.Getenv("TEST_DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPassword := os.Getenv("TEST_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "root"
	}
	dbHost := os.Getenv("TEST_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("TEST_DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}

	// ランダムなDB名を生成
	randBytes := make([]byte, 4)
	_, err := rand.Read(randBytes)
	require.NoError(t, err)
	dbName := fmt.Sprintf("test_db_%s", hex.EncodeToString(randBytes))

	// まずDBを作成するために接続（DB名なし）
	dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPassword, dbHost, dbPort)
	db, err := gorm.Open(mysql.Open(dsnWithoutDB), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// DB作成
	err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)).Error
	require.NoError(t, err)

	// 作成したDBに接続
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUser, dbPassword, dbHost, dbPort, dbName)
	testDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// SQLファイルからテーブル作成
	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filename)
	// test/helper/db.go から見て ../../sql/_init_sql/00001_create_database.sql
	initSQLPath := filepath.Join(baseDir, "../../sql/_init_sql/00001_create_database.sql")

	// 初期SQL実行
	executeSQLFile(t, testDB, initSQLPath, dbName)

	// マイグレーション実行
	migrationDir := filepath.Join(baseDir, "../../sql/migrations")
	migrationFiles, err := filepath.Glob(filepath.Join(migrationDir, "*.up.sql"))
	require.NoError(t, err)

	for _, migrationFile := range migrationFiles {
		executeSQLFile(t, testDB, migrationFile, dbName)
	}

	cleanup := func() {
		// DB接続を閉じる
		sqlDB, err := testDB.DB()
		if err == nil {
			sqlDB.Close()
		}

		// DB削除
		dropDB, err := gorm.Open(mysql.Open(dsnWithoutDB), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			dropDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		}
	}

	return testDB, cleanup
}

func executeSQLFile(t *testing.T, db *gorm.DB, filePath string, dbName string) {
	sqlBytes, err := os.ReadFile(filePath)
	require.NoError(t, err)

	sqlContent := string(sqlBytes)
	// DB名を置換
	sqlContent = strings.ReplaceAll(sqlContent, "`stock_price_repository`", fmt.Sprintf("`%s`", dbName))

	// ステートメントごとに実行
	stmts := strings.Split(sqlContent, ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		// DROP DATABASE, CREATE DATABASE はスキップ (すでに作成済みのため)
		if strings.HasPrefix(strings.ToUpper(stmt), "DROP DATABASE") || strings.HasPrefix(strings.ToUpper(stmt), "CREATE DATABASE") {
			continue
		}
		err = db.Exec(stmt).Error
		require.NoError(t, err, "failed to execute statement in %s: %s", filePath, stmt)
	}
}
