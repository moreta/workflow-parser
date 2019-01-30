package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsesStringer(t *testing.T) {
	cases := []struct {
		uses     fmt.Stringer
		expected string
	}{
		{
			uses:     &UsesRepository{Repository: "actions/workflow-parser", Path: "/", Ref: "master"},
			expected: "actions/workflow-parser@master",
		},
		{
			uses:     &UsesRepository{Repository: "actions/workflow-parser", Path: "/path", Ref: "master"},
			expected: "actions/workflow-parser/path@master",
		},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expected, tc.uses.String())
	}
}
