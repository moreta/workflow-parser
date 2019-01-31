package model

import (
	"strings"
)

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
