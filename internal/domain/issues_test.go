package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestSortByPriority(t *testing.T) {
	tests := []struct {
		name    string
		input   []domain.Issue
		wantIDs []string
	}{
		{
			"urgent first",
			[]domain.Issue{{ID: "a", Priority: 4}, {ID: "b", Priority: 1}},
			[]string{"b", "a"},
		},
		{
			"zero last",
			[]domain.Issue{{ID: "a", Priority: 0}, {ID: "b", Priority: 3}},
			[]string{"b", "a"},
		},
		{
			"stable",
			[]domain.Issue{{ID: "a", Priority: 2}, {ID: "b", Priority: 2}},
			[]string{"a", "b"},
		},
		{
			"mixed",
			[]domain.Issue{{ID: "a", Priority: 3}, {ID: "b", Priority: 0}, {ID: "c", Priority: 1}},
			[]string{"c", "a", "b"},
		},
		{
			"empty",
			[]domain.Issue{},
			[]string{},
		},
		{
			"single",
			[]domain.Issue{{ID: "a", Priority: 5}},
			[]string{"a"},
		},
		{
			"all zero",
			[]domain.Issue{{ID: "a", Priority: 0}, {ID: "b", Priority: 0}},
			[]string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given — input slice (already set)

			// when
			domain.SortByPriority(tt.input)

			// then
			for i, want := range tt.wantIDs {
				if tt.input[i].ID != want {
					t.Errorf("pos %d: got %s, want %s", i, tt.input[i].ID, want)
				}
			}
		})
	}
}
