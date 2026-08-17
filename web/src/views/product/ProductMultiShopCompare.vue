<script setup lang="ts">
import { computed, inject, onBeforeUnmount, onMounted, reactive, ref, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Rank, RefreshRight } from '@element-plus/icons-vue'
import { useAccountRefresh } from '../../composables/useAccountRefresh'
import { listProducts, type Product } from '../../api'
import { useKdzsStore } from '../../stores/kdzs'

const kdzsStore = useKdzsStore()
const sidebarCollapsed = inject<Ref<boolean>>('sidebarCollapsed', ref(false))

interface ShopColumn {
  shopId: string
  shopName: string
  platform: string
  platformName: string
  loading: boolean
  error: string
  /** 远端按「全部状态」拉取后的缓存 */
  allCandidates: Product[]
  /** 按当前商品状态筛过的展示列表 */
  candidates: Product[]
  selected: Product | null
  expanded: boolean
}

const filters = reactive({
  type: '',
  shopIds: [] as string[],
  /** 多关键字标签（回车添加，支持粘贴空格/逗号串） */
  keywordTags: [] as string[],
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
const dragFromIndex = ref<number | null>(null)
const dragOverIndex = ref<number | null>(null)
/** 当前缓存对应的关键字签名；变更才重新打远端 */
let cacheKeywordsKey = ''
let searchSeq = 0

const shopOptions = computed(() =>
  [...kdzsStore.shops].sort((a, b) => {
    if (a.platform !== b.platform) return (a.platform || '').localeCompare(b.platform || '')
    return (a.mallUserName || '').localeCompare(b.mallUserName || '')
  }),
)

const selectedShopCount = computed(() => filters.shopIds.length)

/** 列少时均分铺满；列多时保底宽度并可横向滚动。侧栏收起后主区变宽，flex 列自动变宽。 */
const columnsWrapClass = computed(() => ({
  'is-fit': columns.value.length > 0 && columns.value.length <= 5,
  'is-scroll': columns.value.length > 5,
  'sidebar-collapsed': sidebarCollapsed.value,
}))

const columnStyle = computed(() => {
  const n = columns.value.length || 1
  if (n <= 5) {
    return {
      flex: '1 1 0',
      minWidth: n <= 2 ? '320px' : n <= 3 ? '280px' : '220px',
      width: 'auto',
      maxWidth: '100%',
    }
  }
  return {
    flex: '0 0 280px',
    minWidth: '280px',
    width: '280px',
  }
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

/** 规范化关键字标签：拆分粘贴的「空格/逗号」串并去重 */
function normalizeKeywordTags(tags: string[]): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  for (const raw of tags) {
    for (const part of String(raw).split(/[\s,，;；]+/)) {
      const t = part.trim()
      if (!t) continue
      const key = t.toLowerCase()
      if (seen.has(key)) continue
      seen.add(key)
      out.push(t)
    }
  }
  return out
}

function activeKeywords(): string[] {
  return normalizeKeywordTags(filters.keywordTags)
}

function keywordsCacheKey(keywords: string[] = activeKeywords()): string {
  return keywords
    .map((k) => k.toLowerCase())
    .sort()
    .join('\u0001')
}

/** 精确：标题/简称/编码/商品ID/SKU 规格中同时包含全部关键字 */
function matchExactKeywords(item: Product, keywords: string[]): boolean {
  if (!keywords.length) return true
  const skuText = (item.skus || [])
    .map((s) => [s.propertiesName, s.skuId, s.outerId, s.shortTitle].filter(Boolean).join(' '))
    .join(' ')
  const hay = [
    item.title,
    item.shortTitle,
    item.outerId,
    item.itemId,
    item.productNum,
    skuText,
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
  return keywords.every((kw) => hay.includes(kw.toLowerCase()))
}

function matchStatus(item: Product, type: string): boolean {
  if (!type) return true
  const s = (item.approveStatus || '').toLowerCase().replace(/-/g, '_')
  if (type === 'onsale') return s === 'onsale' || s === 'on_sale'
  if (type === 'instock') return s === 'instock' || s === 'in_stock'
  return true
}

function filterByType(items: Product[]): Product[] {
  return items.filter((p) => matchStatus(p, filters.type))
}

function applyTypeFilter(col: ShopColumn) {
  col.candidates = filterByType(col.allCandidates)
  if (col.selected) {
    const still = col.candidates.find((p) => p.itemId === col.selected?.itemId)
    if (!still) {
      col.selected = col.candidates.length === 1 ? col.candidates[0] : null
    } else {
      col.selected = still
    }
  } else if (col.candidates.length === 1) {
    col.selected = col.candidates[0]
  }
}

function canSearch(): boolean {
  return filters.shopIds.length > 0 && activeKeywords().length > 0
}

/** 状态切换：只从本地缓存筛选，不打远端 */
function onTypeChange() {
  if (!searched.value) return
  for (const col of columns.value) {
    applyTypeFilter(col)
  }
}

function onKeywordTagsChange(val: string[]) {
  filters.keywordTags = normalizeKeywordTags(val || [])
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
  // 始终按「全部状态」拉取，状态筛选项只做本地缓存过滤
  const primary =
    keywords.length <= 1
      ? keywords[0]
      : [...keywords].sort((a, b) => a.length - b.length)[0]
  const data = await listProducts({
    platform: platform || undefined,
    shopId,
    title: primary || undefined,
    pageNo: 1,
    pageSize: 100,
  })
  const items = data.items || []
  if (!keywords.length) return items
  return items.filter((p) => matchExactKeywords(p, keywords))
}

async function handleSearch() {
  filters.keywordTags = normalizeKeywordTags(filters.keywordTags)
  if (!filters.shopIds.length) {
    ElMessage.warning('请先选择要比对的店铺')
    return
  }
  const keywords = activeKeywords()
  if (!keywords.length) {
    ElMessage.warning('请输入至少一个搜索关键字（回车添加，可多个）')
    return
  }

  const seq = ++searchSeq
  const nextKey = keywordsCacheKey(keywords)
  const keywordsChanged = nextKey !== cacheKeywordsKey
  searching.value = true
  searched.value = true

  // 保留已有列顺序（拖动后），仅补全新选店铺
  const prevById = new Map(columns.value.map((c) => [c.shopId, c]))
  const orderedIds = [
    ...columns.value.map((c) => c.shopId).filter((id) => filters.shopIds.includes(id)),
    ...filters.shopIds.filter((id) => !prevById.has(id)),
  ]

  columns.value = orderedIds.map((id) => {
    const meta = shopMeta(id)
    const prev = prevById.get(id)
    const canReuse = !keywordsChanged && !!prev && prev.allCandidates.length >= 0 && !prev.error
    return {
      ...meta,
      loading: !canReuse,
      error: canReuse ? '' : '',
      allCandidates: canReuse ? prev!.allCandidates : [],
      candidates: [],
      selected: prev?.selected && prev.selected.itemId ? prev.selected : null,
      expanded: prev?.expanded ?? true,
    }
  })

  // 关键字未变：已有缓存的列只做状态筛选；新店铺或缺缓存的才请求
  await Promise.all(
    columns.value.map(async (col) => {
      const prev = prevById.get(col.shopId)
      const canReuse = !keywordsChanged && !!prev && !prev.error && Array.isArray(prev.allCandidates)
      if (canReuse) {
        col.allCandidates = prev!.allCandidates
        applyTypeFilter(col)
        col.loading = false
        return
      }
      try {
        const list = await fetchShopCandidates(col.shopId, col.platform, keywords)
        if (seq !== searchSeq) return
        col.allCandidates = list
        col.error = ''
        applyTypeFilter(col)
        if (!col.selected && col.candidates.length === 1) {
          col.selected = col.candidates[0]
        }
      } catch (e: any) {
        if (seq !== searchSeq) return
        col.error = e?.response?.data?.error || e.message || '加载失败'
        col.allCandidates = []
        col.candidates = []
        col.selected = null
      } finally {
        if (seq === searchSeq) col.loading = false
      }
    }),
  )

  if (seq === searchSeq) {
    cacheKeywordsKey = nextKey
    searching.value = false
    const hitShops = columns.value.filter((c) => c.candidates.length > 0).length
    if (!hitShops) {
      ElMessage.warning('各店铺均未找到匹配商品，请调整关键字或状态')
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

function syncShopIdsOrder() {
  filters.shopIds = columns.value.map((c) => c.shopId)
}

function onColumnDragStart(index: number, e: DragEvent) {
  dragFromIndex.value = index
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', String(index))
  }
}

function onColumnDragOver(index: number, e: DragEvent) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  dragOverIndex.value = index
}

function onColumnDrop(index: number, e: DragEvent) {
  e.preventDefault()
  const from = dragFromIndex.value
  dragFromIndex.value = null
  dragOverIndex.value = null
  if (from == null || from === index) return
  const list = [...columns.value]
  const [moved] = list.splice(from, 1)
  list.splice(index, 0, moved)
  columns.value = list
  syncShopIdsOrder()
}

function onColumnDragEnd() {
  dragFromIndex.value = null
  dragOverIndex.value = null
}

function resetAll() {
  filters.type = ''
  filters.shopIds = []
  filters.keywordTags = []
  columns.value = []
  searched.value = false
  searching.value = false
  cacheKeywordsKey = ''
  searchSeq++
}

function refreshPage() {
  void kdzsStore.loadShops()
  // 账号切换后缓存可能失效，强制按关键字重新拉
  if (searched.value && filters.shopIds.length) {
    cacheKeywordsKey = ''
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
  <div class="compare-page" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">
            同商品比对
            <span v-if="selectedShopCount" class="count">（{{ selectedShopCount }} 店）</span>
          </div>
          <div class="actions">
            <el-button :icon="RefreshRight" @click="resetAll">重置</el-button>
          </div>
        </div>
      </template>

      <div class="filter-panel">
        <div class="filter-row">
          <span class="filter-label">商品状态</span>
          <el-radio-group v-model="filters.type" @change="onTypeChange">
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
            <el-select
              v-model="filters.keywordTags"
              multiple
              filterable
              allow-create
              default-first-option
              collapse-tags
              collapse-tags-tooltip
              clearable
              placeholder="输入关键字后回车添加，可多个；需同时命中"
              style="width: min(520px, 100%)"
              @change="onKeywordTagsChange"
            />
            <el-button type="primary" :loading="searching" @click="handleSearch">查询</el-button>
          </div>
        </div>
        <div class="filter-hint muted">
          首次按全部状态拉取并缓存；切换上架/下架仅本地筛选。关键字变更后才会重新查询。
        </div>
      </div>

      <el-empty
        v-if="!searched"
        description="选择店铺并输入关键字后开始比对"
        :image-size="80"
      />

      <div v-else class="columns-wrap" :class="columnsWrapClass" v-loading="searching">
        <div
          v-for="(col, index) in columns"
          :key="col.shopId"
          class="shop-column"
          :class="{
            'is-dragging': dragFromIndex === index,
            'is-drag-over': dragOverIndex === index && dragFromIndex !== index,
          }"
          :style="columnStyle"
          draggable="true"
          @dragstart="onColumnDragStart(index, $event)"
          @dragover="onColumnDragOver(index, $event)"
          @drop="onColumnDrop(index, $event)"
          @dragend="onColumnDragEnd"
        >
          <div class="col-header" title="拖动调整列顺序">
            <el-icon class="drag-handle"><Rank /></el-icon>
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
.compare-page {
  min-width: 0;
}
.page-card {
  min-width: 0;
}
.page-card :deep(.el-card__body) {
  min-width: 0;
}
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
  gap: 12px;
  width: 100%;
  min-width: 0;
  overflow-x: auto;
  padding-bottom: 8px;
  min-height: 320px;
  align-items: stretch;
}
.columns-wrap.is-fit {
  overflow-x: hidden;
}
.shop-column {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #fff;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 280px);
  min-width: 0;
  transition: box-shadow 0.15s, border-color 0.15s, opacity 0.15s;
}
.shop-column.is-dragging {
  opacity: 0.55;
}
.shop-column.is-drag-over {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.25);
}
.col-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  border-bottom: 1px solid #ebeef5;
  background: #f8fafc;
  border-radius: 8px 8px 0 0;
  cursor: grab;
  user-select: none;
}
.col-header:active {
  cursor: grabbing;
}
.drag-handle {
  color: #909399;
  flex-shrink: 0;
}
.shop-name {
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.col-body {
  padding: 12px;
  overflow-y: auto;
  flex: 1;
  min-width: 0;
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
  min-width: 0;
  overflow-x: auto;
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
