package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Customer is the customers table.
type Customer struct {
	ent.Schema
}

func (Customer) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "customers"}}
}

func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
		field.String("email"),
		field.String("city"),
	}
}

func (Customer) Edges() []ent.Edge {
	return []ent.Edge{edge.To("orders", Order.Type)}
}
