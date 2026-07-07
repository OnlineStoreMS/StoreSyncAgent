#!/usr/bin/env bash
set -euo pipefail

APP_PASSWORD="${1:?usage: setup_db.sh APP_PASSWORD}"
DB_NAME="storesyncagent"
DB_USER="storesyncagent"

psql -v ON_ERROR_STOP=1 postgres <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${DB_USER}') THEN
    CREATE ROLE ${DB_USER} LOGIN PASSWORD '${APP_PASSWORD}';
  END IF;
END
\$\$;
SELECT 'CREATE DATABASE ${DB_NAME} OWNER ${DB_USER}'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${DB_NAME}')\gexec
GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};
SQL

echo "database ${DB_NAME} ready for user ${DB_USER}"
echo ""
echo "若表曾由 postgres 超级用户创建，请再执行一次权限修复："
echo "  ./deploy/fix_db_permissions.sh"
