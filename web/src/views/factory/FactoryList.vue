<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useKdzsStore } from '../../stores/kdzs'

const kdzsStore = useKdzsStore()
const platform = ref('FXG')

onMounted(() => {
  void kdzsStore.loadFactories(platform.value)
})

function syncFactories() {
  void kdzsStore.loadFactories(platform.value)
}
</script>

<template>
  <div class="factory-page">
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">合作厂家 <span class="count">({{ kdzsStore.factories.length }})</span></div>
          <div class="actions">
            <el-select v-model="platform" style="width: 120px" @change="syncFactories">
              <el-option label="抖店" value="FXG" />
              <el-option label="淘宝" value="TB" />
            </el-select>
            <el-button type="primary" :loading="kdzsStore.loading.factories" @click="syncFactories">同步厂家</el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="厂家列表来自快递助手「合作厂家」，同步后可用于订单推送给厂家。"
        class="hint"
      />

      <el-table :data="kdzsStore.factories" v-loading="kdzsStore.loading.factories" stripe border empty-text="暂无厂家，请先在快递助手绑定">
        <el-table-column prop="factoryName" label="厂家账号" min-width="140" />
        <el-table-column prop="remark" label="备注名" min-width="140" />
        <el-table-column prop="factoryId" label="厂家ID" min-width="120" />
        <el-table-column prop="bindTime" label="绑定时间" width="170" />
        <el-table-column label="绑定状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.bindStatus === 1 ? 'success' : 'info'" size="small">
              {{ row.bindStatus === 1 ? '已绑定' : '未绑定' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="能力" min-width="160">
          <template #default="{ row }">
            <el-tag v-if="row.supportBindItem" size="small" type="warning" effect="plain">支持商品绑定</el-tag>
            <el-tag v-if="row.hasPrePushTrade" size="small" type="danger" effect="plain">有待推单</el-tag>
            <span v-if="!row.supportBindItem && !row.hasPrePushTrade" class="muted">-</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.actions {
  display: flex;
  gap: 8px;
  align-items: center;
}
.hint {
  margin-bottom: 16px;
}
</style>
