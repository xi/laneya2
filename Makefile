run: laneya
	./laneya

laneya: laneya.go index.html style.css main.js
	go build $<

.PHONY: lint
lint:
	gofmt -w *.go
