<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { HomeFilled, List, OfficeBuilding, Service, Shop } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const activeMenu = computed(() => route.path)

const menuItems = [
  { path: '/dashboard', title: '工作台', icon: HomeFilled },
  { path: '/shops', title: '店铺管理', icon: Shop },
  { path: '/factories', title: '厂家管理', icon: OfficeBuilding },
  { path: '/orders', title: '订单列表', icon: List },
]

const afterSaleItems = [
  { path: '/refunds', title: '售后列表' },
  { path: '/return-exchanges', title: '退换货管理' },
  { path: '/notifications', title: '通知管理' },
]

const logoText = computed(() => (collapsed.value ? 'SSA' : 'StoreSyncAgent'))

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">
      <span class="logo-mark">SS</span>
      <span v-if="!collapsed" class="logo-text">{{ logoText }}</span>
    </div>
    <el-menu
      :default-active="activeMenu"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item
        v-for="item in menuItems"
        :key="item.path"
        :index="item.path"
        @click="navigate(item.path)"
      >
        <el-icon><component :is="item.icon" /></el-icon>
        <span>{{ item.title }}</span>
      </el-menu-item>
      <el-sub-menu index="aftersale">
        <template #title>
          <el-icon><Service /></el-icon>
          <span>售后管理</span>
        </template>
        <el-menu-item
          v-for="item in afterSaleItems"
          :key="item.path"
          :index="item.path"
          @click="navigate(item.path)"
        >
          {{ item.title }}
        </el-menu-item>
      </el-sub-menu>
    </el-menu>
    <div v-if="!collapsed" class="sidebar-footer">快递助手 · 店铺订单同步</div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  background: #001529;
  transition: width 0.2s;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
}
.sidebar.collapsed {
  width: 64px;
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #fff;
  font-weight: 600;
  font-size: 15px;
  border-bottom: 1px solid #ffffff14;
  padding: 0 12px;
}
.logo-mark {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: linear-gradient(135deg, #409eff, #337ecc);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  flex-shrink: 0;
}
.logo-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sidebar :deep(.el-menu) {
  border-right: none;
  flex: 1;
}
.sidebar-footer {
  padding: 12px 16px;
  font-size: 11px;
  color: #ffffff59;
  border-top: 1px solid #ffffff14;
  line-height: 1.4;
}
</style>
