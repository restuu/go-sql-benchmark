package bench

import (
	"slices"
	"testing"
)

func TestSortJoinRows(t *testing.T) {
	tests := []struct {
		name string
		in   []JoinRow
		want []int
	}{
		{"empty", nil, nil},
		{"already ordered", []JoinRow{{OrderID: 1}, {OrderID: 2}}, []int{1, 2}},
		{"shuffled", []JoinRow{{OrderID: 3}, {OrderID: 1}, {OrderID: 2}}, []int{1, 2, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			rows := slices.Clone(tt.in)

			// Act
			SortJoinRows(rows)

			// Assert
			var got []int
			for _, r := range rows {
				got = append(got, r.OrderID)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("order = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortUnionRows(t *testing.T) {
	tests := []struct {
		name string
		in   []UnionRow
		want []string
	}{
		{"empty", nil, nil},
		{"shuffled", []UnionRow{{Name: "b"}, {Name: "a"}, {Name: "c"}}, []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			rows := slices.Clone(tt.in)

			// Act
			SortUnionRows(rows)

			// Assert
			var got []string
			for _, r := range rows {
				got = append(got, r.Name)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("order = %v, want %v", got, tt.want)
			}
		})
	}
}
