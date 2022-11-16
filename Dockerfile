FROM debian

ENV DEBIAN_FRONTEND noninteractive

WORKDIR /go/src/github.com/allfro/device-volume-driver
COPY . .
RUN apt update && \
    apt install -y musl-dev musl-tools git curl && \
    curl -L -o go.tgz https://go.dev/dl/go1.19.3.linux-amd64.tar.gz && \
    tar -zxvf go.tgz && \
    export PATH=$PATH:go/bin && \
    go get && \
    CC=/usr/bin/musl-gcc go build -ldflags "-linkmode external -extldflags -static" -o /dvd

FROM alpine
COPY --from=0 /dvd /dvd
ENTRYPOINT ["/dvd"]
