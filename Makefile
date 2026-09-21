GO ?= go
BIN := bin/resume-cli

.PHONY: build test race vet check demo clean
build:
	$(GO) build -trimpath -o $(BIN) ./cmd/resume-cli
test:
	$(GO) test -cover ./...
race:
	$(GO) test -race ./...
vet:
	$(GO) vet ./...
check: test race vet
demo: build
	$(BIN) parse testdata/resume-zh.pdf
	$(BIN) extract testdata/resume-zh.pdf --mock
	$(BIN) score testdata/resume-zh.pdf --jd testdata/jd.txt --mock
clean:
	rm -f $(BIN)

# Docker shortcuts use files in the repository root; never copy keys into images.
.PHONY: docker-build docker-demo docker-parse docker-extract docker-score
docker-build:
	docker build -t resume-cli .
docker-demo:
	@docker run --rm --network none resume-cli score /examples/resume-zh.pdf --jd /examples/jd.txt --mock
docker-parse:
	@test -f resume.pdf || { echo "Missing resume.pdf in the current directory" >&2; exit 1; }
	@docker run --rm --network none --user "$$(id -u):$$(id -g)" \
		--mount "type=bind,src=$$PWD/resume.pdf,dst=/work/resume.pdf,readonly" \
		resume-cli parse /work/resume.pdf
docker-extract:
	@test -f resume.pdf || { echo "Missing resume.pdf in the current directory" >&2; exit 1; }
	@test -f .env || { echo "Missing .env; copy .env.example and configure your provider/key" >&2; exit 1; }
	@docker run --rm --user "$$(id -u):$$(id -g)" --env-file .env \
		--mount "type=bind,src=$$PWD/resume.pdf,dst=/work/resume.pdf,readonly" \
		resume-cli extract /work/resume.pdf
docker-score:
	@test -f resume.pdf || { echo "Missing resume.pdf in the current directory" >&2; exit 1; }
	@test -f jd.txt || { echo "Missing jd.txt in the current directory" >&2; exit 1; }
	@test -f .env || { echo "Missing .env; copy .env.example and configure your provider/key" >&2; exit 1; }
	@docker run --rm --user "$$(id -u):$$(id -g)" --env-file .env \
		--mount "type=bind,src=$$PWD/resume.pdf,dst=/work/resume.pdf,readonly" \
		--mount "type=bind,src=$$PWD/jd.txt,dst=/work/jd.txt,readonly" \
		resume-cli score /work/resume.pdf --jd /work/jd.txt
