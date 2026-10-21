package runner

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" driver

	benchdb "github.com/restuu/go-sql-benchmark/db"
	"github.com/restuu/go-sql-benchmark/internal/bench"
)

// insertDeletableSQL bulk-inserts $1 copies of the benchmark model and returns their ids.
const insertDeletableSQL = `INSERT INTO models (name, title, fax, web, age, "right", counter)
SELECT $2::text, $3::text, $4::text, $5::text, $6::integer, $7::boolean, $8::bigint
FROM generate_series(1, $1::integer) RETURNING id`

// Store prepares database state between cells; the Postgres implementation is pgStore.
type Store interface {
	// Reseed truncates every table and reloads db/seed.sql. No other connection may be open.
	Reseed(ctx context.Context) error
	// InsertDeletable inserts n untimed rows and returns their ids, for the DELETE scenario.
	InsertDeletable(ctx context.Context, n int) ([]int, error)
}

type pgStore struct{ db *sql.DB }

// NewPGStore returns a Store backed by the given Postgres handle.
func NewPGStore(db *sql.DB) Store { return pgStore{db: db} }

func (s pgStore) Reseed(ctx context.Context) error {
	// A statement without arguments runs on pgx's simple protocol, so the multi-statement script is accepted.
	if _, err := s.db.ExecContext(ctx, benchdb.Seed); err != nil {
		return fmt.Errorf("runner: reseed: %w", err)
	}
	return nil
}

func (s pgStore) InsertDeletable(ctx context.Context, n int) ([]int, error) {
	if n < 0 {
		return nil, fmt.Errorf("runner: insert deletable: negative count %d", n)
	}
	ids := make([]int, 0, n)
	if n == 0 {
		return ids, nil
	}
	m := bench.NewModel()
	rows, err := s.db.QueryContext(ctx, insertDeletableSQL, n, m.Name, m.Title, m.Fax, m.Web, m.Age, m.Right, m.Counter)
	if err != nil {
		return nil, fmt.Errorf("runner: insert deletable: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("runner: insert deletable: scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("runner: insert deletable: %w", err)
	}
	return ids, nil
}
