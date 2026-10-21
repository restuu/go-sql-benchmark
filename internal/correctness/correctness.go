// Package correctness checks that every adapter reads and writes the same values, not only the same
// row counts. Adapters implement bench.Verifier; the reference adapter (database/sql) is first
// compared with golden values derived from db/seed.sql, then every other adapter is compared
// with the reference.
package correctness

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/restuu/go-sql-benchmark/internal/bench"
	"github.com/restuu/go-sql-benchmark/internal/runner"
	"github.com/restuu/go-sql-benchmark/internal/seedcheck"
)

var (
	readModelIDs    = []int{1, 5000, 10000}
	readCustomerIDs = []int{1, 7, 1000}
	readCities      = []string{"city 0", "city 3", "city 49"}

	// distinct differs from the seed in every column, and Right is false: a library that skips
	// zero values or maps columns wrongly cannot pass with bench.NewModel(), whose values equal the seed.
	distinct = bench.Model{Name: "n", Title: "t", Fax: "f", Web: "w", Age: 7, Right: false, Counter: 42}
)

const updatedID = 5

// insertedID is the id the first INSERT after a reseed receives (Reseed restarts identities).
func insertedID() int { return int(seedcheck.ExpectedRows["models"]) + 1 }

// Check compares every subject with ref. ref and the subjects must already be initialised. It
// reseeds before and after (also on failure) and returns one error listing every mismatch.
func Check(ctx context.Context, store runner.Store, ref bench.Adapter, subjects []bench.Adapter) (err error) {
	refV, ok := ref.(bench.Verifier)
	if !ok {
		return fmt.Errorf("correctness: reference %s does not implement bench.Verifier", ref.Name())
	}
	if err := store.Reseed(ctx); err != nil {
		return fmt.Errorf("correctness: %w", err)
	}
	defer func() {
		if rerr := store.Reseed(context.WithoutCancel(ctx)); rerr != nil {
			err = errors.Join(err, fmt.Errorf("correctness: final reseed: %w", rerr))
		}
	}()

	msgs := checkReference(ctx, refV)
	for _, s := range subjects {
		msgs = append(msgs, checkSubject(ctx, store, refV, s)...)
	}
	if len(msgs) > 0 {
		return fmt.Errorf("correctness: %d mismatches:\n  %s", len(msgs), strings.Join(msgs, "\n  "))
	}
	return nil
}

func checkSubject(ctx context.Context, store runner.Store, ref bench.Verifier, s bench.Adapter) []string {
	v, ok := s.(bench.Verifier)
	if !ok {
		return []string{s.Name() + ": does not implement bench.Verifier"}
	}
	// Reseed before the reads too, so a previous subject's writes can never affect them.
	if err := store.Reseed(ctx); err != nil {
		return []string{fmt.Sprintf("%s: reseed before reads: %v", s.Name(), err)}
	}
	msgs := checkReads(ctx, ref, s, v)
	if err := store.Reseed(ctx); err != nil {
		return append(msgs, fmt.Sprintf("%s: reseed before writes: %v", s.Name(), err))
	}
	return append(msgs, checkWrites(ctx, store, ref, s)...)
}

func checkReads(ctx context.Context, ref bench.Verifier, s bench.Adapter, v bench.Verifier) []string {
	var msgs []string
	for _, id := range readModelIDs {
		want, werr := ref.ReadModel(ctx, id)
		got, gerr := v.ReadModel(ctx, id)
		msgs = append(msgs, compareRead(s.Name(), fmt.Sprintf("read model %d", id), werr, gerr, func() string { return diffModel(got, want) })...)
	}
	for _, c := range readCustomerIDs {
		msgs = append(msgs, checkJoin(ctx, ref, s, v, c)...)
	}
	for _, city := range readCities {
		msgs = append(msgs, checkUnion(ctx, ref, s, v, city)...)
	}
	return msgs
}

// compareRead turns the outcome of one reference/subject read pair into mismatch messages.
func compareRead(lib, what string, refErr, gotErr error, diff func() string) []string {
	switch {
	case refErr != nil:
		return []string{fmt.Sprintf("reference: %s: %v", what, refErr)}
	case gotErr != nil:
		return []string{fmt.Sprintf("%s: %s: %v", lib, what, gotErr)}
	}
	if d := diff(); d != "" {
		return []string{fmt.Sprintf("%s: %s: %s", lib, what, d)}
	}
	return nil
}

func checkJoin(ctx context.Context, ref bench.Verifier, s bench.Adapter, v bench.Verifier, customer int) []string {
	what := fmt.Sprintf("read join %d", customer)
	want, werr := ref.ReadJoin(ctx, customer)
	got, gerr := v.ReadJoin(ctx, customer)
	n, aerr := s.Join(ctx, customer)
	if msg := consistency(s.Name(), what, gerr, aerr, len(got), n); msg != "" {
		return []string{msg}
	}
	if errors.Is(gerr, bench.ErrUnsupported) {
		return nil
	}
	return compareRead(s.Name(), what, werr, gerr, func() string { return diffJoin(got, want) })
}

func checkUnion(ctx context.Context, ref bench.Verifier, s bench.Adapter, v bench.Verifier, city string) []string {
	what := fmt.Sprintf("read union %q", city)
	want, werr := ref.ReadUnion(ctx, city)
	got, gerr := v.ReadUnion(ctx, city)
	n, aerr := s.Union(ctx, city)
	if msg := consistency(s.Name(), what, gerr, aerr, len(got), n); msg != "" {
		return []string{msg}
	}
	if errors.Is(gerr, bench.ErrUnsupported) {
		return nil
	}
	return compareRead(s.Name(), what, werr, gerr, func() string { return diffUnion(got, want) })
}

// consistency checks that the Verifier and the Adapter agree on whether a scenario is supported,
// and that the Adapter's row count equals the number of rows the Verifier returned.
func consistency(lib, what string, verErr, adErr error, verRows, adRows int) string {
	verUnsupported := errors.Is(verErr, bench.ErrUnsupported)
	adUnsupported := errors.Is(adErr, bench.ErrUnsupported)
	switch {
	case verUnsupported != adUnsupported:
		return fmt.Sprintf("%s: %s: unsupported on one side only (verifier %v, adapter %v)", lib, what, verErr, adErr)
	case verUnsupported:
		return ""
	case adErr != nil:
		return fmt.Sprintf("%s: %s: adapter call: %v", lib, what, adErr)
	case verErr == nil && adRows != verRows:
		return fmt.Sprintf("%s: %s: adapter counted %d rows, verifier returned %d", lib, what, adRows, verRows)
	}
	return ""
}
