server: *.go
	go build

.PHONY: live
live:
	find . -name '*.go' | entr -r make lint run

.PHONY: run
run: server
	./server -v -s

.PHONY: lint
lint:
	gofmt -w *.go
