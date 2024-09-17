FROM golang:1.22-bullseye AS stage1
# Build a base image that pre-caches the extra tools we need. In particular,
# sqlite3 can take a minute or two we'd like to avoid if possible.
#

ENV CGO_ENABLED=1
ENV GOCACHE=/go/.cache

RUN apt-get update && apt-get install -y libwebp-dev gcc

WORKDIR /go/src
COPY go.* ./

RUN go mod download && go mod verify
RUN go install github.com/mattn/go-sqlite3
RUN chmod -R a+rw /go/.cache

FROM stage1 AS stage2
WORKDIR /go/src
COPY *.yaml *.go build/Makefile ./
COPY cmd ./cmd/
COPY internal ./internal/
RUN make build

FROM debian:bullseye
WORKDIR /app
RUN apt-get update \
    && apt-get install -y ca-certificates \
                          libwebp6 \
                          libwebpdemux2 \
                          libwebpmux3

COPY web/static ./static
COPY apps/community/apps ./apps
COPY apps/local/ ./apps
COPY --from=stage2 /go/src/bin/pixelgw ./bin/pixelgw
CMD ["/app/bin/pixelgw"]

