package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowedEventType(t *testing.T) {
	// This is not exhaustive. Rather, it's a demonstration of some of the interesting bits of the func.
	allowed := []string{
		"push",
		"PUSH",
		"pull_request",
		// TODO support filters:
		//"pull_request.open",
	}

	for _, s := range allowed {
		assert.True(t, IsAllowedEventType(s), "should allow %q", s)
	}

	// This is also not exhaustive. We want to have this done by universe, after all.
	notAllowed := []string{
		"installation",
		"randommashingofkeyboard",
	}

	for _, s := range notAllowed {
		assert.False(t, IsAllowedEventType(s), "should not allow %q", s)
	}
}

func TestIsMatchingEventType(t *testing.T) {
	examples := []struct {
		on            string
		hookEventType string
		match         bool
	}{
		// hookEventType will always be lower-case.
		{"PUSH", "push", true},
		{"push", "push", true},
		{"blahblah", "push", false},
		// TODO support filters:
		//{"pull_request.open", "pull_request", /*???*/ true},
	}

	for _, ex := range examples {
		assert.Equal(t, ex.match, IsMatchingEventType(ex.on, ex.hookEventType), "Should on=%q match a hook with event type %q?", ex.on, ex.hookEventType)
	}
}

func TestAddAndRemoveEventTypes(t *testing.T) {
	assert.False(t, IsAllowedEventType("bogus"))
	AddAllowedEventType("bogus")
	assert.True(t, IsAllowedEventType("bogus"))
	RemoveAllowedEventType("bogus")
	assert.False(t, IsAllowedEventType("bogus"))
}
