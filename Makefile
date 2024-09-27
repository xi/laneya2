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

.PHONY: install
install:
	install -Dm 755 server "${DESTDIR}/usr/bin/laneya"
	install -Dm 644 index.html "${DESTDIR}/var/www/laneya/index.html"
	install -Dm 644 static/main.js "${DESTDIR}/var/www/laneya/static/main.js"
	install -Dm 644 static/dpad.js "${DESTDIR}/var/www/laneya/static/dpad.js"
	install -Dm 644 static/style.css "${DESTDIR}/var/www/laneya/static/style.css"
	install -Dm 644 README.md "${DESTDIR}/usr/share/doc/laneya/README.md"
