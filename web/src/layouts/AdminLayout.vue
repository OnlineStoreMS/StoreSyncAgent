<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowDown, Expand, Fold, Refresh } from '@element-plus/icons-vue'
import Sidebar from './Sidebar.vue'
import { useKdzsStore } from '../stores/kdzs'
import { formatAccountSubtitle, formatAccountTitle } from '../utils/account'

const route = useRoute()
const collapsed = ref(false)
const kdzsStore = useKdzsStore()

const breadcrumbs = computed(() => {
  const title = (route.meta.title as string) || 'StoreSyncAgent'
  if (route.path.startsWith('/shops')) return ['同步中心', '店铺管理']
  if (route.path.startsWith('/factories')) return ['同步中心', '厂家管理']
  if (route.path.startsWith('/orders')) return ['同步中心', '订单列表']
  if (route.path.startsWith('/return-exchanges')) return ['售后管理', '退换货管理']
  if (route.path.startsWith('/notifications')) return ['售后管理', '通知管理']
  if (route.path.startsWith('/refunds')) return ['售后管理', '售后列表']
  return ['首页', title]
})

const activeAccountLabel = computed(() => {
  const info = kdzsStore.loginInfo
  if (!info.loggedIn) return '未连接'
  const active = kdzsStore.accounts.find((acc) => acc.active)
  if (active) return formatAccountTitle(active)
  return formatAccountTitle({
    name: info.accountName,
    mobile: info.mobile,
    roleLabel: info.accountRoleLabel || '商家版',
  })
})

onMounted(async () => {
  await kdzsStore.loadStatus()
  await kdzsStore.loadAccounts()
})

async function refreshStatus() {
  await kdzsStore.loadStatus()
  await kdzsStore.loadAccounts()
}

async function onSwitchAccount(accountId: string) {
  if (accountId === kdzsStore.loginInfo.accountId) return
  await kdzsStore.switchKdzsAccount(accountId)
}
</script>

<template>
  <div class="admin-layout">
    <Sidebar v-model:collapsed="collapsed" />
    <div class="main-area">
      <header class="header">
        <div class="header-left">
          <el-button :icon="collapsed ? Expand : Fold" text @click="collapsed = !collapsed" />
          <el-breadcrumb separator="/">
            <el-breadcrumb-item v-for="(item, i) in breadcrumbs" :key="i">{{ item }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-dropdown trigger="click" @command="onSwitchAccount">
            <div class="account-trigger">
              <el-tag v-if="kdzsStore.loginInfo.loggedIn" type="success" effect="plain">
                快递助手
              </el-tag>
              <el-tag v-else type="info" effect="plain">未连接</el-tag>
              <span class="account-text">{{ activeAccountLabel }}</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="acc in kdzsStore.accounts"
                  :key="acc.id"
                  :command="acc.id"
                >
                  <div class="account-option">
                    <span class="account-option-label" :title="formatAccountTitle(acc)">
                      {{ formatAccountTitle(acc) }}
                    </span>
                    <span v-if="formatAccountSubtitle(acc)" class="dropdown-mobile">
                      {{ formatAccountSubtitle(acc) }}
                    </span>
                    <el-tag v-if="acc.active" class="account-option-tag" size="small" type="success" effect="plain">
                      当前
                    </el-tag>
                  </div>
                </el-dropdown-item>
                <el-dropdown-item v-if="!kdzsStore.accounts.length" disabled>
                  请在 configs/config.yaml 配置 accounts
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button :icon="Refresh" :loading="kdzsStore.loading.status || kdzsStore.loading.switch" @click="refreshStatus">
            刷新连接
          </el-button>
        </div>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  height: 100vh;
  background: #f0f2f5;
}
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.header {
  height: 56px;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.account-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 8px;
  white-space: nowrap;
  max-width: 360px;
}
.account-trigger :deep(.el-tag) {
  flex-shrink: 0;
}
.account-trigger:hover {
  background: #f5f7fa;
}
.account-text {
  font-size: 13px;
  color: #606266;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dropdown-mobile {
  flex-shrink: 0;
  color: #909399;
  font-size: 12px;
}
.account-option {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 280px;
  max-width: 360px;
  white-space: nowrap;
}
.account-option-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
.account-option-tag {
  flex-shrink: 0;
}
.content {
  flex: 1;
  overflow: auto;
  padding: 16px;
}
</style>
