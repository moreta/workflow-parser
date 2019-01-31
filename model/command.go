package model

import (
	"strings"
)

// Command represents the optional "runs" and "args" attributes.
// Each one takes one of two forms:
//   - runs="entrypoint arg1 arg2 ..."
//   - runs=[ "entrypoint", "arg1", "arg2", ... ]
// If the user uses the string form, the StringCommand type should be used.
// If the user uses the list form, the ListCommand type should be used.
// Both types have methods to either split or join depending on the use.
type Command interface {
	isCommand()
	Split() []string
	Join() string
}

type StringCommand struct {
	Value string
}

type ListCommand struct {
	Values []string
}

func (s *StringCommand) isCommand() {}
func (l *ListCommand) isCommand()   {}

func (s *StringCommand) Split() []string {
	return strings.Fields(s.Value)
}

func (l *ListCommand) Split() []string {
	return l.Values
}

func (s *StringCommand) Join() string {
	return s.Value
}

func (l *ListCommand) Join() string {
	return strings.Join(l.Values, " ")
}
