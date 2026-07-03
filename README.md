# StoreSyncAgent

快递助手（Kdzs）代发同步工具：订单、售后、多账号切换、场景提醒。

## 功能

- 多快递助手账号切换
- 订单列表 / 解密 / 推厂家
- 售后场景筛选与 SLA 提醒
- 工作台首页售后提醒

## 本地开发

```bash
# 后端
cp configs/config.example.yaml configs/config.yaml
# 编辑 config.yaml 或通过环境变量注入密码（见下方）

go run ./cmd/api -config configs/config.yaml

# 前端（开发模式，代理 API）
cd web && npm install && npm run dev
```

## 敏感配置

**不要**将真实密码写入 Git。`configs/config.yaml` 已在 `.gitignore` 中。

推荐方式：

1. `config.yaml` 中 `accounts.password` 留空
2. 通过环境变量注入：

| 变量 | 说明 |
|------|------|
| `KDZS_PASSWORD` | 所有账号共用密码 |
| `KDZS_ACCOUNT_{ID}_PASSWORD` | 按账号 ID 单独配置，如 `KDZS_ACCOUNT_ACCOUNT_1_PASSWORD` |

## Docker 本地部署

```bash
cp configs/config.example.yaml configs/config.yaml
cp .env.example .env
# 编辑 configs/config.yaml（账号手机号等）和 .env（密码）

docker compose up -d --build
```

访问 http://localhost:8097

## 镜像

GitHub Actions 推送至：

`ghcr.io/onlinestorems/storesyncagent:latest`

拉取运行：

```bash
docker pull ghcr.io/onlinestorems/storesyncagent:latest
docker run -d --name storesyncagent \
  -p 8097:8097 \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  -e KDZS_PASSWORD=your_password \
  ghcr.io/onlinestorems/storesyncagent:latest
```

## 仓库

https://github.com/OnlineStoreMS/StoreSyncAgent
