<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, RefreshRight } from '@element-plus/icons-vue'
import { useAccountRefresh } from '../../composables/useAccountRefresh'
import { listProducts, type Product } from '../../api'
import { useKdzsStore } from '../../stores/kdzs'

const kdzsStore = useKdzsStore()

interface ShopColumn {
  shopId: string
  shopName: string
  platform: string
  platformName: string
  loading: boolean
  error: string
  candidates: Product[]
  selected: Product | null
  expanded: boolean
}

const filters = reactive({
  type: '',
  shopIds: [] as string[],
  keywords: '',
})

const typeOptions = [
  { label: '全部状态', value: '' },
  { label: '上架', value: 'onsale' },
  { label: '下架', value: 'instock' },
]

const platformOptions = [
  { label: '抖店', value: 'FXG' },
  { label: '淘宝', value: 'TB' },
  { label: '小红书', value: 'XHS' },
]

const searching = ref(false)
const searched = ref(false)
const columns = ref<ShopColumn[]>([])
let searchSeq = 0

const shopOptions = computed(() =>
  [...kdzsStore.shops].sort((a, b) => {
    if (a.platform !== b.platform) return (a.platform || '').localeCompare(b.platform || '')
    return (a.mallUserName || '').localeCompare(b.mallUserName || '')
  }),
)

const selectedShopCount = computed(() => filters.shopIds.length)

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

/** 解析多关键字：空格 / 逗号 / 中文逗号 / 分号 */
function parseKeywords(raw: string): string[] {
  return raw
    .split(/[\s,，;；]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 精确：标题/简称/商家编码/商品ID 中同时包含全部关键字（忽略大小写） */
function matchExactKeywords(item: Product, keywords: string[]): boolean {
  if (!keywords.length) return true
  const hay = [
    item.title,
    item.shortTitle,
    item.outerId,
    item.itemId,
    item.productNum,
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
  return keywords.every((kw) => hay.includes(kw.toLowerCase()))
}

function shopMeta(shopId: string) {
  const shop = kdzsStore.shops.find((s) => s.mallUserId === shopId)
  return {
    shopId,
    shopName: shop?.mallUserName || shopId,
    platform: shop?.platform || '',
    platformName: shop?.platformName || platformLabel(shop?.platform || ''),
  }
}

async function fetchShopCandidates(shopId: string, platform: string, keywords: string[]): Promise<Product[]> {
  const primary = keywords[0] || undefined
  const data = await listProducts({
    platform: platform || undefined,
    shopId,
    type: filters.type || undefined,
    title: primary,
    pageNo: 1,
    pageSize: 50,
  })
  const items = data.items || []
  if (!keywords.length) return items
  return items.filter((p) => matchExactKeywords(p, keywords))
}

async function handleSearch() {
  if (!filters.shopIds.length) {
    ElMessage.warning('请先选择要比对的店铺')
    return
  }
  const keywords = parseKeywords(filters.keywords)
  if (!keywords.length) {
    ElMessage.warning('请输入至少一个搜索关键字')
    return
  }

  const seq = ++searchSeq
  searching.value = true
  searched.value = true

  columns.value = filters.shopIds.map((id) => {
    const meta = shopMeta(id)
    return {
      ...meta,
      loading: true,
      error: '',
      candidates: [],
      selected: null,
      expanded: true,
    }
  })

  await Promise.all(
    columns.value.map(async (col) => {
      try {
        const list = await fetchShopCandidates(col.shopId, col.platform, keywords)
        if (seq !== searchSeq) return
        col.candidates = list
        // 唯一命中时自动选定，便于直接展开比对
        if (list.length === 1) {
          col.selected = list[0]
        }
      } catch (e: any) {
        if (seq !== searchSeq) return
        col.error = e?.response?.data?.error || e.message || '加载失败'
        col.candidates = []
      } finally {
        if (seq === searchSeq) col.loading = false
      }
    }),
  )

  if (seq === searchSeq) {
    searching.value = false
    const hitShops = columns.value.filter((c) => c.candidates.length > 0).length
    if (!hitShops) {
      ElMessage.warning('各店铺均未找到匹配商品，请调整关键字')
    }
  }
}

function selectProduct(col: ShopColumn, product: Product) {
  col.selected = product
  col.expanded = true
}

function clearSelection(col: ShopColumn) {
  col.selected = null
}

function resetAll() {
  filters.type = ''
  filters.shopIds = []
  filters.keywords = ''
  columns.value = []
  searched.value = false
  searching.value = false
  searchSeq++
}

function refreshPage() {
  void kdzsStore.loadShops()
  if (searched.value && filters.shopIds.length) {
    void handleSearch()
  }
}

useAccountRefresh(refreshPage)

onMounted(async () => {
  await kdzsStore.loadShops()
})

onBeforeUnmount(() => {
  searchSeq++
})
</script>

<template>
  <div class="compare-page">
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">
            多店铺同商品比对
            <span v-if="selectedShopCount" class="count">（{{ selectedShopCount }} 店）</span>
          </div>
          <div class="actions">
            <el-button :icon="RefreshRight" @click="resetAll">重置</el-button>
            <el-button type="primary" :icon="Search" :loading="searching" @click="handleSearch">搜索比对</el-button>
          </div>
        </div>
      </template>

      <div class="filter-panel">
        <div class="filter-row">
          <span class="filter-label">商品状态</span>
          <el-radio-group v-model="filters.type">
            <el-radio-button v-for="opt in typeOptions" :key="opt.value || 'all'" :label="opt.value">
              {{ opt.label }}
            </el-radio-button>
          </el-radio-group>
        </div>
        <div class="filter-row">
          <span class="filter-label">比对店铺</span>
          <el-select
            v-model="filters.shopIds"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            clearable
            placeholder="选择多个平台店铺"
            style="flex: 1; max-width: 720px"
          >
            <el-option
              v-for="shop in shopOptions"
              :key="shop.mallUserId"
              :label="`${shop.mallUserName}（${shop.platformName || platformLabel(shop.platform)}）`"
              :value="shop.mallUserId"
            />
          </el-select>
        </div>
        <div class="filter-row">
          <span class="filter-label">快速搜索</span>
          <div class="filters">
            <el-input
              v-model="filters.keywords"
              clearable
              placeholder="精确关键字，支持多个（空格 / 逗号分隔），需同时命中"
              style="width: min(520px, 100%)"
              @keyup.enter="handleSearch"
            />
            <el-button type="primary" :loading="searching" @click="handleSearch">查询</el-button>
          </div>
        </div>
        <div class="filter-hint muted">
          每个店铺一列；搜索后在列内选定同一款商品，展开即可横向比对图片、规格、SKU ID、价格、库存。
        </div>
      </div>

      <el-empty
        v-if="!searched"
        description="选择店铺并输入关键字后开始比对"
        :image-size="80"
      />

      <div v-else class="columns-wrap" v-loading="searching">
        <div
          v-for="col in columns"
          :key="col.shopId"
          class="shop-column"
        >
          <div class="col-header">
            <div class="shop-name" :title="col.shopName">{{ col.shopName }}</div>
            <el-tag size="small" type="info">{{ col.platformName || platformLabel(col.platform) }}</el-tag>
          </div>

          <div v-if="col.loading" class="col-body muted">加载中…</div>
          <div v-else-if="col.error" class="col-body error">{{ col.error }}</div>
          <div v-else-if="!col.selected" class="col-body">
            <div class="cand-meta">
              命中 <strong>{{ col.candidates.length }}</strong> 款
              <span v-if="col.candidates.length > 1" class="muted"> · 请选定比对商品</span>
            </div>
            <el-empty
              v-if="!col.candidates.length"
              description="无匹配商品"
              :image-size="56"
            />
            <div
              v-for="p in col.candidates"
              :key="p.itemId"
              class="cand-card"
              @click="selectProduct(col, p)"
            >
              <el-image
                v-if="p.picUrl"
                :src="p.picUrl"
                fit="cover"
                class="cand-thumb"
                referrerpolicy="no-referrer"
              />
              <div v-else class="cand-thumb placeholder">无图</div>
              <div class="cand-info">
                <div class="cand-title">{{ p.title || '-' }}</div>
                <div class="cand-sub muted">
                  ID {{ p.itemId }}
                  <template v-if="p.outerId"> · {{ p.outerId }}</template>
                </div>
                <div class="cand-stats">
                  <span>¥{{ p.price || '-' }}</span>
                  <span>库存 {{ p.stock ?? '-' }}</span>
                  <el-tag :type="approveTagType(p.approveStatus)" size="small">
                    {{ p.approveStatusLabel || p.approveStatus || '-' }}
                  </el-tag>
                </div>
              </div>
            </div>
          </div>

          <div v-else class="col-body selected-body">
            <div class="selected-head">
              <el-image
                v-if="col.selected.picUrl"
                :src="col.selected.picUrl"
                fit="cover"
                class="spu-thumb"
                referrerpolicy="no-referrer"
                :preview-src-list="[col.selected.picUrl]"
                preview-teleported
              />
              <div class="selected-meta">
                <div class="cand-title">{{ col.selected.title }}</div>
                <div class="cand-sub muted">
                  ID {{ col.selected.itemId }}
                  <template v-if="col.selected.outerId"> · {{ col.selected.outerId }}</template>
                </div>
                <div class="cand-stats">
                  <span>¥{{ col.selected.price || '-' }}</span>
                  <span>库存 {{ col.selected.stock ?? '-' }}</span>
                  <el-tag :type="approveTagType(col.selected.approveStatus)" size="small">
                    {{ col.selected.approveStatusLabel || col.selected.approveStatus || '-' }}
                  </el-tag>
                </div>
                <div class="selected-actions">
                  <el-button link type="primary" @click="col.expanded = !col.expanded">
                    {{ col.expanded ? '收起 SKU' : '展开 SKU' }}
                  </el-button>
                  <el-button link @click="clearSelection(col)">重选</el-button>
                </div>
              </div>
            </div>

            <div v-show="col.expanded" class="sku-panel">
              <el-table
                v-if="col.selected.skus?.length"
                :data="col.selected.skus"
                size="small"
                border
                stripe
                max-height="480"
              >
                <el-table-column label="图片" width="64">
                  <template #default="{ row: sku }">
                    <el-image
                      v-if="sku.picUrl || col.selected?.picUrl"
                      :src="sku.picUrl || col.selected?.picUrl"
                      fit="cover"
                      class="sku-thumb"
                      referrerpolicy="no-referrer"
                      :preview-src-list="[sku.picUrl || col.selected?.picUrl || '']"
                      preview-teleported
                    />
                    <span v-else class="muted">-</span>
                  </template>
                </el-table-column>
                <el-table-column prop="propertiesName" label="规格" min-width="120" show-overflow-tooltip />
                <el-table-column prop="skuId" label="SKU ID" min-width="110" show-overflow-tooltip />
                <el-table-column prop="price" label="价格" width="80" />
                <el-table-column prop="quantity" label="库存" width="70" />
              </el-table>
              <div v-else class="muted sku-empty">无 SKU 明细</div>
            </div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.row-between {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.card-title {
  font-weight: 600;
  font-size: 16px;
}
.count {
  color: #909399;
  font-weight: 400;
  font-size: 13px;
}
.actions {
  display: flex;
  gap: 8px;
}
.filter-panel {
  margin-bottom: 16px;
  padding: 16px;
  background: #fafafa;
  border-radius: 6px;
}
.filter-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}
.filter-row:last-child {
  margin-bottom: 0;
}
.filter-label {
  width: 72px;
  flex-shrink: 0;
  line-height: 32px;
  color: #606266;
  font-size: 13px;
}
.filters {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  flex: 1;
}
.filter-hint {
  margin-top: 8px;
  margin-left: 84px;
  font-size: 12px;
  line-height: 1.5;
}
.muted {
  color: #909399;
}
.error {
  color: #f56c6c;
}
.columns-wrap {
  display: flex;
  gap: 16px;
  overflow-x: auto;
  padding-bottom: 8px;
  min-height: 320px;
  align-items: flex-start;
}
.shop-column {
  flex: 0 0 380px;
  width: 380px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fff;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 280px);
}
.col-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 12px 14px;
  border-bottom: 1px solid #ebeef5;
  background: #f8fafc;
  border-radius: 8px 8px 0 0;
}
.shop-name {
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.col-body {
  padding: 12px;
  overflow-y: auto;
  flex: 1;
}
.cand-meta {
  font-size: 13px;
  margin-bottom: 10px;
}
.cand-card {
  display: flex;
  gap: 10px;
  padding: 10px;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  margin-bottom: 8px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.cand-card:hover {
  border-color: #409eff;
  background: #f0f7ff;
}
.cand-thumb,
.spu-thumb {
  width: 56px;
  height: 56px;
  border-radius: 4px;
  flex-shrink: 0;
}
.cand-thumb.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  color: #c0c4cc;
  font-size: 12px;
}
.cand-info,
.selected-meta {
  min-width: 0;
  flex: 1;
}
.cand-title {
  font-size: 13px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.cand-sub {
  font-size: 12px;
  margin-top: 4px;
}
.cand-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-top: 6px;
  font-size: 12px;
  color: #606266;
}
.selected-head {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}
.selected-actions {
  margin-top: 6px;
  display: flex;
  gap: 4px;
}
.sku-panel {
  border-top: 1px dashed #e4e7ed;
  padding-top: 10px;
}
.sku-thumb {
  width: 40px;
  height: 40px;
  border-radius: 4px;
}
.sku-empty {
  padding: 12px 0;
  text-align: center;
  font-size: 13px;
}
</style>
