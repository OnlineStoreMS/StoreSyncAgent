<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAccountRefresh } from '../../composables/useAccountRefresh'
import { useKdzsStore } from '../../stores/kdzs'

const kdzsStore = useKdzsStore()

const platformSummary = computed(() => {
  const map = new Map<string, number>()
  for (const shop of kdzsStore.shops) {
    map.set(shop.platformName, (map.get(shop.platformName) || 0) + 1)
  }
  return [...map.entries()].map(([name, count]) => ({ name, count }))
})

useAccountRefresh(() => kdzsStore.loadShops())

onMounted(() => {
  void kdzsStore.loadShops()
})
</script>

<template>
  <div class="shop-page">
    <el-row :gutter="16" class="summary-row" v-if="platformSummary.length">
      <el-col v-for="item in platformSummary" :key="item.name" :xs="12" :sm="8" :lg="4">
        <el-card shadow="hover" class="mini-stat">
          <div class="mini-stat-label">{{ item.name }}</div>
          <div class="mini-stat-value">{{ item.count }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">店铺列表 <span class="count">({{ kdzsStore.shops.length }})</span></div>
          <el-button type="primary" :loading="kdzsStore.loading.shops" @click="kdzsStore.loadShops()">刷新</el-button>
        </div>
      </template>
      <el-table :data="kdzsStore.shops" v-loading="kdzsStore.loading.shops" stripe border empty-text="暂无店铺">
        <el-table-column prop="platformName" label="平台" width="100" />
        <el-table-column prop="mallUserName" label="店铺名称" min-width="180" />
        <el-table-column prop="mallUserId" label="店铺ID" min-width="140" />
        <el-table-column prop="bindTime" label="绑定时间" width="170" />
        <el-table-column prop="expireTime" label="授权到期" width="170" />
        <el-table-column label="授权状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.tokenValid ? 'success' : 'warning'" size="small">
              {{ row.tokenValid ? '有效' : '待续期' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.summary-row {
  margin-bottom: 16px;
}
.mini-stat {
  margin-bottom: 16px;
  text-align: center;
}
.mini-stat-label {
  font-size: 13px;
  color: #909399;
}
.mini-stat-value {
  font-size: 28px;
  font-weight: 600;
  margin-top: 4px;
}
</style>
