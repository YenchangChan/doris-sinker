# 构建阶段
FROM golang:1.21-alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /build

# 复制go mod文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o doris-sinker \
    cmd/doris-sinker/main.go

# 运行阶段
FROM alpine:latest

# 安装ca证书
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非root用户
RUN addgroup -g 1000 sinker && \
    adduser -D -u 1000 -G sinker sinker

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/doris-sinker .
COPY --from=builder /build/configs/config.yaml ./configs/

# 修改权限
RUN chown -R sinker:sinker /app

# 切换到非root用户
USER sinker

# 暴露端口
EXPOSE 9090 6060

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

# 启动命令
ENTRYPOINT ["./doris-sinker"]
CMD ["-config", "configs/config.yaml"]
