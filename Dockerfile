FROM golang:1.22 AS builder

RUN go version

RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates git zlib1g-dev

COPY . /go/src/worker
WORKDIR /go/src/worker

RUN git submodule update --init --recursive --remote

RUN set -Eeux && \
    go mod download && \
    go mod verify

RUN GOOS=linux GOARCH=amd64 \
    go build \
    -tags=jsoniter \
    -trimpath \
    -o main cmd/worker/main.go

FROM ubuntu:latest

RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates curl

COPY --from=builder /go/src/worker/locale /srv/worker/locale
COPY --from=builder /go/src/worker/main /srv/worker/main

RUN chmod +x /srv/worker/main

RUN useradd -m container
USER container
WORKDIR /srv/worker

CMD ["/srv/worker/main"]