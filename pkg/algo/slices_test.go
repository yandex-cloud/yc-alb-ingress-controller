package algo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlicesContainSameElements(t *testing.T) {
	for _, entry := range map[string]struct {
		lhs, rhs []int
		exp      bool
	}{
		"empty": {
			lhs: []int{},
			rhs: []int{},
			exp: true,
		},
		"basic true": {
			lhs: []int{1, 2, 3},
			rhs: []int{1, 2, 3},
			exp: true,
		},
		"basic false": {
			lhs: []int{1, 2, 3},
			rhs: []int{1, 2, 4},
			exp: false,
		},
		"duplicates true": {
			lhs: []int{1, 2, 2, 3},
			rhs: []int{1, 1, 2, 3},
			exp: true,
		},
		"duplicates false": {
			lhs: []int{1, 2, 2, 3, 3},
			rhs: []int{1, 1, 2, 3, 4},
			exp: false,
		},
	} {
		act := ContainSameElements(entry.lhs, entry.rhs)
		assert.Equal(t, entry.exp, act)
	}
}
