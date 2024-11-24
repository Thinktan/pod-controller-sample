FROM golang:1.22 AS builder
WORKDIR /app

# 复制项目文件
COPY go.mod go.sum ./
RUN go mod download

COPY pod-controller.go ./

# 构建可执行文件
RUN go build -o pod-controller pod-controller.go

FROM ubuntu:latest
WORKDIR /root/
COPY --from=builder /app/pod-controller .
ENTRYPOINT ["/root/pod-controller"]
