//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"log"

	stDriver "github.com/Code0716/stock-price-repository/driver"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type contextKey int

const dbTxContextKey contextKey = iota

// Transaction transaction
// gorm.io/genは中間テーブルのqueryがなぜか登録されないなど出来ないことも多いそうなので、完全以降はまだしない。
// https://zenn.dev/mizuko_dev/articles/dbeb7c93883b57#%E4%B8%AD%E9%96%93%E3%83%86%E3%83%BC%E3%83%96%E3%83%AB%E3%81%AEquery%E3%81%8C%E3%81%AA%E3%81%9C%E3%81%8B%E7%99%BB%E9%8C%B2%E3%81%95%E3%82%8C%E3%81%AA%E3%81%84
type Transaction interface {
	DoInTx(ctx context.Context, fn func(context.Context) error) error
}

type transaction struct {
	db *gorm.DB
}

// NewTransaction init Transaction
func NewTransaction() Transaction {
	db, _, err := stDriver.NewDBConn()
	if err != nil {
		panic(any(err))
	}
	gm, err := stDriver.NewGorm(db)
	if err != nil {
		panic(any(err))
	}
	return &transaction{
		gm,
	}
}

// DoInTx トランザクション
func (t *transaction) DoInTx(ctx context.Context, fn func(context.Context) error) error {
	tx := t.db.Begin()
	var err error

	ctx = setTx(ctx, tx)

	log.Print("transaction start. \t\n")
	err = fn(ctx)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "")
	}

	log.Print("transaction commit start. \t\n")
	err = tx.Commit().Error
	if err != nil {
		if err == gorm.ErrInvalidTransaction {
			err = errors.Wrap(err, "")
		}

		if rollbackErr := tx.Rollback().Error; rollbackErr != nil {
			if rollbackErr != gorm.ErrInvalidTransaction {
				err = errors.Wrap(rollbackErr, err.Error())
			}
		}
		log.Printf("transaction error:\t%v\n", err)

		return errors.Wrap(err, "")
	}
	log.Print("transaction commit done. \t\n")

	return nil
}

// setTx set context.Context to Tx
func setTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, dbTxContextKey, tx)
}

// GetTxQuery txを取得する。
func GetTxQuery(ctx context.Context) (*genQuery.Query, bool) {
	tx, ok := ctx.Value(dbTxContextKey).(*gorm.DB)
	if tx == nil {
		return nil, false
	}
	return genQuery.Use(tx), ok
}

// GetTxDB txを取得する。
func GetTxDB(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(dbTxContextKey).(*gorm.DB)
	return tx, ok
}

//  gormでテーブルをsql export するなら。
// package main

// import (
//     "fmt"
//     "os"
//     "strings"

//     "gorm.io/driver/mysql"
//     "gorm.io/gorm"
// )

// func main() {
//     dsn := "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
//     db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
//     if err != nil {
//         panic(err)
//     }

//     sqlFile, err := os.Create("export.sql")
//     if err != nil {
//         panic(err)
//     }
//     defer sqlFile.Close()

//     var rows []string
//     db.Table("your_table_name").Find(&rows)
//     for _, row := range rows {
//         sqlFile.WriteString(fmt.Sprintf("INSERT INTO your_table_name VALUES (%s);\n", row))
//     }

//     fmt.Println("Export completed!")
// }
