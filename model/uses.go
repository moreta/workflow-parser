package model

// ActionUses represents the mandatory "uses" block in an action.
// It takes one of three forms:
//   - "./path"
//   - "owner/repo/path@ref", with the "/path" part optional
//   - "docker://image"
// The parser leaves the original value in the "Raw" field.
// The parser fills in the other parts of the struct when they're set.
// So, the first form leaves "Repo" and "Ref" empty, and the second form
// optionally leaves "Path" empty.
type Uses struct {
	Repo  string
	Path  string
	Ref   string
	Image string
	Raw   string
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

// Form returns a string describing the nature of the Action: in-repo, cross-repo, or docker image.
func (u Uses) Form() ActionUsesForm {
	if u.Image != "" {
		return DockerImageUsesForm
	}
	if u.Repo != "" {
		return CrossRepoUsesForm
	}
	if u.Path != "" {
		return InRepoUsesForm
	}
	return UnknownDockerImageUses
}
