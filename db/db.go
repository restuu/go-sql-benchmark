// Package db embeds the benchmark schema and seed so the runner can reseed between cells.
package db

import _ "embed"

// Schema is db/schema.sql.
//
//go:embed schema.sql
var Schema string

// Seed is db/seed.sql: it truncates every table and reloads the benchmark data.
//
//go:embed seed.sql
var Seed string
