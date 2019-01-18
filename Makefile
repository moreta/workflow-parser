all: test cmd/parser

dep:
	dep ensure

test:
	go test ./parser

cmd/parser: cmd/main.go
	go build -o $@ $<

clean:
	rm -f cmd/parser
