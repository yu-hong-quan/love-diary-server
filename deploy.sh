#!/usr/bin/env bash
# 服务器部署脚本：拉取 GHCR 镜像并启动 love-diary-go API
# 用法：在 /home/www/love-diary-go 目录执行 ./deploy.sh
# 可选：GHCR_TOKEN=xxx ./deploy.sh login   # 首次登录 GHCR
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

GHCR_USER="${GHCR_USER:-yu-hong-quan}"

cmd_login() {
  if [[ -z "${GHCR_TOKEN:-}" ]]; then
    echo "用法: GHCR_TOKEN=<GitHub_PAT> ./deploy.sh login"
    exit 1
  fi
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USER" --password-stdin
  echo "==> GHCR 登录成功"
}

cmd_deploy() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "错误: 未安装 docker"
    exit 1
  fi

  if ! docker compose version >/dev/null 2>&1; then
    echo "错误: 未安装 docker compose 插件"
    exit 1
  fi

  if [[ ! -f docker-compose.yml ]]; then
    echo "错误: 当前目录缺少 docker-compose.yml"
    exit 1
  fi

  if [[ ! -f .env ]]; then
    echo "错误: 缺少 .env，请先: cp .env.docker.example .env && vim .env"
    exit 1
  fi

  mkdir -p uploads

  echo "==> [love-diary-go] 拉取镜像..."
  docker compose pull

  echo "==> [love-diary-go] 启动容器..."
  docker compose up -d

  echo "==> [love-diary-go] 等待健康检查..."
  sleep 3

  if curl -sf "http://127.0.0.1:3000/health" >/dev/null; then
    echo "==> [love-diary-go] /health 正常"
  else
    echo "==> [love-diary-go] 警告: /health 未通过，请执行: docker compose logs -f love-diary-go"
  fi

  echo "==> [love-diary-go] 容器状态:"
  docker compose ps
  echo "==> 完成"
}

case "${1:-deploy}" in
  login)  cmd_login ;;
  deploy|"") cmd_deploy ;;
  *)
    echo "用法: ./deploy.sh [deploy|login]"
    exit 1
    ;;
esac
