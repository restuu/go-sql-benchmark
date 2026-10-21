package correctness

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/restuu/go-sql-benchmark/internal/bench"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		subjects func(w *world) []bench.Adapter
		wantErr  []string // substrings; nil means Check must pass
	}{
		{"correct adapter passes", func(w *world) []bench.Adapter {
			return []bench.Adapter{&fakeLib{w: w, name: "good"}}
		}, nil},
		{"unsupported union reported consistently passes", func(w *world) []bench.Adapter {
			return []bench.Adapter{&fakeLib{w: w, name: "gormlike", noUnion: true}}
		}, nil},
		{"title and fax swapped on write", func(w *world) []bench.Adapter {
			return []bench.Adapter{&fakeLib{w: w, name: "swapper", swapTitleFax: true}}
		}, []string{"swapper", "insert"}},
		{"false right not written by update", func(w *world) []bench.Adapter {
			return []bench.Adapter{&fakeLib{w: w, name: "zeroskip", dropFalseOnWrite: true}}
		}, []string{"zeroskip", "update"}},
		{"wrong value on read", func(w *world) []bench.Adapter {
			return []bench.Adapter{&fakeLib{w: w, name: "misread", wrongRead: true}}
		}, []string{"misread", "read model 5000"}},
		{"delete leaves the row", func(w *world) []bench.Adapter {
			return []bench.Adapter{&fakeLib{w: w, name: "keeper", keepOnDelete: true}}
		}, []string{"keeper", "delete"}},
		{"update of a missing id inserts", func(w *world) []bench.Adapter {
			return []bench.Adapter{&fakeLib{w: w, name: "upserter", upsertOnUpdate: true}}
		}, []string{"upserter", "update of a missing id"}},
		{"adapter without a verifier", func(w *world) []bench.Adapter {
			return []bench.Adapter{plainAdapter{&fakeLib{w: w, name: "plain"}}}
		}, []string{"plain", "bench.Verifier"}},
		{"verifier says unsupported but adapter runs", func(w *world) []bench.Adapter {
			return []bench.Adapter{&fakeLib{w: w, name: "inconsistent", verifierOnlyNoUn: true}}
		}, []string{"inconsistent", "union"}},
		{"several subjects all reported", func(w *world) []bench.Adapter {
			return []bench.Adapter{
				&fakeLib{w: w, name: "swapper", swapTitleFax: true},
				&fakeLib{w: w, name: "misread", wrongRead: true},
			}
		}, []string{"swapper", "misread"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			w := newWorld()
			store := &fakeStore{w: w}
			ref := &fakeLib{w: w, name: "reference"}

			// Act
			err := Check(context.Background(), store, ref, tt.subjects(w))

			// Assert
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Check() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Check() = nil, want error containing %v", tt.wantErr)
			}
			for _, sub := range tt.wantErr {
				if !strings.Contains(err.Error(), sub) {
					t.Errorf("Check() = %q, missing %q", err, sub)
				}
			}
		})
	}
}

func TestCheckReferenceGolden(t *testing.T) {
	tests := []struct {
		name string
		ref  func(w *world) bench.Adapter
	}{
		{"model values differ from the seed", func(w *world) bench.Adapter {
			return &fakeLib{w: w, name: "reference", wrongFirst: true}
		}},
		{"reference without a verifier", func(w *world) bench.Adapter {
			return plainAdapter{&fakeLib{w: w, name: "reference"}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			w := newWorld()
			ref := tt.ref(w)

			// Act
			err := Check(context.Background(), &fakeStore{w: w}, ref, nil)

			// Assert
			if err == nil || !strings.Contains(err.Error(), "reference") {
				t.Fatalf("Check() = %v, want an error naming the reference", err)
			}
		})
	}
}

func TestCheckReseedFailure(t *testing.T) {
	// Arrange
	w := newWorld()
	store := &fakeStore{w: w, reseedErr: errors.New("db down")}

	// Act
	err := Check(context.Background(), store, &fakeLib{w: w, name: "reference"}, nil)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("Check() = %v, want the reseed error", err)
	}
}

func TestCheckReseedsWhenDone(t *testing.T) {
	// Arrange
	w := newWorld()
	store := &fakeStore{w: w}
	subject := &fakeLib{w: w, name: "good"}

	// Act
	err := Check(context.Background(), store, &fakeLib{w: w, name: "reference"}, []bench.Adapter{subject})

	// Assert
	if err != nil {
		t.Fatalf("Check() = %v", err)
	}
	if len(w.models) != fakeModels {
		t.Fatalf("models after Check = %d, want the reseeded %d", len(w.models), fakeModels)
	}
}
