//go:build ignore

// Command generate generates the gorm/gen model structs and typed query API from the live schema
// (BENCH_DSN). Run it with `make gen-gormgen` (it is excluded from the build); the output is committed.
package main

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func main() {
	dsn := os.Getenv("BENCH_DSN")
	if dsn == "" {
		log.Fatal("generate: BENCH_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		log.Fatalf("generate: open: %v", err)
	}

	g := gen.NewGenerator(gen.Config{
		OutPath:      "internal/adapters/gormgen/query",
		ModelPkgPath: "internal/adapters/gormgen/entity",
		Mode:         gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	g.UseDB(db)
	g.ApplyBasic(
		g.GenerateModel("models"),
		g.GenerateModel("customers"),
		g.GenerateModel("orders"),
	)
	g.Execute()
}
