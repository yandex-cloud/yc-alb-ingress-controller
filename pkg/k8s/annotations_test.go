package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseAnnTestCase struct {
	name    string
	value   string
	exp     map[string]string
	wantErr bool
}

var parseAnnTestCases = []parseAnnTestCase{
	{
		name:    "empty",
		value:   "",
		exp:     nil,
		wantErr: false,
	},
	{
		name:    "valid",
		value:   "key1=value1,key2=value2",
		exp:     map[string]string{"key1": "value1", "key2": "value2"},
		wantErr: false,
	},
	{
		name:    "invalid",
		value:   "key1=value1,key2",
		wantErr: true,
	},
	{
		name:    "empty key",
		value:   "key1=value1,=value2",
		wantErr: true,
	},
	{
		// History snapshot
		name:    "invalid commas",
		value:   "key1=value1,value2",
		wantErr: true,
	},
	{
		// History snapshot
		name:    "invalid equals",
		value:   "key1=1+1=5",
		wantErr: true,
	},
}

func TestParseConfigsFromAnnotationValue(t *testing.T) {
	for _, tc := range parseAnnTestCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ParseConfigsFromAnnotationValue(tc.value)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, actual)
		})
	}
}

func TestParseModifyHeadersFromAnnotationValue(t *testing.T) {
	var testCases []parseAnnTestCase
	testCases = append(testCases, parseAnnTestCases...)
	testCases = append(testCases,
		[]parseAnnTestCase{
			{
				name:    "headers with commas",
				value:   "X-Robots-Tag=noarchive,X-Robots-Tag=nofollow,X-Robots-Tag=noindex",
				exp:     map[string]string{"X-Robots-Tag": "noarchive,nofollow,noindex"},
				wantErr: false,
			},
			{
				name:    "few headers with commas",
				value:   "h1=v1,h3=v5,h2=v4,h1=v2,h2=v3",
				exp:     map[string]string{"h1": "v1,v2", "h2": "v4,v3", "h3": "v5"},
				wantErr: false,
			},
		}...,
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ParseModifyHeadersFromAnnotationValue(tc.value)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, actual)
		})
	}
}
