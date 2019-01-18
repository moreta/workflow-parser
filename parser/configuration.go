package parser

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/hcl/ast"
)

// Configuration is a parsed main.workflow file, with some helpers that the deployer wants.
type Configuration struct {
	Actions   []*Action
	Workflows []*Workflow
	Errors    []*Error
	Version   int

	posMap map[interface{}]ast.Node
}

// Action represents a single "action" stanza in a .workflow file.
type Action struct {
	Identifier string
	Uses       ActionUses
	Runs, Args ActionCommand
	Needs      []string
	Env        map[string]string
	Secrets    []string
}

// ActionCommand represents the optional "runs" and "args" attributes.
// Each one takes one of two forms:
//   - runs="entrypoint arg1 arg2 ..."
//   - runs=[ "entrypoint", "arg1", "arg2", ... ]
// If the user uses the string form, "Raw" contains that value, and
// "Parsed" contains an array of the string value split at whitespace.
// If the user uses the array form, "Raw" is empty, and "Parsed" contains
// the array.
type ActionCommand struct {
	Raw    string
	Parsed []string
}

// Workflow represents a single "workflow" stanza in a .workflow file.
type Workflow struct {
	Identifier string
	On         string
	Resolves   []string
}

// Error represents an error identified by the parser, either syntactic
// (HCL) or semantic (.workflow) in nature.  There are fields for location
// (File, Line, Column), severity, and base error string.  The `Error()`
// function on this type concatenates whatever bits of the location are
// available with the message.  The severity is only used for filtering.
type Error struct {
	message  string
	Pos      ErrorPos
	Severity Severity
}

// ErrorPos represents the location of an error in a user's workflow
// file(s).
type ErrorPos struct {
	File   string
	Line   int
	Column int
}

// newFatal creates a new error at the FATAL level, indicating that the
// file is so broken it should not be displayed.
func newFatal(pos ErrorPos, format string, a ...interface{}) *Error {
	return &Error{
		message:  fmt.Sprintf(format, a...),
		Pos:      pos,
		Severity: FATAL,
	}
}

// newError creates a new error at the ERROR level, indicating that the
// file can be displayed but cannot be run.
func newError(pos ErrorPos, format string, a ...interface{}) *Error {
	return &Error{
		message:  fmt.Sprintf(format, a...),
		Pos:      pos,
		Severity: ERROR,
	}
}

// newWarning creates a new error at the WARNING level, indicating that
// the file might be runnable but might not execute as intended.
func newWarning(pos ErrorPos, format string, a ...interface{}) *Error {
	return &Error{
		message:  fmt.Sprintf(format, a...),
		Pos:      pos,
		Severity: WARNING,
	}
}

func (e *Error) Error() string {
	var sb strings.Builder
	if e.Pos.Line != 0 {
		sb.WriteString("Line ")                  // nolint: errcheck
		sb.WriteString(strconv.Itoa(e.Pos.Line)) // nolint: errcheck
		sb.WriteString(": ")                     // nolint: errcheck
	}
	if sb.Len() > 0 {
		sb.WriteString(e.message) // nolint: errcheck
		return sb.String()
	}
	return e.message
}

const (
	// WARNING indicates a mistake that might affect correctness
	WARNING = iota

	// ERROR indicates a mistake that prevents execution of any workflows in the file
	ERROR

	// FATAL indicates a mistake that prevents even drawing the file
	FATAL
)

// Severity represents the level of an error encountered while parsing a
// workflow file.  See the comments for WARNING, ERROR, and FATAL, above.
type Severity int

// GetAction looks up action by identifier.
//
// If the action is not found, nil is returned.
func (c *Configuration) GetAction(id string) *Action {
	for _, action := range c.Actions {
		if action.Identifier == id {
			return action
		}
	}
	return nil
}

// GetWorkflow looks up a workflow by identifier.
//
// If the workflow is not found, nil is returned.
func (c *Configuration) GetWorkflow(id string) *Workflow {
	for _, workflow := range c.Workflows {
		if workflow.Identifier == id {
			return workflow
		}
	}
	return nil
}

// GetWorkflows gets all Workflow structures that match a given type of event.
// e.g., GetWorkflows("push")
func (c *Configuration) GetWorkflows(eventType string) []*Workflow {
	var ret []*Workflow
	for _, workflow := range c.Workflows {
		if IsMatchingEventType(workflow.On, eventType) {
			ret = append(ret, workflow)
		}
	}
	return ret
}

// FirstError searches a Configuration for the first error at or above a
// given severity level.  Checking the return value against nil is a good
// way to see if the file has any errors at or above the given severity.
// A caller intending to execute the file might check for
// `c.FirstError(parser.WARNING)`, while a caller intending to display
// the file might check for `c.FirstError(parser.FATAL)`.
func (c *Configuration) FirstError(severity Severity) error {
	for _, e := range c.Errors {
		if e.Severity >= severity {
			return e
		}
	}
	return nil
}

type byLineNumber []*Error

func (a byLineNumber) Len() int           { return len(a) }
func (a byLineNumber) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byLineNumber) Less(i, j int) bool { return a[i].Pos.Line < a[j].Pos.Line }

// SortErrors sorts the errors reported by the parser.  Do this after
// parsing is complete.  The sort is stable, so order is preserved within
// a single line: left to right, syntax errors before validation errors.
func (c *Configuration) SortErrors() {
	sort.Stable(byLineNumber(c.Errors))
}
