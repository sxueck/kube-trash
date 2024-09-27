FROM m.daocloud.io/docker.io/golang:1.23.1-alpine3.20 AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

COPY . .

RUN GO111MODULE=on GOPROXY="https://goproxy.cn" CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /go/bin/trash .

FROM m.daocloud.io/docker.io/alpine:3.20

RUN echo "https://mirrors.aliyun.com/alpine/v3.20/main" > /etc/apk/repositories && \
    echo "https://mirrors.aliyun.com/alpine/v3.20/community" >> /etc/apk/repositories

RUN apk add --no-cache tzdata ca-certificates && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    rm -rf /var/cache/apk/*

COPY --from=builder /go/bin/trash /trash

CMD ["/trash"]