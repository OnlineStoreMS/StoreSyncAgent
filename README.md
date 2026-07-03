# StoreSyncAgent

快递助手（Kdzs）代发同步工具：订单、售后、多账号切换、场景提醒。

## 架构

| 服务 | 端口 | 说明 |
|------|------|------|
| `storesyncagent-api` | 8097 | Go API |
| `storesyncagent-web` | 5177 | Nginx + Vue，反代 `/api/` → API |

## 本地开发

```bash
cp configs/config.example.yaml configs/config.yaml
export KDZS_PASSWORD=你的密码   # 或写入 config.yaml（勿提交）

# 后端
go run ./cmd/api -config configs/config.yaml

# 前端（开发代理）
cd web && npm install && npm run dev
```

本地一体化调试 API 静态资源：`go run ./cmd/api -config configs/config.yaml -web-dist web/dist`

## Docker（仓库内）

```bash
cp configs/config.example.yaml configs/config.yaml
cp .env.example .env

docker compose up -d --build
```

访问 http://localhost:5177

## 生产部署（推荐）

使用 **deploy** 仓库独立目录：

```bash
cd ~/projects/deploy/storesyncagent
make init-env-acr
make up-images
```

见 [deploy/storesyncagent/README.md](../deploy/storesyncagent/README.md)（若与 deploy 同级）。

## CI → 阿里云 ACR

Workflow：`.github/workflows/docker-push-acr.yml`

推送镜像：

- `storesyncagent-api`
- `storesyncagent-web`

Organization Secrets（与 ProductCore 共用）：`ALIYUN_ACR_REGISTRY`、`ALIYUN_ACR_NAMESPACE`、`ALIYUN_ACR_USER`、`ALIYUN_ACR_PASSWORD`

## 敏感配置

`configs/config.yaml`、`.env` 已在 `.gitignore`。

| 变量 | 说明 |
|------|------|
| `KDZS_PASSWORD` | 所有账号共用密码 |
| `KDZS_ACCOUNT_{ID}_PASSWORD` | 按账号 ID，如 `KDZS_ACCOUNT_ACCOUNT_1_PASSWORD` |

## 仓库

https://github.com/OnlineStoreMS/StoreSyncAgent
