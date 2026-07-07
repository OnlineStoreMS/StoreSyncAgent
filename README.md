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

## 配置与部署

业务数据（KDZS 账号、退换货、通知配置）均存 **PostgreSQL**，通过 UserCore 应用中心 SSO 登录。

平台统一部署见 **`deploy` 仓库**：

```bash
make sync-configs && make up-images
```

`configs/config.yaml` 与 `deploy/configs/storesyncagent.yaml` 结构一致；KDZS 账号在 Web「账号管理」维护，不在配置文件中填写。

## 数据库

```bash
make init-db APP_PASSWORD=你的密码
make fix-db-perms
```

## Docker / ACR

镜像名：`storesyncagent-api`、`storesyncagent-web`，CI 推送到阿里云 ACR（见 `.github/workflows/docker-push-acr.yml`）。

## 仓库

https://github.com/OnlineStoreMS/StoreSyncAgent
