package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsesForm(t *testing.T) {
	cases := []struct {
		name         string
		action       string
		expectedForm ActionUsesForm
	}{
		{
			name:         "docker",
			action:       `action "a" { uses = "docker://alpine" }`,
			expectedForm: DockerImageUsesForm,
		},
		{
			name:         "in-repo",
			action:       `action "a" { uses = "./actions/foo" }`,
			expectedForm: InRepoUsesForm,
		},
		{
			name:         "cross-repo",
			action:       `action "a" { uses = "name/owner/path@5678ac" }`,
			expectedForm: CrossRepoUsesForm,
		},
		{
			name:         "cross-repo-no-path",
			action:       `action "a" { uses = "name/owner@5678ac" }`,
			expectedForm: CrossRepoUsesForm,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			workflow, err := Parse(strings.NewReader(tc.action))
			require.NoError(tt, err)
			assert.Equalf(tt, tc.expectedForm, workflow.Actions[0].Uses.Form(), "%+v", workflow.Actions[0].Uses)
		})
	}
}
