package model

import (
	"fmt"
)

type uses interface {
	fmt.Stringer
	isUses()
}

// UsesDockerRegistry represents `uses = "docker://image"`
type UsesDockerImage struct {
	Image string
}

// UsesRepository represents `uses = "<owner>/<repository>[/path]@<ref>"`
type UsesRepository struct {
	Repository string
	Path       string
	Ref        string
}

// UsesPath represents `uses = "./<path>"`
type UsesPath struct {
	Path string
}

func (u *UsesDockerImage) isUses() {}
func (u *UsesRepository) isUses()  {}
func (u *UsesPath) isUses()        {}

func (u *UsesDockerImage) String() string {
	return fmt.Sprintf("docker://%s", u.Image)
}

func (u *UsesRepository) String() string {
	if u.Path == "" {
		return fmt.Sprintf("%s@%s", u.Repository, u.Ref)
	}

	return fmt.Sprintf("%s/%s@%s", u.Repository, u.Path, u.Ref)
}

func (u *UsesPath) String() string {
	return u.Path
}

// ActionUsesForm is which of the "uses" forms specified by an action.
type ActionUsesForm string

const (
	// InRepoUsesForm describes an Action referring to code in the same repository as its workflow.
	InRepoUsesForm ActionUsesForm = "in_repo"

	// CrossRepoUsesForm describes an Action that refers to code in a specific repository from its workflow file.
	CrossRepoUsesForm ActionUsesForm = "cross_repo"

	// DockerImageUsesForm describes an Action that refers to a Docker image.
	DockerImageUsesForm ActionUsesForm = "docker"

	// UnknownDockerImageUses describes an Unknown type of Action.
	UnknownDockerImageUses ActionUsesForm = "unknown"
)
