FROM golang:1.25.5-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY samples.go ./
COPY testdata ./testdata
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /resume-cli ./cmd/resume-cli

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends poppler-utils poppler-data ca-certificates fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /resume-cli /usr/local/bin/resume-cli
COPY testdata /examples
RUN mkdir /work && chmod 1777 /work
WORKDIR /work
USER 65532:65532
ENTRYPOINT ["resume-cli"]
CMD ["--help"]
