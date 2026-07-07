# StoreSyncAgent — OSMS 电商店铺同步

StoreSyncAgent 是 **OSMS** 平台下的电商店铺同步应用，与 StoreCore、SupplyCore 等模块采用相同技术栈与部署方式。

## 功能模块

| 模块 | 说明 |
|------|------|
| 店铺 / 订单 | 快递助手（Kdzs）店铺绑定、代发订单拉取与解密 |
| 售后 | 售后列表、SLA 提醒、退换货管理 |
| 通知 | 飞书 Webhook 场景推送（含物流条形码） |
| 账号管理 | 多租户 KDZS 账号（PostgreSQL 存储） |

## 端口

| 服务 | 端口 |
|------|------|
| API | 8097 |
| Web | 5178 |

## 本地开发

```bash
cp configs/config.example.yaml configs/config.yaml
# 编辑 configs/config.yaml（数据库、auth 等）

make run

# 前端
cd web && npm install && npm run dev
```

本地一体化调试 API 静态资源：`go run ./cmd/api -config configs/config.yaml -web-dist web/dist`

登录：从 UserCore 应用中心（`:5174`）进入「电商店铺同步」；本地可将 `auth.enabled` 设为 `false` 跳过 SSO。

## 数据库

```bash
make init-db APP_PASSWORD=你的密码
make fix-db-perms
```

平台统一部署使用 `deploy` 仓库中的 PostgreSQL 配置；本地开发默认 SQLite（见 `configs/config.example.yaml`）。

## Docker / ACR

镜像名：`storesyncagent-api`、`storesyncagent-web`，CI 推送到阿里云 ACR（见 `.github/workflows/docker-push-acr.yml`）。

**平台编排见 `/home/asialeaf/projects/deploy`**：`make sync-configs && make up-images`

## 仓库

https://github.com/OnlineStoreMS/StoreSyncAgent
