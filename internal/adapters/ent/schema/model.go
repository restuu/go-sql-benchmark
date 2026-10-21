// Package schema is the ent codegen input. The database schema itself comes from db/schema.sql;
// ent never migrates it.
//
// Derived in part from github.com/efectn/go-orm-benchmarks (MIT), see LICENSE.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// Model is the models table.
type Model struct {
	ent.Schema
}

func (Model) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "models"}}
}

func (Model) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("title"),
		field.String("fax"),
		field.String("web"),
		field.Int("age"),
		field.Bool("right"),
		field.Int64("counter"),
	}
}
