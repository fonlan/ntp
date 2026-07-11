# NTP - 轻量书签导航

NTP 是一个自托管的书签导航服务，适合做浏览器首页或常用站点入口。它支持分组管理、拖拽排序、搜索引擎切换、书签导入导出，以及自动抓取网站标题和图标。

## 特性

- 书签和分组管理
- 拖拽排序
- 搜索引擎切换与自定义
- 浏览器书签导入/导出
- 自动抓取网站标题和图标
- SQLite 存储，部署简单
- 可选登录认证

## 快速开始

### Docker Compose

```bash
docker compose up -d --build
```

启动后访问 `http://localhost:8080`。

### Docker

```bash
docker build -t ntp .

docker run -d \
  --name ntp \
  -p 8080:8080 \
  -e PORT=8080 \
  -v ./data:/root/data \
  ntp
```

### 本地运行

```bash
go mod download
go run .
```

本地编译需要 `CGO_ENABLED=1`，并安装 SQLite 开发库。

## 配置

- `PORT`：服务端口，默认 `8080`
- `AUTH_USERNAME` / `AUTH_PASSWORD`：启用登录认证
- `SESSION_SECRET`：Session 密钥
- `SESSION_TTL`：登录有效期，默认 `365d`（一年）

## 数据持久化

运行数据默认保存在 `data/`，包括数据库文件和图标文件。部署时建议把该目录挂载到持久化卷。

## 许可证

MIT License
