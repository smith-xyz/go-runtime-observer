# GO_VERSION is the version of Go to INSTRUMENT (can be older)
ARG GO_VERSION=1.23.0
# BUILDER_GO_VERSION is the version to BUILD the instrumenter (must be >= 1.24.0)
ARG BUILDER_GO_VERSION=1.24.0

FROM golang:${BUILDER_GO_VERSION} AS instrumenter

ARG GO_VERSION

RUN apt-get update && apt-get install -y \
    git \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /instrumenter
COPY instrumenter/ ./instrumenter/
COPY go.mod go.sum ./

RUN go build -o /bin/instrumenter ./instrumenter

RUN wget https://go.dev/dl/go${GO_VERSION}.src.tar.gz && \
    tar -xzf go${GO_VERSION}.src.tar.gz && \
    mv go /go-source

COPY runtime/instrumentlog /go-source/src/runtime/instrumentlog

RUN /bin/instrumenter \
    -src /go-source \
    -output /instrumented && \
    cp /instrumented/reflect/value.go /go-source/src/reflect/value.go

FROM golang:${BUILDER_GO_VERSION} AS builder

COPY --from=instrumenter /go-source /go-source

WORKDIR /go-source/src
ENV GOROOT_BOOTSTRAP=/usr/local/go
RUN ./make.bash

FROM debian:bullseye-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

ARG GO_VERSION
LABEL org.opencontainers.image.version="${GO_VERSION}"
LABEL org.opencontainers.image.description="Instrumented Go ${GO_VERSION} for runtime observation"

COPY --from=builder /go-source /usr/local/go

ENV GOROOT=/usr/local/go
ENV PATH=/usr/local/go/bin:$PATH
ENV GOTOOLCHAIN=local
ENV INSTRUMENTATION_LOG_PATH=/tmp/instrumentation.log

WORKDIR /work

ENTRYPOINT ["go"]
CMD ["version"]