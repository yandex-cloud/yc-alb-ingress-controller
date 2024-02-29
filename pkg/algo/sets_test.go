package algo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetXOR(t *testing.T) {
	for _, entry := range []struct {
		desc          string
		lhs, rhs, exp map[string]struct{}
	}{
		{
			desc: "both empty",
			lhs:  make(map[string]struct{}),
			rhs:  make(map[string]struct{}),
			exp:  make(map[string]struct{}),
		},
		{
			desc: "left empty",
			lhs:  make(map[string]struct{}),
			rhs: map[string]struct{}{
				"smth": {},
			},
			exp: map[string]struct{}{
				"smth": {},
			},
		},
		{
			desc: "right empty",
			lhs: map[string]struct{}{
				"smth1": {},
				"smth2": {},
			},
			rhs: make(map[string]struct{}),
			exp: map[string]struct{}{
				"smth1": {},
				"smth2": {},
			},
		},
		{
			desc: "equal",
			lhs: map[string]struct{}{
				"smth1": {},
				"smth2": {},
			},
			rhs: map[string]struct{}{
				"smth1": {},
				"smth2": {},
			},
			exp: make(map[string]struct{}),
		},
		{
			desc: "basic",
			lhs: map[string]struct{}{
				"smth1": {},
				"smth2": {},
			},
			rhs: map[string]struct{}{
				"smth1": {},
				"smth3": {},
			},
			exp: map[string]struct{}{
				"smth2": {},
				"smth3": {},
			},
		},
	} {
		act := SetsExceptUnion(entry.lhs, entry.rhs)
		assert.Equal(t, entry.exp, act)
	}
}
