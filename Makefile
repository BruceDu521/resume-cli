GO ?= go
BIN := bin/resume-cli

.PHONY: build test race vet check demo clean
build:
	$(GO) build -trimpath -o $(BIN) ./cmd/resume-cli
test:
	GOPROXY=off $(GO) test -cover ./...
race:
	GOPROXY=off $(GO) test -race ./...
vet:
	GOPROXY=off $(GO) vet ./...
check: test race vet
demo: build
	$(BIN) parse testdata/resume-zh.pdf
	$(BIN) extract testdata/resume-zh.pdf --mock
	$(BIN) score testdata/resume-zh.pdf --jd testdata/jd.txt --mock
clean:
	rm -f $(BIN)
