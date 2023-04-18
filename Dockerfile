
FROM golang:1.19 as builder

ARG GOOS=linux
ARG GOARCH=amd64
ARG BUILD_TYPE=consumer

WORKDIR /go/src/github.com/Klarrio/tw-github-sourcer
COPY . .
RUN make -e GOARCH=${GOARCH} -e GOOS=${GOOS} build.${BUILD_TYPE}

FROM alpine:3.17

ARG GOOS=linux
ARG GOARCH=amd64


RUN apk add --no-cache ca-certificates

COPY --from=builder /go/src/github.com/Klarrio/tw-github-sourcer/tw-github-sourcer-${GOOS}-${GOARCH} /tw-github-sourcer

ENTRYPOINT ["/tw-github-sourcer"]

