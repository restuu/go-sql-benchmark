package correctness

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/runner"
)

// missingID is an id no seeded or inserted row has.
const missingID = 99_999_999

// checkWrites runs INSERT, UPDATE and DELETE through the subject and reads the results back
// through the reference. The database must be freshly seeded.
func checkWrites(ctx context.Context, store runner.Store, ref bench.Verifier, s bench.Adapter) []string {
	name := s.Name()
	var msgs []string

	n, err := s.Insert(ctx, distinct)
	msgs = append(msgs, expectRows(name, "insert", n, err, 1)...)
	inserted := distinct
	inserted.ID = insertedID()
	msgs = append(msgs, expectStored(ctx, ref, name, "insert", inserted)...)

	n, err = s.Update(ctx, updatedID, distinct)
	msgs = append(msgs, expectRows(name, "update", n, err, 1)...)
	updated := distinct
	updated.ID = updatedID
	msgs = append(msgs, expectStored(ctx, ref, name, "update", updated)...)

	msgs = append(msgs, checkMissingID(ctx, s)...)
	return append(msgs, checkDelete(ctx, store, ref, s)...)
}

// checkMissingID verifies the write semantics every adapter must share: UPDATE of a missing id
// affects nothing (no upsert) and SELECT of a missing id fails.
func checkMissingID(ctx context.Context, s bench.Adapter) []string {
	name := s.Name()
	n, err := s.Update(ctx, missingID, distinct)
	msgs := expectRows(name, "update of a missing id", n, err, 0)
	if _, err := s.Select(ctx, missingID); err == nil {
		msgs = append(msgs, name+": select of a missing id succeeded")
	}
	return msgs
}

func checkDelete(ctx context.Context, store runner.Store, ref bench.Verifier, s bench.Adapter) []string {
	name := s.Name()
	ids, err := store.InsertDeletable(ctx, 1)
	if err != nil {
		return []string{fmt.Sprintf("%s: delete: prepare row: %v", name, err)}
	}
	n, err := s.Delete(ctx, ids[0])
	msgs := expectRows(name, "delete", n, err, 1)
	switch _, err := ref.ReadModel(ctx, ids[0]); {
	case err == nil:
		msgs = append(msgs, fmt.Sprintf("%s: delete: row %d still present", name, ids[0]))
	case !errors.Is(err, sql.ErrNoRows):
		msgs = append(msgs, fmt.Sprintf("%s: delete: read back id %d: %v", name, ids[0], err))
	}
	if _, ok := s.(bench.DeleteCountUnreported); ok {
		return msgs // it cannot tell a missing id from a deleted one
	}
	n, err = s.Delete(ctx, ids[0])
	return append(msgs, expectRows(name, "second delete", n, err, 0)...)
}

func expectRows(lib, op string, got int, err error, want int) []string {
	if err != nil {
		return []string{fmt.Sprintf("%s: %s: %v", lib, op, err)}
	}
	if got != want {
		return []string{fmt.Sprintf("%s: %s: rows = %d, want %d", lib, op, got, want)}
	}
	return nil
}

// expectStored reads the row back through the reference and compares it with want.
func expectStored(ctx context.Context, ref bench.Verifier, lib, op string, want bench.Model) []string {
	got, err := ref.ReadModel(ctx, want.ID)
	if err != nil {
		return []string{fmt.Sprintf("%s: %s: read back id %d: %v", lib, op, want.ID, err)}
	}
	if d := diffModel(got, want); d != "" {
		return []string{fmt.Sprintf("%s: %s: stored row differs: %s", lib, op, d)}
	}
	return nil
}
