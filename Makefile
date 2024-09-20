.PHONY: live
live:
	printf 'laneya.go\nindex.html\nstyle.css\nmain.js\n' | entr -r make lint run

.PHONY: run
run: laneya
	./laneya -v

laneya: laneya.go index.html style.css main.js
	go build $<

.PHONY: lint
lint:
	gofmt -w *.go
