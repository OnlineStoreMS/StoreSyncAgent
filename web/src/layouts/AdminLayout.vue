<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowDown, Expand, Fold, Refresh } from '@element-plus/icons-vue'
import Sidebar from './Sidebar.vue'
import { portalAppsUrl, portalLoginUrl } from '../utils/auth'
import { useSessionStore } from '../stores/session'
import { useKdzsStore } from '../stores/kdzs'
import { formatAccountSubtitle, formatAccountTitle } from '../utils/account'

const route = useRoute()
const collapsed = ref(false)
const sessionStore = useSessionStore()
const kdzsStore = useKdzsStore()

const userInitial = computed(() => {
  const name = sessionStore.session?.user.displayName?.trim()
  return name ? name[0].toUpperCase() : '?'
})

const breadcrumbs = computed(() => {
  const title = (route.meta.title as string) || '电商店铺同步'
  return ['OSMS 店铺同步', title]
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
  void sessionStore.load()
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

function backToPortal() {
  window.location.href = portalAppsUrl()
}

function logout() {
  sessionStore.clear()
  window.location.href = portalLoginUrl()
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
                  请在配置中设置快递助手账号
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button :icon="Refresh" :loading="kdzsStore.loading.status || kdzsStore.loading.switch" @click="refreshStatus">
            刷新连接
          </el-button>
          <el-dropdown trigger="click" @command="(cmd: string) => cmd === 'logout' ? logout() : backToPortal()">
            <div class="user-trigger">
              <el-avatar :size="32" class="user-avatar">{{ userInitial }}</el-avatar>
              <div v-if="sessionStore.session" class="user-meta">
                <span class="user-name">{{ sessionStore.session.user.displayName }}</span>
                <span class="tenant-name">{{ sessionStore.session.tenant.name }}</span>
              </div>
              <span v-else class="user-loading">加载中…</span>
              <el-icon class="user-arrow"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="portal">返回应用中心</el-dropdown-item>
                <el-dropdown-item command="logout" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
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
  max-width: 280px;
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
  min-width: 240px;
  max-width: 320px;
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
.user-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 8px;
}
.user-trigger:hover {
  background: #f5f7fa;
}
.user-avatar {
  background: #409eff;
  color: #fff;
  flex-shrink: 0;
}
.user-meta {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
  max-width: 140px;
}
.user-name {
  font-size: 13px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tenant-name {
  font-size: 11px;
  color: #909399;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-loading {
  font-size: 12px;
  color: #909399;
}
.user-arrow {
  color: #909399;
}
.content {
  flex: 1;
  overflow: auto;
  padding: 16px;
}
</style>
