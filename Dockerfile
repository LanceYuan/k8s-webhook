# syntax = docker/dockerfile:experimental
FROM golang:1.16.13 AS build
WORKDIR /go/src/app
COPY . .
ENV GOPROXY=https://goproxy.io,direct
ENV GOPATH=/go
RUN --mount=type=cache,id=golang,target=/go/pkg/mod go install .

FROM ubuntu:20.04
WORKDIR /opt/
RUN apt-get update && apt-get install tzdata
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY --from=build /go/bin/k8s-webhook .
CMD ["/opt/k8s-webhook"]