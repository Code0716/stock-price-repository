package database

import (
	"testing"

	"gorm.io/gorm"

	"github.com/Code0716/stock-price-repository/test/helper"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	return helper.SetupTestDB(t)
}
