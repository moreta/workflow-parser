package parser

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	hclparser "github.com/hashicorp/hcl/hcl/parser"
	"github.com/hashicorp/hcl/hcl/token"
	"github.com/soniakeys/graph"
	"github.com/github/actions-parser/model"
)

const minVersion = 0
const maxVersion = 0
const maxSecrets = 100

type parseState struct {
  Version   int
  Actions   []*model.Action
  Workflows []*model.Workflow
  Errors    ErrorList

  posMap map[interface{}]ast.Node
}

// Parse parses a .workflow file and return the actions and global variables found within.
//
// Parameters:
//  - reader - an opened main.workflow file
//
// Returns: a model.Configuration
//
// A note about error handling: although Parse returns an error, the only
// errors handled that way are genuine surprises like I/O errors.  Parse
// errors are all handled by being appended to the Errors field of the
// returned model.Configuration object.  The caller can enumerate all
// errors and filter them by severity to see if it makes sense to proceed
// with displaying or executing the workflows in the file.
func Parse(reader io.Reader) (*model.Configuration, ErrorList, error) {
	// FIXME - check context for deadline?
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}

	root, err := hcl.ParseBytes(b)
	if err != nil {
		if posError, ok := err.(*hclparser.PosError); ok {
			return &model.Configuration{}, ErrorList{
				newError(
					ErrorPos{File: posError.Pos.Filename, Line: posError.Pos.Line, Column: posError.Pos.Column},
					posError.Err.Error())}, nil
		}
		return &model.Configuration{}, ErrorList{
			newError(ErrorPos{}, err.Error())}, nil
	}

	parseState := parseAndValidate(root.Node)
	return &model.Configuration{
		Version:   parseState.Version,
		Actions:   parseState.Actions,
		Workflows: parseState.Workflows,
	}, parseState.Errors, nil
}

// parseAndValidate converts a HCL AST into a parseState and validates
// high-level structure.
// Parameters:
//  - root - the contents of a .workflow file, as AST
// Returns:
//  - a parseState structure containing actions and workflow definitions
func parseAndValidate(root ast.Node) *parseState {
	c := parseOnly(root)
	c.validate()
	c.Errors.sort()

	return c
}

// parseOnly traverses the AST of a HCL and constructs a parseState.
// Syntax and low-level semantic errors are flagged, but file-level
// structural properties are not checked.
//
// Parameters:
//  - root - the contents of a .workflow file, as []byte
// Returns:
//  - a parseState structure containing actions, workflow definitions, and global
//    variables
func parseOnly(root ast.Node) *parseState {
	c := &parseState{
		posMap: make(map[interface{}]ast.Node),
	}
	c.parseRoot(root)

	return c
}

func (c *parseState) validate() {
	c.analyzeDependencies()
	c.checkCircularDependencies()
	c.checkActions()
	c.checkFlows()
}

func uniqStrings(items []string) []string {
	seen := make(map[string]bool)
	ret := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			ret = append(ret, item)
		}
	}
	return ret
}

// checkCircularDependencies finds loops in the action graph.
// It emits a fatal error for each cycle it finds, in the order (top to
// bottom, left to right) they appear in the .workflow file.
func (c *parseState) checkCircularDependencies() {
	// make a map from action name to node ID, which is the index in the c.Actions array
	// That is, c.Actions[actionmap[X]].Identifier == X
	actionmap := make(map[string]graph.NI)
	for i, action := range c.Actions {
		actionmap[action.Identifier] = graph.NI(i)
	}

	// make an adjacency list representation of the action dependency graph
	adjList := make(graph.AdjacencyList, len(c.Actions))
	for i, action := range c.Actions {
		adjList[i] = make([]graph.NI, 0, len(action.Needs))
		for _, depName := range action.Needs {
			if depIdx, ok := actionmap[depName]; ok {
				adjList[i] = append(adjList[i], depIdx)
			}
		}
	}

	// find cycles, and print a fatal error for each one
	g := graph.Directed{AdjacencyList: adjList}
	g.Cycles(func(cycle []graph.NI) bool {
		node := c.posMap[&c.Actions[cycle[len(cycle)-1]].Needs]
		c.Errors = append(c.Errors,
			newFatal(posFromNode(node), "Circular dependency on `%s'", c.Actions[cycle[0]].Identifier))
		return true
	})
}

// checkActions returns error if any actions are syntactically correct but
// have structural errors
func (c *parseState) checkActions() {
	secrets := make(map[string]bool)
	for _, t := range c.Actions {
		// Ensure the Action has a `uses` attribute
		if t.Uses.Raw == "" {
			c.Errors = append(c.Errors, newError(posFromNode(c.posMap[t]), "Action `%s' must have a `uses' attribute", t.Identifier))
			// continue, checking other actions
		}

		// Ensure there aren't too many secrets
		for _, str := range t.Secrets {
			if !secrets[str] {
				secrets[str] = true
				if len(secrets) == maxSecrets+1 {
					c.Errors = append(c.Errors,
						newError(posFromNode(c.posMap[&t.Secrets]), "All actions combined must not have more than %d unique secrets", maxSecrets))
				}
			}
		}

		// Ensure that no environment variable or secret begins with
		// "GITHUB_", unless it's "GITHUB_TOKEN".
		// Also ensure that all environment variable names come from the legal
		// form for environment variable names.
		// Finally, ensure that the same key name isn't used more than once
		// between env and secrets, combined.
		for k := range t.Env {
			c.checkEnvironmentVariable(k, c.posMap[&t.Env])
		}
		secretVars := make(map[string]bool)
		for _, k := range t.Secrets {
			c.checkEnvironmentVariable(k, c.posMap[&t.Secrets])
			if _, found := t.Env[k]; found {
				c.Errors = append(c.Errors, newError(posFromNode(c.posMap[&t.Secrets]),
					"Secret `%s' conflicts with an environment variable with the same name", k))
			}
			if secretVars[k] {
				c.Errors = append(c.Errors, newWarning(posFromNode(c.posMap[&t.Secrets]),
					"Secret `%s' redefined", k))
			}
			secretVars[k] = true
		}
	}
}

var envVarChecker = regexp.MustCompile(`\A[A-Za-z_][A-Za-z_0-9]*\z`)

func (c *parseState) checkEnvironmentVariable(key string, node ast.Node) {
	if key != "GITHUB_TOKEN" && strings.HasPrefix(key, "GITHUB_") {
		c.Errors = append(c.Errors, newWarning(posFromNode(node),
			"Environment variables and secrets beginning with `GITHUB_' are reserved"))
	}
	if !envVarChecker.MatchString(key) {
		c.Errors = append(c.Errors, newWarning(posFromNode(node),
			"Environment variables and secrets must contain only A-Z, a-z, 0-9, and _ characters, got `%s'", key))
	}
}

// checkFlows appends an error if any workflows are syntactically correct but
// have structural errors
func (c *parseState) checkFlows() {
	actionmap := makeActionMap(c.Actions)
	for _, f := range c.Workflows {
		// make sure there's an `on` attribute
		if f.On == "" {
			c.Errors = append(c.Errors, newError(posFromNode(c.posMap[f]),
				"Workflow `%s' must have an `on' attribute", f.Identifier))
			// continue, checking other workflows
		} else if !model.IsAllowedEventType(f.On) {
			c.Errors = append(c.Errors, newError(posFromNode(c.posMap[&f.On]),
				"Workflow `%s' has unknown `on' value `%s'", f.Identifier, f.On))
			// continue, checking other workflows
		}

		// make sure that the actions that are resolved all exist
		for _, actionID := range f.Resolves {
			_, ok := actionmap[actionID]
			if !ok {
				c.Errors = append(c.Errors, newError(posFromNode(c.posMap[&f.Resolves]),
					"Workflow `%s' resolves unknown action `%s'", f.Identifier, actionID))
				// continue, checking other workflows
			}
		}
	}
}

func makeActionMap(actions []*model.Action) map[string]*model.Action {
	actionmap := make(map[string]*model.Action)
	for _, action := range actions {
		actionmap[action.Identifier] = action
	}
	return actionmap
}

// Fill in Action dependencies for all actions based on explicit dependencies
// declarations.
//
// c.Actions is an array of Action objects, as parsed.  The Action objects in
// this array are mutated, by setting Action.dependencies for each.
func (c *parseState) analyzeDependencies() {
	actionmap := makeActionMap(c.Actions)
	for _, action := range c.Actions {
		// analyze explicit dependencies for each "needs" keyword
		c.analyzeNeeds(action, actionmap)
	}

	// uniq all the dependencies lists
	for _, action := range c.Actions {
		if len(action.Needs) >= 2 {
			action.Needs = uniqStrings(action.Needs)
		}
	}
}

func (c *parseState) analyzeNeeds(action *model.Action, actionmap map[string]*model.Action) {
	for _, need := range action.Needs {
		_, ok := actionmap[need]
		if !ok {
			c.Errors = append(c.Errors, newError(posFromNode(c.posMap[&action.Needs]),
				"Action `%s' needs nonexistent action `%s'", action.Identifier, need))
			// continue, checking other actions
		}
	}
}

// literalToStringMap converts a object value from the AST to a
// map[string]string.  For example, the HCL `{ a="b" c="d" }` becomes the
// Go expression map[string]string{ "a": "b", "c": "d" }.
// If the value doesn't adhere to that format -- e.g.,
// if it's not an object, or it has non-assignment attributes, or if any
// of its values are anything other than a string, the function appends an
// appropriate error.
func (c *parseState) literalToStringMap(node ast.Node) map[string]string {
	obj, ok := node.(*ast.ObjectType)

	if !ok {
		c.Errors = append(c.Errors, newError(posFromNode(node), "Expected object, got %s", typename(node)))
		return nil
	}

	c.checkAssignmentsOnly(obj.List, "")

	ret := make(map[string]string)
	for _, item := range obj.List.Items {
		if !isAssignment(item) {
			continue
		}
		str, ok := c.literalToString(item.Val)
		if ok {
			key := c.identString(item.Keys[0].Token)
			if key != "" {
				if _, found := ret[key]; found {
					c.Errors = append(c.Errors, newWarning(posFromNode(node),
						"Environment variable `%s' redefined", key))
				}
				ret[key] = str
			}
		}
	}

	return ret
}

func (c *parseState) identString(t token.Token) string {
	switch t.Type {
	case token.STRING:
		return t.Value().(string)
	case token.IDENT:
		return t.Text
	default:
		c.Errors = append(c.Errors, newError(posFromToken(t),
			"Each identifier should be a string, got %s",
			strings.ToLower(t.Type.String())))
		return ""
	}
}

// literalToStringArray converts a list value from the AST to a []string.
// For example, the HCL `[ "a", "b", "c" ]` becomes the Go expression
// []string{ "a", "b", "c" }.
// If the value doesn't adhere to that format -- it's not a list, or it
// contains anything other than strings, the function appends an
// appropriate error.
// If promoteScalars is true, then values that are scalar strings are
// promoted to a single-entry string array.  E.g., "foo" becomes the Go
// expression []string{ "foo" }.
func (c *parseState) literalToStringArray(node ast.Node, promoteScalars bool) ([]string, bool) {
	literal, ok := node.(*ast.LiteralType)
	if ok {
		if promoteScalars && literal.Token.Type == token.STRING {
			return []string{literal.Token.Value().(string)}, true
		}
		c.Errors = append(c.Errors, newError(posFromNode(node), "Expected list, got %s", typename(node)))
		return nil, false
	}

	list, ok := node.(*ast.ListType)
	if !ok {
		c.Errors = append(c.Errors, newError(posFromNode(node), "Expected list, got %s", typename(node)))
		return nil, false
	}

	ret := make([]string, 0, len(list.List))
	for _, literal := range list.List {
		str, ok := c.literalToString(literal)
		if ok {
			ret = append(ret, str)
		}
	}

	return ret, true
}

// literalToString converts a literal value from the AST into a string.
// If the value isn't a scalar or isn't a string, the function appends an
// appropriate error and returns "", false.
func (c *parseState) literalToString(node ast.Node) (string, bool) {
	val := c.literalCast(node, token.STRING)
	if val == nil {
		return "", false
	}
	return val.(string), true
}

// literalToInt converts a literal value from the AST into an int64.
// Supported number formats are: 123, 0x123, and 0123.
// Exponents (1e6) and floats (123.456) generate errors.
// If the value isn't a scalar or isn't a number, the function appends an
// appropriate error and returns 0, false.
func (c *parseState) literalToInt(node ast.Node) (int64, bool) {
	val := c.literalCast(node, token.NUMBER)
	if val == nil {
		return 0, false
	}
	return val.(int64), true
}

func (c *parseState) literalCast(node ast.Node, t token.Type) interface{} {
	literal, ok := node.(*ast.LiteralType)
	if !ok {
		c.Errors = append(c.Errors, newError(posFromNode(node),
			"Expected %s, got %s", strings.ToLower(t.String()), typename(node)))
		return nil
	}

	if literal.Token.Type != t {
		c.Errors = append(c.Errors, newError(posFromNode(node),
			"Expected %s, got %s", strings.ToLower(t.String()), typename(node)))
		return nil
	}

	return literal.Token.Value()
}

// parseRoot parses the root of the AST, filling in c.Version, c.Actions,
// and c.Workflows.
func (c *parseState) parseRoot(node ast.Node) {
	objectList, ok := node.(*ast.ObjectList)
	if !ok {
		// It should be impossible for HCL to return anything other than an
		// ObjectList as the root node.  This error should never happen.
		c.Errors = append(c.Errors, newError(posFromNode(node), "Internal error: root node must be an ObjectList"))
		return
	}

	c.Actions = make([]*model.Action, 0, len(objectList.Items))
	c.Workflows = make([]*model.Workflow, 0, len(objectList.Items))
	identifiers := make(map[string]bool)
	for idx, item := range objectList.Items {
		if item.Assign.IsValid() {
			c.parseVersion(idx, item)
			continue
		}
		c.parseBlock(item, identifiers)
	}
}

// parseBlock parses a single, top-level "action" or "workflow" block,
// appending it to c.Actions or c.Workflows as appropriate.
func (c *parseState) parseBlock(item *ast.ObjectItem, identifiers map[string]bool) {
	if len(item.Keys) != 2 {
		c.Errors = append(c.Errors, newError(posFromNode(item), "Invalid toplevel declaration"))
		return
	}

	cmd := c.identString(item.Keys[0].Token)
	var id string

	switch cmd {
	case "action":
		action := c.actionifyItem(item)
		if action != nil {
			id = action.Identifier
			c.Actions = append(c.Actions, action)
		}
	case "workflow":
		workflow := c.workflowifyItem(item)
		if workflow != nil {
			id = workflow.Identifier
			c.Workflows = append(c.Workflows, workflow)
		}
	default:
		c.Errors = append(c.Errors, newError(posFromNode(item), "Invalid toplevel keyword, `%s'", cmd))
		return
	}

	if identifiers[id] {
		c.Errors = append(c.Errors, newError(posFromNode(item), "Identifier `%s' redefined", id))
	}

	identifiers[id] = true
}

// parseVersion parses a top-level `version=N` statement, filling in
// c.Version.
func (c *parseState) parseVersion(idx int, item *ast.ObjectItem) {
	if len(item.Keys) != 1 || c.identString(item.Keys[0].Token) != "version" {
		// not a valid `version` declaration
		c.Errors = append(c.Errors, newError(posFromNode(item.Val), "Toplevel declarations cannot be assignments"))
		return
	}
	if idx != 0 {
		c.Errors = append(c.Errors, newError(posFromNode(item.Val), "`version` must be the first declaration"))
		return
	}
	version, ok := c.literalToInt(item.Val)
	if !ok {
		return
	}
	if version < minVersion || version > maxVersion {
		c.Errors = append(c.Errors, newError(posFromNode(item.Val), "`version = %d` is not supported", version))
		return
	}
	c.Version = int(version)
}

// parseIdentifier parses the double-quoted identifier (name) for a
// "workflow" or "action" block.
func (c *parseState) parseIdentifier(key *ast.ObjectKey) string {
	id := key.Token.Text
	if len(id) < 3 || id[0] != '"' || id[len(id)-1] != '"' {
		c.Errors = append(c.Errors, newError(posFromNode(key), "Invalid format for identifier `%s'", id))
		return ""
	}
	return id[1 : len(id)-1]
}

// parseRequiredString parses a string value, setting its value into the
// out-parameter `value` and returning true if successful.
func (c *parseState) parseRequiredString(value *string, val ast.Node, nodeType, name, id string) bool {
	if *value != "" {
		c.Errors = append(c.Errors, newWarning(posFromNode(val), "`%s' redefined in %s `%s'", name, nodeType, id))
		// continue, allowing the redefinition
	}

	newVal, ok := c.literalToString(val)
	if !ok {
		c.Errors = append(c.Errors, newError(posFromNode(val),
			"Invalid format for `%s' in %s `%s', expected string", name, nodeType, id))
		return false
	}

	if newVal == "" {
		c.Errors = append(c.Errors, newError(posFromNode(val),
			"`%s' value in %s `%s' cannot be blank", name, nodeType, id))
		return false
	}

	*value = newVal
	return true
}

// parseBlockPreamble parses the beginning of a "workflow" or "action"
// block.
func (c *parseState) parseBlockPreamble(item *ast.ObjectItem, nodeType string) (string, *ast.ObjectType) {
	id := c.parseIdentifier(item.Keys[1])
	if id == "" {
		return "", nil
	}

	node := item.Val
	obj, ok := node.(*ast.ObjectType)
	if !ok {
		c.Errors = append(c.Errors, newError(posFromNode(node), "Each %s must have an { ...  } block", nodeType))
		return "", nil
	}

	c.checkAssignmentsOnly(obj.List, id)

	return id, obj
}

// actionifyItem converts an AST block to an Action object.
func (c *parseState) actionifyItem(item *ast.ObjectItem) *model.Action {
	id, obj := c.parseBlockPreamble(item, "action")
	if obj == nil {
		return nil
	}

	action := &model.Action{
		Identifier: id,
	}
	c.posMap[action] = item

	for _, item := range obj.List.Items {
		c.parseActionAttribute(c.identString(item.Keys[0].Token), action, item.Val)
	}

	return action
}

// parseActionAttribute parses a single key-value pair from an "action"
// block.  This function rejects any unknown keys and enforces formatting
// requirements on all values.
// It also has higher-than-normal cyclomatic complexity, so we ask the
// gocyclo linter to ignore it.
// nolint: gocyclo
func (c *parseState) parseActionAttribute(name string, action *model.Action, val ast.Node) {
	switch name {
	case "uses":
		c.parseUses(action, val)
	case "needs":
		needs, ok := c.literalToStringArray(val, true)
		if ok {
			action.Needs = needs
			c.posMap[&action.Needs] = val
		}
	case "runs":
		c.parseCommand(action, &action.Runs, name, val, false)
	case "args":
		c.parseCommand(action, &action.Args, name, val, true)
	case "env":
		env := c.literalToStringMap(val)
		if env != nil {
			action.Env = env
		}
		c.posMap[&action.Env] = val
	case "secrets":
		secrets, ok := c.literalToStringArray(val, false)
		if ok {
			action.Secrets = secrets
			c.posMap[&action.Secrets] = val
		}
	default:
		c.Errors = append(c.Errors, newWarning(posFromNode(val), "Unknown action attribute `%s'", name))
	}
}

// parseUses sets the action.Uses value based on the contents of the AST
// node.  This function enforces formatting requirements on the value.
func (c *parseState) parseUses(action *model.Action, node ast.Node) {
	if action.Uses.Path != "" {
		c.Errors = append(c.Errors, newWarning(posFromNode(node),
			"`uses' redefined in action `%s'", action.Identifier))
		// continue, allowing the redefinition
	}
	strVal, ok := c.literalToString(node)
	if !ok {
		return
	}

	if strVal == "" {
		c.Errors = append(c.Errors, newError(posFromNode(node),
			"`uses' value in action `%s' cannot be blank", action.Identifier))
		return
	}
	action.Uses.Raw = strVal
	if strings.HasPrefix(strVal, "./") {
		action.Uses.Path = strVal
		// Repo and Ref left blank
		return
	}

	if strings.HasPrefix(strVal, "docker://") {
		action.Uses.Image = strings.TrimPrefix(strVal, "docker://")
		return
	}

	tok := strings.Split(strVal, "@")
	if len(tok) != 2 {
		c.Errors = append(c.Errors, newError(posFromNode(node),
			"The `uses' attribute must be a path, a Docker image, or owner/repo@ref"))
		return
	}
	ref := tok[1]
	tok = strings.SplitN(tok[0], "/", 3)
	if len(tok) < 2 {
		c.Errors = append(c.Errors, newError(posFromNode(node),
			"The `uses' attribute must be a path, a Docker image, or owner/repo@ref"))
		return
	}
	action.Uses.Ref = ref
	action.Uses.Repo = tok[0] + "/" + tok[1]
	if len(tok) == 3 {
		action.Uses.Path = "/" + tok[2]
	} else {
		action.Uses.Path = "/"
	}
}

// parseUses sets the action.Runs or action.Command value based on the
// contents of the AST node.  This function enforces formatting
// requirements on the value.
func (c *parseState) parseCommand(action *model.Action, dest *model.ActionCommand, name string, node ast.Node, allowBlank bool) {
	if len(dest.Parsed) > 0 {
		c.Errors = append(c.Errors, newWarning(posFromNode(node),
			"`%s' redefined in action `%s'", name, action.Identifier))
		// continue, allowing the redefinition
	}

	// Is it a list?
	if _, ok := node.(*ast.ListType); ok {
		if parsed, ok := c.literalToStringArray(node, false); ok {
			dest.Parsed = parsed
		}
		return
	}

	// If not, parse a whitespace-separated string into a list.
	var raw string
	var ok bool
	if raw, ok = c.literalToString(node); !ok {
		c.Errors = append(c.Errors, newError(posFromNode(node),
			"The `%s' attribute must be a string or a list", name))
		return
	}
	if raw == "" && !allowBlank {
		c.Errors = append(c.Errors, newError(posFromNode(node),
			"`%s' value in action `%s' cannot be blank", name, action.Identifier))
		return
	}
	dest.Raw = raw
	dest.Parsed = strings.Fields(raw)
}

func typename(val interface{}) string {
	switch cast := val.(type) {
	case *ast.ListType:
		return "list"
	case *ast.LiteralType:
		return strings.ToLower(cast.Token.Type.String())
	case *ast.ObjectType:
		return "object"
	default:
		return fmt.Sprintf("%T", val)
	}
}

// workflowifyItem converts an AST block to a Workflow object.
func (c *parseState) workflowifyItem(item *ast.ObjectItem) *model.Workflow {
	id, obj := c.parseBlockPreamble(item, "workflow")
	if obj == nil {
		return nil
	}

	var ok bool
	workflow := &model.Workflow{Identifier: id}
	for _, item := range obj.List.Items {
		name := c.identString(item.Keys[0].Token)

		switch name {
		case "on":
			ok = c.parseRequiredString(&workflow.On, item.Val, "workflow", name, id)
			if ok {
				c.posMap[&workflow.On] = item
			}
		case "resolves":
			if workflow.Resolves != nil {
				c.Errors = append(c.Errors, newWarning(posFromNode(item.Val),
					"`resolves' redefined in workflow `%s'", id))
				// continue, allowing the redefinition
			}
			workflow.Resolves, ok = c.literalToStringArray(item.Val, true)
			c.posMap[&workflow.Resolves] = item
			if !ok {
				c.Errors = append(c.Errors, newError(posFromNode(item.Val),
					"Invalid format for `resolves' in workflow `%s', expected list of strings", id))
				// continue, allowing workflow with no `resolves`
			}
		default:
			c.Errors = append(c.Errors, newWarning(posFromNode(item.Val),
				"Unknown workflow attribute `%s'", name))
			// continue, treat as no-op
		}
	}

	c.posMap[workflow] = item
	return workflow
}

func isAssignment(item *ast.ObjectItem) bool {
	return len(item.Keys) == 1 && item.Assign.IsValid()
}

// checkAssignmentsOnly ensures that all elements in the object are "key =
// value" pairs.
func (c *parseState) checkAssignmentsOnly(objectList *ast.ObjectList, actionID string) {
	for _, item := range objectList.Items {
		if !isAssignment(item) {
			var desc string
			if actionID == "" {
				desc = "the object"
			} else {
				desc = fmt.Sprintf("action `%s'", actionID)
			}
			c.Errors = append(c.Errors, newError(posFromObjectItem(item),
				"Each attribute of %s must be an assignment", desc))
			continue
		}

		child, ok := item.Val.(*ast.ObjectType)
		if ok {
			c.checkAssignmentsOnly(child.List, actionID)
		}
	}
}

// posFromNode returns an ErrorPos (file, line, and column) from an AST
// node, so we can report specific locations for each parse error.
func posFromNode(node ast.Node) ErrorPos {
	var pos *token.Pos
	switch cast := node.(type) {
	case *ast.ObjectList:
		if len(cast.Items) > 0 {
			if len(cast.Items[0].Keys) > 0 {
				pos = &cast.Items[0].Keys[0].Token.Pos
			}
		}
	case *ast.ObjectItem:
		return posFromNode(cast.Val)
	case *ast.ObjectType:
		pos = &cast.Lbrace
	case *ast.LiteralType:
		pos = &cast.Token.Pos
	case *ast.ListType:
		pos = &cast.Lbrack
	case *ast.ObjectKey:
		pos = &cast.Token.Pos
	}

	if pos == nil {
		return ErrorPos{}
	}
	return ErrorPos{File: pos.Filename, Line: pos.Line, Column: pos.Column}
}

// posFromObjectItem returns an ErrorPos from an ObjectItem.  This is for
// cases where posFromNode(item) would fail because the item has no Val
// set.
func posFromObjectItem(item *ast.ObjectItem) ErrorPos {
	if len(item.Keys) > 0 {
		return posFromNode(item.Keys[0])
	}
	return ErrorPos{}
}

// posFromToken returns an ErrorPos from a Token.  We can't use
// posFromNode here because Tokens aren't Nodes.
func posFromToken(token token.Token) ErrorPos {
	return ErrorPos{File: token.Pos.Filename, Line: token.Pos.Line, Column: token.Pos.Column}
}
