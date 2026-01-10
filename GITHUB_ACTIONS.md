# GitHub Actions CI/CD 配置指南

本文档详细说明如何使用 GitHub Actions 自动构建 Docker 镜像并推送到 GitHub Container Registry (GHCR)。

## 功能特性

- ✅ **自动构建**：推送到 main/master 分支或打标签时自动触发
- ✅ **多架构支持**：同时构建 linux/amd64 和 linux/arm64 镜像
- ✅ **智能标签**：自动生成版本标签（latest、版本号、分支名、commit SHA）
- ✅ **Layer 缓存**：使用 GitHub Actions 缓存加速构建
- ✅ **安全推送**：仅非 PR 的推送事件才会推送到 GHCR

## 工作流文件

位置：`.github/workflows/docker-publish.yml`

### 触发条件

```yaml
on:
  push:
    branches: [ main, master ]    # 推送到主分支时构建
    tags: [ 'v*.*.*' ]            # 推送版本标签时构建
  pull_request:
    branches: [ main, master ]    # PR 时构建（不推送）
```

### 构建平台

```yaml
platforms: linux/amd64,linux/arm64
```

### 自动生成的标签

| 触发事件 | 生成的标签示例 | 说明 |
|---------|--------------|------|
| 推送到 main | `latest`, `main`, `sha-abc1234` | 最新版本 |
| 推送 v1.2.3 | `v1.2.3`, `v1.2`, `v1`, `latest` | 版本标签 |
| Pull Request | `pr-123` | 仅构建，不推送 |

## 首次配置步骤

### 1. 启用 GitHub Actions

1. 进入 GitHub 仓库页面
2. 点击 `Settings` → `Actions` → `General`
3. 滚动到 `Workflow permissions`
4. 选择 **"Read and write permissions"**
5. 点击 `Save`

### 2. 推送代码触发构建

```bash
git add .
git commit -m "chore: 添加 GitHub Actions workflow"
git push origin main
```

### 3. 查看构建状态

1. 进入仓库的 `Actions` 标签页
2. 查看工作流运行状态
3. 构建成功后，镜像会自动推送到 GHCR

### 4. 验证镜像发布

1. 进入 GitHub 仓库页面
2. 点击右侧的 `Packages` 标签（或访问 `https://github.com/用户名/仓库名/pkgs/container/ntp`）
3. 查看已发布的 Docker 镜像

## 使用 GHCR 镜像

### 公开仓库（推荐）

如果你的仓库是**公开**的，任何人都可以拉取镜像：

```bash
docker pull ghcr.io/你的用户名/ntp:latest
```

### 私有仓库

如果仓库是**私有**的，需要先登录：

```bash
# 方法 1：使用 GitHub Personal Access Token
echo $GITHUB_TOKEN | docker login ghcr.io -u 你的用户名 --password-stdin

# 方法 2：使用 GitHub 用户名和 PAT
docker login ghcr.io
# 输入用户名：你的 GitHub 用户名
# 输入密码：Personal Access Token (需要 `read:packages` 权限)
```

### 生成 Personal Access Token（私有仓库需要）

1. GitHub → `Settings` → `Developer settings` → `Personal access tokens` → `Tokens (classic)`
2. 点击 `Generate new token (classic)`
3. 勾选权限：
   - `read:packages`
   - `write:packages` (如果需要推送)
4. 生成并复制 Token

### 运行容器

```bash
docker run -d \
  --name ntp \
  -p 8080:8080 \
  -e PORT=8080 \
  -v ./data:/root/data \
  ghcr.io/你的用户名/ntp:latest
```

### 在 docker-compose.yml 中使用

```yaml
version: '3.8'

services:
  ntp:
    image: ghcr.io/你的用户名/ntp:latest
    container_name: ntp
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
    volumes:
      - ./data:/root/data
    restart: unless-stopped
```

## 发布新版本

### 方法 1：自动发布（推送到主分支）

```bash
git add .
git commit -m "feat: 新功能"
git push origin main
```

这会自动构建并打上 `latest` 标签。

### 方法 2：发布版本（Git 标签）

```bash
# 创建版本标签
git tag v1.0.0
git push origin v1.0.0
```

这会构建并打上多个标签：`v1.0.0`、`v1.0`、`v1`、`latest`。

## 故障排查

### 构建失败：Permission denied

**原因**：Actions 权限未正确配置

**解决**：
1. `Settings` → `Actions` → `General`
2. 选择 `Read and write permissions`
3. 重新运行工作流

### 登录 GHCR 失败：denied

**原因**：Token 权限不足或过期

**解决**：
1. 检查 Token 是否有 `read:packages` 权限
2. 对于私有仓库，确保使用 PAT 而非 GITHUB_TOKEN
3. 重新生成 Token

## 相关资源

- [GitHub Actions 文档](https://docs.github.com/cn/actions)
- [GitHub Packages 文档](https://docs.github.com/cn/packages)
- [Docker Build Push Action](https://github.com/docker/build-push-action)
