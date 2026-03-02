package paintress

import "testing"

func TestSortByPriority(t *testing.T) {
	tests := []struct {
		name    string
		input   []Issue
		wantIDs []string
	}{
		{
			"urgent first",
			[]Issue{{ID: "a", Priority: 4}, {ID: "b", Priority: 1}},
			[]string{"b", "a"},
		},
		{
			"zero last",
			[]Issue{{ID: "a", Priority: 0}, {ID: "b", Priority: 3}},
			[]string{"b", "a"},
		},
		{
			"stable",
			[]Issue{{ID: "a", Priority: 2}, {ID: "b", Priority: 2}},
			[]string{"a", "b"},
		},
		{
			"mixed",
			[]Issue{{ID: "a", Priority: 3}, {ID: "b", Priority: 0}, {ID: "c", Priority: 1}},
			[]string{"c", "a", "b"},
		},
		{
			"empty",
			[]Issue{},
			[]string{},
		},
		{
			"single",
			[]Issue{{ID: "a", Priority: 5}},
			[]string{"a"},
		},
		{
			"all zero",
			[]Issue{{ID: "a", Priority: 0}, {ID: "b", Priority: 0}},
			[]string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given — input slice (already set)

			// when
			SortByPriority(tt.input)

			// then
			for i, want := range tt.wantIDs {
				if tt.input[i].ID != want {
					t.Errorf("pos %d: got %s, want %s", i, tt.input[i].ID, want)
				}
			}
		})
	}
}
