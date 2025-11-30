package gormdb

import (
	"github.com/oggyb/insider-assessment/internal/db"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type GormDB struct {
	conn *gorm.DB
}

func New(dsn string) (*GormDB, error) {
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, err
	}
	return &GormDB{conn: conn}, nil
}

func (g *GormDB) Conn() any {
	return g.conn
}

// verify it satisfies db.DB
var _ db.DB = (*GormDB)(nil)
