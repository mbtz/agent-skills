.PHONY: build release

build:
	go build ./cmd/askill

release:
	./update-version.sh
