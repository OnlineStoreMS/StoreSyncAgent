.PHONY: init-env init-env-acr up down up-images pull-images login-acr logs ps restart

COMPOSE := docker compose -f docker-compose.yml
COMPOSE_ACR := docker compose -f docker-compose.yml -f docker-compose.acr.yml

APP_IMAGES := storesyncagent-api storesyncagent-web

init-env:
	@test -f .env || cp .env.example .env
	@test -f configs/config.yaml || cp configs/config.example.yaml configs/config.yaml
	@echo "已创建 .env 与 configs/config.yaml，请填写快递助手账号与密码"

init-env-acr:
	@test -f .env || cp .env.acr.example .env
	@test -f configs/config.yaml || cp configs/config.example.yaml configs/config.yaml
	@echo "已创建 .env（ACR 模式）与 configs/config.yaml"

login-acr:
	@set -a && . ./.env && set +a && \
	test -n "$$ALIYUN_ACR_REGISTRY" || (echo "请在 .env 配置 ALIYUN_ACR_REGISTRY"; exit 1) && \
	echo "$$ALIYUN_ACR_PASSWORD" | docker login "$$ALIYUN_ACR_REGISTRY" -u "$$ALIYUN_ACR_USER" --password-stdin

pull-images: login-acr
	@set -a && . ./.env && set +a && \
	for img in $(APP_IMAGES); do \
	  echo ">>> pull $${ACR_IMAGE_PREFIX}/$$img:$${IMAGE_TAG:-latest}"; \
	  docker pull "$${ACR_IMAGE_PREFIX}/$$img:$${IMAGE_TAG:-latest}"; \
	done

up: init-env
	$(COMPOSE) up -d --build

up-images: init-env-acr pull-images
	$(COMPOSE_ACR) up -d --no-build --remove-orphans

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

restart:
	$(COMPOSE) restart
