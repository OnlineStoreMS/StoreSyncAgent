<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useAccountRefresh } from '../../composables/useAccountRefresh'
import {
  getProductSyncProgress,
  listProducts,
  syncProducts,
  type Product,
  type ProductSyncProgress,
} from '../../api'
import { useKdzsStore } from '../../stores/kdzs'

const kdzsStore = useKdzsStore()

const loading = reactive({ products: false, sync: false })
const products = ref<Product[]>([])
const total = ref(0)
const syncProgress = ref<ProductSyncProgress | null>(null)
let progressTimer: ReturnType<typeof setInterval> | null = null

const filters = reactive({
  platform: 'FXG',
  shopId: '',
  title: '',
  pageNo: 1,
  pageSize: 20,
})

const platformOptions = [
  { label: '抖店', value: 'FXG' },
  { label: '淘宝', value: 'TB' },
  { label: '小红书', value: 'XHS' },
]

const shopOptions = computed(() =>
  kdzsStore.shops.filter((s) => !filters.platform || s.platform === filters.platform),
)

const syncing = computed(() => syncProgress.value != null && !syncProgress.value.finish)

const progressText = computed(() => {
  const p = syncProgress.value
  if (!p) return ''
  if (p.errorMessage) return p.errorMessage
  const counts = p.syncItemCount
  if (counts && Object.keys(counts).length) {
    const parts = Object.entries(counts).map(([shopId, count]) => {
      const shop = kdzsStore.shops.find((s) => s.mallUserId === shopId)
      const name = shop?.mallUserName || shopId
      return `${name}: ${count}`
    })
    return parts.join('；')
  }
  if (p.process != null && p.process > 0) return `同步进度 ${p.process}%`
  return '正在同步商品，请耐心等待…'
})

function platformLabel(code: string) {
  return platformOptions.find((o) => o.value === code)?.label || code
}

function approveTagType(status?: string) {
  const s = (status || '').toLowerCase()
  if (s === 'onsale' || s === 'on_sale') return 'success'
  if (s === 'instock' || s === 'in_stock') return 'info'
  if (s === 'soldout' || s === 'sold_out') return 'warning'
  return 'info'
}

async function loadProducts() {
  loading.products = true
  try {
    const data = await listProducts({
      platform: filters.platform || undefined,
      shopId: filters.shopId || undefined,
      title: filters.title || undefined,
      pageNo: filters.pageNo,
      pageSize: filters.pageSize,
    })
    products.value = data.items || []
    total.value = data.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '加载商品失败')
  } finally {
    loading.products = false
  }
}

async function pollSyncProgress() {
  if (!filters.platform) return
  try {
    const data = await getProductSyncProgress(filters.platform)
    syncProgress.value = data
    if (data.finish) {
      stopProgressPoll()
      if (!data.errorMessage) {
        ElMessage.success('商品同步完成')
        await loadProducts()
      }
    }
  } catch {
    // ignore transient poll errors
  }
}

function startProgressPoll() {
  stopProgressPoll()
  void pollSyncProgress()
  progressTimer = setInterval(() => {
    void pollSyncProgress()
  }, 3000)
}

function stopProgressPoll() {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

async function handleSync() {
  if (!filters.platform) {
    ElMessage.warning('请选择平台')
    return
  }
  loading.sync = true
  try {
    await syncProducts({
      platform: filters.platform,
      shopIds: filters.shopId ? [filters.shopId] : undefined,
    })
    ElMessage.success('已发起商品同步')
    syncProgress.value = { finish: false, process: 0 }
    startProgressPoll()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '同步失败')
  } finally {
    loading.sync = false
  }
}

function onFilterChange() {
  filters.pageNo = 1
  void loadProducts()
}

function onPlatformChange() {
  filters.shopId = ''
  onFilterChange()
}

function onPageChange(page: number) {
  filters.pageNo = page
  void loadProducts()
}

function refreshPage() {
  void kdzsStore.loadShops()
  void loadProducts()
  if (syncing.value) {
    void pollSyncProgress()
  }
}

useAccountRefresh(refreshPage)

onMounted(async () => {
  await kdzsStore.loadShops()
  await loadProducts()
  await pollSyncProgress()
  if (syncProgress.value && !syncProgress.value.finish) {
    startProgressPoll()
  }
})

onBeforeUnmount(() => {
  stopProgressPoll()
})
</script>

<template>
  <div class="product-page">
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">商品列表 <span class="count">({{ total }})</span></div>
          <div class="actions">
            <el-select v-model="filters.platform" style="width: 120px" @change="onPlatformChange">
              <el-option v-for="opt in platformOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <el-select
              v-model="filters.shopId"
              clearable
              filterable
              placeholder="全部店铺"
              style="width: 200px"
              @change="onFilterChange"
            >
              <el-option
                v-for="shop in shopOptions"
                :key="shop.mallUserId"
                :label="shop.mallUserName"
                :value="shop.mallUserId"
              />
            </el-select>
            <el-input
              v-model="filters.title"
              clearable
              placeholder="商品标题"
              style="width: 180px"
              @keyup.enter="onFilterChange"
              @clear="onFilterChange"
            />
            <el-button type="warning" :loading="loading.sync" @click="handleSync">同步商品</el-button>
            <el-button type="primary" :loading="loading.products" @click="refreshPage">刷新</el-button>
          </div>
        </div>
      </template>

      <el-alert
        v-if="syncing"
        type="info"
        :closable="false"
        show-icon
        :title="progressText"
        class="hint"
      />

      <el-table
        :data="products"
        v-loading="loading.products"
        stripe
        border
        empty-text="暂无商品，请先同步"
        row-key="itemId"
      >
        <el-table-column type="expand">
          <template #default="{ row }">
            <div v-if="row.skus?.length" class="sku-expand">
              <el-table :data="row.skus" size="small" border>
                <el-table-column prop="propertiesName" label="规格" min-width="160" />
                <el-table-column prop="skuId" label="SKU ID" min-width="140" />
                <el-table-column prop="outerId" label="商家编码" min-width="120" />
                <el-table-column prop="price" label="价格" width="90" />
                <el-table-column prop="quantity" label="库存" width="80" />
                <el-table-column prop="status" label="状态" width="90" />
              </el-table>
            </div>
            <div v-else class="muted sku-expand">无 SKU 明细</div>
          </template>
        </el-table-column>
        <el-table-column label="图片" width="72">
          <template #default="{ row }">
            <el-image
              v-if="row.picUrl"
              :src="row.picUrl"
              fit="cover"
              style="width: 48px; height: 48px; border-radius: 4px"
              :preview-src-list="[row.picUrl]"
              preview-teleported
            />
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="商品标题" min-width="220" show-overflow-tooltip />
        <el-table-column prop="shopName" label="店铺" min-width="140" show-overflow-tooltip />
        <el-table-column label="平台" width="90">
          <template #default="{ row }">
            {{ row.platformName || platformLabel(row.platform || '') }}
          </template>
        </el-table-column>
        <el-table-column prop="price" label="价格" width="90" />
        <el-table-column prop="stock" label="库存" width="80" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="approveTagType(row.approveStatus)" size="small">
              {{ row.approveStatusLabel || row.approveStatus || '-' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="itemId" label="商品ID" min-width="140" show-overflow-tooltip />
        <el-table-column prop="outerId" label="商家编码" min-width="120" show-overflow-tooltip />
      </el-table>

      <div class="pager" v-if="total > filters.pageSize">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="total"
          :page-size="filters.pageSize"
          :current-page="filters.pageNo"
          @current-change="onPageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}
.hint {
  margin-bottom: 16px;
}
.sku-expand {
  padding: 8px 16px;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.muted {
  color: #909399;
}
</style>
