# Overview

The Actions Workflow language describes workflows, which map repository
events to sequences of actions that run in response to those events.  It
is based on Hashicorp HCL but supports only a subset of HCL features.
This document specifies that subset.

# By example

```
# Workflow files can have comments, which begin with a # character and
# continue to the end of the line.  Comments may be on their own line, or
# may appear to the right of real content.
#
# Workflow files can have a version specifier, which must appear before
# any other (non-blank, non-comment) content.  Currently, the only legal
# version is 0.
version = 0

# Workflow files contain one or more workflows, which map an event to one
# or more actions that the workflow resolves.  Each workflow has a name,
# which is a double-quoted string.  UTF-8 characters and C-style escapes
# are supported in the string.  The workflow name is displayed to the user
# but has no other form or meaning.  Following the workflow name is a
# block of key-value pairs, contained in curly braces.
workflow "this happens when I push" {
  # Each key-value pair is an identifier, an equals sign, and a value of
  # the correct type for that identifier.  Only specific identifiers are
  # allowed, and no key may appear twice in the same block.  For
  # workflows, those allowed identifiers are: on and resolves.  The "on"
  # key is required; "resolves" is optional.

  # "on" identifies the event that will cause Actions to run this
  # workflow.  It's value is a double-quoted string, case-insensitive,
  # drawn from the list of known event types.
  on = "fork"

  # "resolves" identifies one or more actions that will be resolved when
  # the given event occurs.  Resolving an action means running all
  # (transitive) prerequisites for that action and then running the action
  # itself.  Any failed action prevents subsequent actions from running.
  # The value for resolves may be either a string or an array of strings.
  # Arrays are designated with square brackets and commas.
  resolves = [ "goal1", "goal2" ]
}

# Workflow files also contain one or more actions.  Like workflows,
# actions have a double-quoted string name and a block of key-value pairs.
# The name for an action can be any user-selected string, which must match
# actions that workflows resolve.
action "goal1" {
  # The valid keys in an action block are: uses, needs, runs, args, env,
  # and secrets.  The uses key is required; all others are optional.

  # The "uses" keyword identifies what actual code this action will run.
  # The value is always a string, and may take three forms:
  #   - "./path", which identifies a Dockerfile in the current repository
  #   - "owner/repo/path@ref", which identifies a Dockerfile in another
  #     GitHub repository.  Ref may be a branch, tag, or SHA.
  #   - "docker://image", which identifies a docker image
  uses = "docker://alpine"
  # or: uses = "./local-directory"
  # or: uses = "actions/bin/filter@master"

  # The "needs" keyword identifies one or more actions that must complete
  # successfully before this action can begin.  The value can be a string
  # or an array of strings.
  needs = "ci"

  # The "runs" keyword identifies a command to run in the action's Docker
  # container.  Its value can be either a string or an array of strings.
  # If the value is a string, Actions will parse it by separating at
  # whitespace.  If the value is an array, the array elements are passed
  # literally to Docker with no further parsing.
  runs = "echo hello"

  # The "args" keyword identifies arguments to attach to whatever command
  # is running in the Docker container.  If the container has an
  # entrypoint, the arguments will be appended to it.  In the rare case
  # where an action specifies with "runs" and "args" keywords, their
  # contents are appended (runs, then args) and passed as the command to
  # run in the container.  The "args" value may be either a string or an
  # array.  If it is a string, it is parsed by separating at whitespace.
  args = [ "world" ]

  # The "env" keyword identifies environment variables that will be
  # present in the Docker container for this action.  The value is a hash
  # (object), surrounded by braces, mapping environment variable names to
  # environment variable values.  Both names and values are strings.
  # Environment variables beginning with "GITHUB_" are reserved.
  env = {
    KEY = "VALUE"
    KEY2 = "VALUE2"
  }

  # The "secrets" keyword identifies secrets that will be present as
  # environment variables in the container.  The values of these secrets
  # are stored elsewhere, not in .workflow files, so only the names of
  # secrets appear here.  The value is an array of strings.  No secret can
  # have the same name as an environment variable.  The number of secrets
  # allowed in an entire workflow file is currently limited to 100.
  secrets = [ "GITHUB_TOKEN" ]
}

# Each action named in a "resolves" or "needs" key must be present in the
# file.  Circular dependencies are prohibited.
action "ci" {
  uses = "./ci"
}

action "goal2" {
  uses = "docker://alpine"
  runs = "echo howdy"
}
```

# Grammar

```
WORKFLOW_FILE ::= [VERSION] BLOCK*

VERSION ::= "version" "=" INTEGER

INTEGER ::= "0" | /^ [1-9][0-9]* $/x

BLOCK ::= WORKFLOW | ACTION

WORKFLOW ::= "workflow" STRING "{" WORKFLOW_KVP* "}"

# STRING is a double-quoted UTF-8 string with optional escape characters,
# as used in HCL, Go, and C, among others.

WORKFLOW_KVP ::= ON_KVP | RESOLVES_KVP

ON_KVP ::= "on" "=" EVENT_STRING

EVENT_STRING ::= STRING   # from the allowable set

RESOLVES_KVP ::= "resolves" "=" STRING_OR_ARRAY

STRING_OR_ARRAY ::= STRING | STRING_ARRAY

STRING_ARRAY ::= EMPTY_ARRAY | "[" ( STRING "," )* STRING "]"

EMPTY_ARRAY ::= "[" "]"

ACTION ::= "action" STRING "{" ACTION_KVP* "}"

ACTION_KVP ::= USES_KVP | NEEDS_KVP | RUNS_KVP | ARGS_KVP | ENV_KVP | SECRETS_KVP

USES_KVP ::= "uses" "=" USES_STRING

USES_STRING ::= /^ " ( \.\/\.* | \w+\/\w+\/.*@\w+ | docker:\/\/.* " $/x

NEEDS_KVP ::= "needs" "=" STRING_OR_ARRAY

RUNS_KVP ::= "runs" "=" STRING_OR_ARRAY

ARGS_KVP ::= "args" "=" STRING_OR_ARRAY

ENV_KVP ::= "env" "=" "{" ENV_VAR* "}"

ENV_VAR ::= IDENTIFIER "=" STRING

IDENTIFIER ::= /^ [_a-z] [_0-9a-z]* $/xi

SECRETS_KVP ::= "secrets" "=" IDENT_STRING_ARRAY

IDENT_STRING_ARRAY ::= EMPTY_ARRAY | "[" ( IDENT_STRING "," )* IDENT_STRING "]"

IDENT_STRING = "[" IDENTIFIER "]"
```
