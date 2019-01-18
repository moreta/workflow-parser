# GitHub Actions Workflow Parser

This repository contains the official parser for GitHub Actions
`main.workflow` files, along with unit tests and a command-line driver.

# Using the parser

```go
import "github.com/github/actions-parser/parser"
...
config, err := parser.Parse(fileName)
```

`parser.Parse` returns an error only if some system error was encountered.
Problems with the contents of the file will be returned in the
`config.Errors` array, so that several errors may be indicated at once.

If the file is parsed with no fatal errors, `config.Actions` and
`config.Workflows` will contain objects representing all the `action` and
`workflow` blocks in the file.


# Developing

You'll need a copy of go v1.9 or higher.  You might also want a copy of
dep.

On OS X, `brew install go dep` will get you there.

To run the tests and build a command-line binary that validates workflow
files, run `make`.  The resulting validation binary works like so:

```
$ ./bin/parser foo.workflow
foo.workflow is a valid file with 9 actions and 1 workflow
```
