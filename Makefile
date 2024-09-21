.PHONY: live
live:
	echo laneya.go | entr -r make lint run

.PHONY: run
run: laneya
	./laneya -v -s

laneya: laneya.go
	go build $<

.PHONY: lint
lint:
	gofmt -w *.go
