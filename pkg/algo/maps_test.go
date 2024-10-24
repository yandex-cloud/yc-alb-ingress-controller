package algo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapMerge(t *testing.T) {
	testData := []struct {
		name      string
		a, b, exp map[string]int
		wantErr   bool
	}{
		{
			name:    "OK 1",
			a:       map[string]int{"a": 1},
			b:       nil,
			exp:     map[string]int{"a": 1},
			wantErr: false,
		},
		{
			name:    "OK 2",
			a:       map[string]int{"a": 1},
			b:       map[string]int{"b": 2},
			exp:     map[string]int{"a": 1, "b": 2},
			wantErr: false,
		},
		{
			name:    "OK 3",
			a:       map[string]int{"a": 1, "b": 2},
			b:       map[string]int{"b": 2, "c": 3},
			exp:     map[string]int{"a": 1, "b": 2, "c": 3},
			wantErr: false,
		},
		{
			name:    "Conflict",
			a:       map[string]int{"a": 1, "b": 3},
			b:       map[string]int{"b": 2, "c": 3},
			wantErr: true,
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapMerge(tt.a, tt.b)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.exp, got)
		})
	}
}
