# StoreSyncAgent

快递助手（Kdzs）代发同步工具：订单、售后、多账号切换、场景提醒。

## 架构

| 服务 | 端口 | 说明 |
|------|------|------|
| `storesyncagent-api` | 8097 | Go API |
| `storesyncagent-web` | 5178 | Nginx + Vue，反代 `/api/` → API（5177 留给 AfterSales HTTPS） |

## 本地开发

```bash
make init-env
# 编辑 configs/config.yaml（手机号）与 .env（KDZS_PASSWORD）

# 后端
go run ./cmd/api -config configs/config.yaml

# 前端（开发代理）
cd web && npm install && npm run dev
```

本地一体化调试 API 静态资源：`go run ./cmd/api -config configs/config.yaml -web-dist web/dist`

## Docker 部署（本仓库）

```bash
make init-env    # 创建 .env、configs/config.yaml
make up          # 本地 build 并启动
```

- Web：http://localhost:5178
- API：http://localhost:8097/api/v1/health

其他命令：`make down` / `make logs` / `make ps` / `make restart`

### 从 ACR 拉取镜像（生产）

```bash
make init-env-acr   # cp .env.acr.example → .env
# 填写 ACR 凭证、KDZS_PASSWORD、configs/config.yaml

make up-images
```

## CI → 阿里云 ACR

Workflow：`.github/workflows/docker-push-acr.yml`

推送镜像：`storesyncagent-api`、`storesyncagent-web`

Organization Secrets（与 ProductCore 共用）：`ALIYUN_ACR_REGISTRY`、`ALIYUN_ACR_NAMESPACE`、`ALIYUN_ACR_USER`、`ALIYUN_ACR_PASSWORD`

## 敏感配置

`configs/config.yaml`、`.env` 已在 `.gitignore`。

| 变量 | 说明 |
|------|------|
| `KDZS_PASSWORD` | 所有账号共用密码 |
| `KDZS_ACCOUNT_{ID}_PASSWORD` | 按账号 ID，如 `KDZS_ACCOUNT_ACCOUNT_1_PASSWORD` |

## 仓库

https://github.com/OnlineStoreMS/StoreSyncAgent
