FROM golang:1.26-alpine AS builder

WORKDIR /app

# 先复制依赖文件，利用 Docker 缓存层加速
COPY backend/go.mod backend/go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# 复制后端源码并编译
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /blog-server ./cmd/api/main.go

# —— 第二阶段：运行镜像 ——
FROM alpine:3.21

# 时区 + 基础工具
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从编译阶段复制二进制文件
COPY --from=builder /blog-server .
# 复制前端静态文件
COPY frontend/ ./frontend/

EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=15s --timeout=5s --retries=3 CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./blog-server"]
