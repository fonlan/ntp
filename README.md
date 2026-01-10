# NTP - 轻量书签导航

NTP 是一个轻量的书签导航（Start Page）服务：支持书签分组管理、拖拽排序、搜索引擎切换、书签导入/导出，以及自动获取站点标题/图标。

## 特性

- 🔍 **多搜索引擎切换**：预置常见搜索引擎，支持自定义添加（URL 使用 `{q}` 作为关键词占位符）
- 📁 **分组/书签管理**：支持创建分组、批量排序、拖拽排序
- 🤖 **自动获取网站信息**：添加书签时自动抓取标题与图标；支持上传自定义图标
- 📥 **导入/导出**：兼容浏览器书签 HTML（Netscape Bookmark File）
- 💾 **SQLite 存储**：单文件数据库，无需额外数据库服务
- 🔐 **登录认证**：可选的用户认证功能，保护书签数据安全

## 技术栈

- **后端**：Go 1.19 + `net/http`
- **数据库**：SQLite（`mattn/go-sqlite3`，需要 CGO）
- **前端**：原生 HTML/CSS/JavaScript
- **部署**：Docker + Docker Compose

## 目录结构

```text
.
├─ main.go                 # HTTP 路由与启动入口
├─ handlers/               # 请求处理（Controller）
├─ models/                 # 数据模型与 Repository（SQLite）
│  ├─ migrations/          # 数据库迁移（编译期 embed）
│  └─ schema.sql           # 初始 schema（与 V001 迁移内容一致，主要作参考）
├─ services/               # 业务服务（图标下载/缓存等）
├─ middleware/             # 中间件（Locale/i18n 等）
├─ static/                 # 前端静态资源
└─ data/                   # 运行时数据（建议持久化挂载）
```

## 快速开始

### Docker Compose（推荐）

```bash
docker compose up -d --build
```

访问 `http://localhost:8080`。

默认使用卷挂载将 `./data` 映射到容器内 `/root/data`，用于持久化数据库与图标文件。

### Docker（不使用 Compose）

```bash
docker build -t ntp .

docker run -d \
  --name ntp \
  -p 8080:8080 \
  -e PORT=8080 \
  -v ./data:/root/data \
  ntp
```

### 使用 GHCR 镜像（推荐）

本项目通过 GitHub Actions 自动构建并发布 Docker 镜像到 GitHub Container Registry (GHCR)。

#### 自动构建触发条件

- 推送到 `main` 或 `master` 分支
- 推送版本标签（如 `v1.0.0`）
- 创建 Pull Request（仅构建，不推送）

#### 拉取并运行镜像

```bash
# 拉取最新镜像（公开仓库无需登录）
docker pull ghcr.io/你的用户名/ntp:latest

# 运行容器
docker run -d \
  --name ntp \
  -p 8080:8080 \
  -e PORT=8080 \
  -v ./data:/root/data \
  ghcr.io/你的用户名/ntp:latest
```

#### 可用镜像标签

- `latest`：最新稳定版（main/master 分支）
- `v1.0.0`、`v1.0`、`v1`：版本标签（基于 Git 标签）
- `main`、`master`：分支名
- `sha-abc1234`：基于提交 SHA

**注意**：首次使用前，需要在 GitHub 仓库设置中启用 "Actions" 权限：`Settings` → `Actions` → `General` → `Workflow permissions` → 选择 "Read and write permissions"。

### 本地运行/开发

本项目依赖 `go-sqlite3`（CGO），需要本机具备 C 编译环境与 SQLite 开发库：

- Debian/Ubuntu：`sudo apt-get install -y build-essential libsqlite3-dev`
- Alpine：`apk add --no-cache gcc musl-dev sqlite-dev`
- macOS（Homebrew）：`brew install sqlite`

运行：

```bash
go mod download
go run .
```

编译并运行（确保 `CGO_ENABLED=1`）：

```bash
CGO_ENABLED=1 go build -o ntp .
./ntp
```

## 配置

- `PORT`：服务端口（默认：`8080`）
- `AUTH_USERNAME`：登录用户名（可选，与 `AUTH_PASSWORD` 同时设置时启用认证）
- `AUTH_PASSWORD`：登录密码（可选，与 `AUTH_USERNAME` 同时设置时启用认证）
- `SESSION_SECRET`：Session 加密密钥（可选，不设置时自动生成）

**认证配置说明**：
- 如果设置了 `AUTH_USERNAME` 和 `AUTH_PASSWORD`，访问首页和 API 需要先登录
- 如果不设置这两个环境变量，认证将被禁用（公开访问，无需登录）
- 建议在生产环境中设置强密码并使用 `SESSION_SECRET` 增强 Session 安全性

**Docker Compose 配置示例**：

```yaml
services:
  ntp:
    image: ntp
    ports:
      - "8080:8080"
    volumes:
      - ./data:/root/data
    environment:
      - PORT=8080
      - AUTH_USERNAME=admin
      - AUTH_PASSWORD=your_secure_password_here
      - SESSION_SECRET=your_random_secret_string_here
```

## 数据与持久化

- 数据目录：启动时自动创建 `data/` 与 `data/icons/`
- 数据库文件：`data/bookmarks.db`
  - WAL 模式下会出现 `data/bookmarks.db-wal` / `data/bookmarks.db-shm`，同样需要持久化
- 图标文件：`data/icons/*`，通过 `/data/icons/` 访问

## 数据库迁移

程序启动时会自动执行未执行的迁移。迁移文件位于 `models/migrations/*.sql`，并通过 Go `embed` 编译进二进制。

- 命名格式：`V{版本号}__{描述}.sql`（从 `001` 递增）
- 迁移记录：`schema_migrations` 表

添加新迁移：

1. 在 `models/migrations/` 新建 `V002__xxx.sql`
2. 写入 SQL
3. 重新构建并部署，启动时自动应用

详见：`models/migrations/MIGRATIONS.md`

## 国际化（i18n）

- 语言包：`i18n/en.json`、`i18n/zh.json`
- 前端请求会携带 `X-Locale: en|zh`，后端也会根据 `Accept-Language` 回退判断

## HTTP API（可选）

项目提供一组 JSON API 供二次集成（默认启用 CORS）。

**注意**：如果启用了认证（设置了 `AUTH_USERNAME` 和 `AUTH_PASSWORD`），所有 API 接口（除了登录/登出接口）都需要先登录才能访问。

### Authentication

- `POST /api/login`：登录（`{username, password}`，返回 session cookie）
- `POST /api/logout`：登出（清除 session cookie）
- `GET /api/auth/check`：检查登录状态（返回 `{authenticated, enabled}`）

### Bookmarks

- `GET /api/bookmarks`：获取全部书签（可选：`?group_id=ID`）
- `POST /api/bookmarks`：创建书签
- `PUT /api/bookmark/{id}`：更新书签
- `DELETE /api/bookmark/{id}`：删除书签
- `POST /api/bookmarks/reorder`：批量排序（`[{id, sort_order}]`）
- `GET /api/bookmarks/search?q=keyword`：搜索
- `POST /api/bookmarks/import`：导入（multipart，字段名 `file`）
- `GET /api/bookmarks/export`：导出（返回 HTML 文件）

### Groups

- `GET /api/groups`：获取分组
- `POST /api/groups`：创建分组
- `PUT /api/groups/{id}`：更新分组
- `DELETE /api/groups/{id}`：删除分组
- `POST /api/groups/reorder`：批量排序（`[{id, sort_order}]`）

### Search Engines

- `GET /api/search-engines`：获取搜索引擎
- `POST /api/search-engines`：创建搜索引擎
- `PUT /api/search-engine/{id}`：更新搜索引擎
- `DELETE /api/search-engine/{id}`：删除搜索引擎
- `POST /api/search-engines/reorder`：批量排序（`[{id, sort_order}]`）

### Metadata / Icons

- `POST /api/fetch-metadata`：抓取标题/图标（`{url}`）
- `POST /api/upload-icon`：上传图标（multipart，字段名 `icon`，最大 512KB；返回 `icon_path`）

## 许可证

MIT License
