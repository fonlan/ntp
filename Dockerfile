# 多阶段构建
FROM golang:1.25.5-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=1 GOOS=linux go build -o ntp .

# 最终镜像
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

# 从构建阶段复制二进制文件和资源
COPY --from=builder /app/ntp .
COPY --from=builder /app/static ./static
COPY --from=builder /app/i18n ./i18n

# 创建数据目录
RUN mkdir -p data

EXPOSE 8080

CMD ["./ntp"]
