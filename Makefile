GO ?= go
.PHONY: build test vet race fmt fmt-check module-check clean
build:
	GO="$(GO)" python3 scripts/build_go.py
fmt:
	$(GO) fmt ./...
fmt-check:
	@files="$$(gofmt -l cmd internal)" && test -z "$$files" || { printf '%s\n' "$$files"; exit 1; }
test:
	$(GO) test ./...
vet:
	$(GO) vet ./...
race:
	$(GO) test -race ./...
module-check:
	$(GO) mod tidy -diff
	$(GO) mod verify
clean:
	rm -f dist/almadial
