workflow "on push" {
	on = "push"
	resolves = "go-ci"
}

action "go-ci" {
	uses = "piki/actions-go-builder@master"
}
