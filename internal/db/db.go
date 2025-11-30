package db

// DB is a generic database port that allows swapping
// GORM, sqlc, pgx, bun, ent or even in-memory DB.
type DB interface {
	Conn() any
}
