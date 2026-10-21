package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Order is the orders table.
type Order struct {
	ent.Schema
}

func (Order) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "orders"}}
}

func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.Int("customer_id"),
		field.String("status"),
		field.Int("total"),
		field.Time("created_at"),
	}
}

func (Order) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).Ref("orders").Field("customer_id").Unique().Required(),
	}
}
