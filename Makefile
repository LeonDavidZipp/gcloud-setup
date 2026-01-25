.PHONY: build run lint fix fmt clean install test

build:
	go build -o gcsetup .

run: build
	./gcsetup

install: build
	sudo mv gcsetup /usr/local/bin/

lint:
	golangci-lint run

fix:
	golangci-lint run --fix

fmt:
	golangci-lint fmt

test:
	go test ./...

clean:
	rm -f gcsetup
