# love-diary-go

恋爱日记 Go API 服务，与 `love-diary` 前端接口对齐。

**GitHub Actions → GHCR → 服务器 Docker Compose + Watchtower**，与前端独立部署、互不影响。

## 一、首次发布镜像（GitHub Actions）

1. 将代码推送到 `master` 分支（`love-diary-go/` 有变更时会触发构建）
2. 打开仓库 **Actions** → **Build and Push love-diary-go**，确认成功
3. 镜像地址：`ghcr.io/yu-hong-quan/love-diary-go:latest`
   （与 `docker-compose.yml` 中一致；若仓库 owner 不同请改 compose 里的镜像名）

> 首次使用 GHCR：仓库 **Settings → Actions → General → Workflow permissions** 选 **Read and write permissions**。

## 二、服务器部署（宝塔面板）

在服务器上准备目录 `/home/www/love-diary-go/`，只需两个文件：

```
/home/www/love-diary-go/
├── docker-compose.yml    # 从本仓库复制
├── deploy.sh             # 一键部署脚本
├── uploads/              # 图片目录（部署时自动创建）
└── .env                  # 从 .env.docker.example 复制并填写
```

### 1. 配置 `.env`

```bash
cp .env.docker.example .env
vim .env
```

```env
PORT=3000
GIN_MODE=release
UPLOAD_DIR=/app/uploads
DB_HOST=你的数据库IP
DB_PORT=5432
DB_USER=love-diary-sql
DB_PASSWORD=你的密码
DB_NAME=love_diary
JWT_SECRET=随机长字符串
```

### 2. 登录 GHCR 并启动

首次拉取私有镜像需登录（GitHub → Settings → Developer settings → PAT，勾选 `read:packages`）：

```bash
chmod +x deploy.sh
GHCR_TOKEN=<你的PAT> ./deploy.sh login
./deploy.sh
```

或手动：

```bash
echo <GITHUB_PAT> | docker login ghcr.io -u yu-hong-quan --password-stdin
docker compose pull
docker compose up -d
```

### 3. 验证

```bash
curl http://127.0.0.1:3000/health
docker compose logs -f love-diary-go
```

Watchtower 每 5 分钟检查 `ghcr.io/yu-hong-quan/love-diary-go:latest`，有新版本会自动拉取并重启容器。

### 4. 宝塔 Nginx 反代（配置域名 + SSL）

宝塔面板 → 网站 → 添加站点 `api.你的域名.com`，站点配置：

```nginx
server {
    listen 80;
    server_name api.你的域名.com;

    client_max_body_size 20m;  # 上传/资料接口；头像已改 multipart，若仍用 base64 需留足余量

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

宝塔面板 → 网站 → api.你的域名.com → SSL → 一键申请 Let's Encrypt。

## 三、本地调试

```bash
# 方式 A：直接 Go 运行
go run .

# 方式 B：本地 Docker 构建
cp .env.docker.example .env
docker compose -f docker-compose.build.yml up -d --build
```

## 四、数据库

部署前在 PostgreSQL 执行：

- `../sql/users.sql`
- 数据迁移脚本 `love-diary-migrate.sql`
- `sql/romantic_toasts.sql`（首页浪漫 toast 文案池，可重复执行）

## 五、整体架构

```
浏览器
  ├─ https://你的域名.com    → 宝塔Nginx → love-diary 容器 :8080 (Vue SPA)
  └─ https://api.你的域名.com → 宝塔Nginx → love-diary-go 容器 :3000 (Go API)
                                               ↓
                                         PostgreSQL (远程)
```

两个项目各自独立部署：分别放在 `/home/www/love-diary` 和 `/home/www/love-diary-go`，各自有独立的 `docker-compose.yml` 和 Watchtower。

## API

| 方法 | 路径 |
|------|------|
| GET | `/health` |
| GET | `/romantic-toasts/random` |
| POST | `/auth/login` |
| POST | `/upload/image`, `/upload/images` |
| CRUD | `/travel-diaries`, `/daily-diaries`, `/whispers`, `/special-dates` |
| GET | `/tree`, `/tree/logs` |
| POST | `/tree/water` |
