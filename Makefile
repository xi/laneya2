run: laneya
	./laneya

laneya: laneya.go
	go build $<

.PHONY: lint
lint:
	gofmt -w *.go
