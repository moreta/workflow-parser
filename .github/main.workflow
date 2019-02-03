workflow "on push" {
	on = "push"
	resolves = "go-ci"
}

action "go-ci" {
	uses = "cedrickring/golang-action@92b89ca0095ea0972cefaaaf1d48966d7c958553"
}
