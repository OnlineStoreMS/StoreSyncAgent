<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useAccountRefresh } from '../../composables/useAccountRefresh'
import { ElMessage } from 'element-plus'
import { decryptOrders, listOrders, setOrderAgentType, type Order, type OrderFilters, type OrderListResponse, type TradeGoods } from '../../api'
import { useKdzsStore } from '../../stores/kdzs'
import { copyToClipboard } from '../../utils/clipboard'
import { dateShortcuts, defaultDateRange, formatOrderCopyDateTime } from '../../utils/date'

const kdzsStore = useKdzsStore()
const route = useRoute()

const loading = reactive({ orders: false, decrypt: false, agent: false, decryptRow: {} as Record<string, boolean> })
const orders = ref<Order[]>([])
const selectedOrders = ref<Order[]>([])
const orderTotal = ref(0)
const pushDialogVisible = ref(false)
const selectedFactoryId = ref('')

const [defaultStart, defaultEnd] = defaultDateRange()

const filters = reactive({
  platform: 'FXG',
  shopId: '',
  tradeStatus: 'wait_audit',
  timeType: 0,
  dateRange: [defaultStart, defaultEnd] as [string, string],
  pageNo: 1,
  pageSize: 20,
})

const orderStats = ref<OrderListResponse['stats']>()
const orderHint = ref('')
const appliedFilters = ref<OrderFilters>()

const platformOptions = [
  { label: '抖店', value: 'FXG' },
  { label: '淘宝', value: 'TB' },
  { label: '小红书', value: 'XHS' },
]

const statusOptions = [
  { label: '待推单', value: 'wait_audit' },
  { label: '待发货', value: 'wait_send' },
  { label: '全部', value: 'all' },
]

const timeTypeOptions = [
  { label: '下单时间', value: 0 },
  { label: '发货时间', value: 1 },
]

const selectedShopName = computed(() => {
  if (!filters.shopId) return ''
  return kdzsStore.shops.find((s) => s.mallUserId === filters.shopId)?.mallUserName || filters.shopId
})

const filterSummaryTags = computed(() => {
  const f = appliedFilters.value
  if (!f) return []
  const tags = [
    `${f.timeTypeLabel}：${f.startDateTime} ~ ${f.endDateTime}`,
    `状态：${f.statusLabel}`,
  ]
  if (f.platformName) tags.push(`平台：${f.platformName}`)
  if (selectedShopName.value) tags.push(`店铺：${selectedShopName.value}`)
  return tags
})

const statCards = computed(() => [
  {
    label: '待推单',
    subLabel: '首页统计',
    value: orderStats.value?.waitingPushTotal,
    color: '#e6a23c',
    tradeStatus: 'wait_audit',
  },
  {
    label: '待发货',
    subLabel: '首页统计',
    value: orderStats.value?.waitingSendTotal,
    color: '#409eff',
    tradeStatus: 'wait_send',
  },
])

async function loadOrders() {
  loading.orders = true
  try {
    const [startDateTime, endDateTime] = filters.dateRange
    const data = await listOrders({
      platform: filters.platform || undefined,
      shopId: filters.shopId || undefined,
      tradeStatus: filters.tradeStatus,
      timeType: filters.timeType,
      startDateTime,
      endDateTime,
      pageNo: filters.pageNo,
      pageSize: filters.pageSize,
    })
    orders.value = data.items || []
    orderTotal.value = data.total || 0
    orderStats.value = data.stats
    orderHint.value = data.hint || ''
    appliedFilters.value = data.filters
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '加载订单失败')
  } finally {
    loading.orders = false
  }
}

function onFilterChange() {
  filters.pageNo = 1
  loadOrders()
}

function onStatCardClick(tradeStatus: string) {
  if (filters.tradeStatus === tradeStatus) return
  filters.tradeStatus = tradeStatus
  filters.pageNo = 1
  selectedOrders.value = []
  loadOrders()
}

function onPageChange(page: number) {
  filters.pageNo = page
  loadOrders()
}

function orderSysTid(order: Order) {
  return order.sysTids?.[0] || ''
}

function applyDecryptedItems(items: Order[]) {
  const bySysTid = new Map(items.map((item) => [orderSysTid(item), item]))
  orders.value = orders.value.map((order) => {
    const sysTid = orderSysTid(order)
    const decrypted = sysTid ? bySysTid.get(sysTid) : undefined
    if (!decrypted) return order
    return {
      ...order,
      receiverName: decrypted.receiverName,
      receiverMobile: decrypted.receiverMobile,
      receiverAddress: decrypted.receiverAddress,
      formattedReceiver: decrypted.formattedReceiver,
      decrypted: true,
    }
  })
}

async function copyText(text: string) {
  const ok = await copyToClipboard(text)
  if (ok) {
    ElMessage.success('已复制')
  } else {
    ElMessage.error('复制失败')
  }
}

function formatOrderGoodsLines(goods?: TradeGoods[]): string {
  return (goods || [])
    .map((g) => {
      const spec = g.skuName?.trim() || g.title?.trim() || g.outerId?.trim() || ''
      if (!spec) return ''
      const num = g.num && g.num > 0 ? g.num : 1
      return `${spec} x${num}`
    })
    .filter(Boolean)
    .join('\n')
}

function buildOrderCopyText(order: Order): string {
  const address = order.formattedReceiver?.trim() || ''
  const goodsBlock = formatOrderGoodsLines(order.goods)
  const lines = [formatOrderCopyDateTime(new Date()), '', address, '---']
  if (goodsBlock) lines.push(goodsBlock)
  return lines.join('\n')
}

async function copyOrderText(order: Order) {
  if (!order.formattedReceiver?.trim()) {
    ElMessage.warning('暂无收件信息')
    return
  }
  await copyText(buildOrderCopyText(order))
}

async function decryptOne(order: Order) {
  const sysTid = orderSysTid(order)
  if (!sysTid) {
    ElMessage.warning('缺少系统订单号')
    return
  }
  loading.decryptRow[sysTid] = true
  try {
    const data = await decryptOrders({
      platform: order.platform || filters.platform,
      tradeStatus: filters.tradeStatus,
      sysTids: [sysTid],
    })
    applyDecryptedItems(data.items || [])
    ElMessage.success('解密成功')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '解密失败')
  } finally {
    loading.decryptRow[sysTid] = false
  }
}

async function decryptAll() {
  const sysTids = orders.value.map(orderSysTid).filter(Boolean)
  if (!sysTids.length) {
    ElMessage.warning('当前页没有可解密的订单')
    return
  }
  loading.decrypt = true
  try {
    const data = await decryptOrders({
      platform: filters.platform,
      tradeStatus: filters.tradeStatus,
      sysTids,
    })
    applyDecryptedItems(data.items || [])
    ElMessage.success(`已解密 ${data.items?.length || 0} 条订单`)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '批量解密失败')
  } finally {
    loading.decrypt = false
  }
}

function selectedSysTids(list = selectedOrders.value) {
  return list.map(orderSysTid).filter(Boolean)
}

function showAgentResult(result: { successList?: string[]; failList?: string[]; failMessageMap?: Record<string, string> }, actionLabel: string) {
  const ok = result.successList?.length || 0
  const fail = result.failList?.length || 0
  if (fail === 0) {
    ElMessage.success(`${actionLabel}成功 ${ok} 条`)
  } else {
    const firstFail = result.failList?.[0]
    const msg = firstFail && result.failMessageMap?.[firstFail]
    ElMessage.warning(`${actionLabel}：成功 ${ok} 条，失败 ${fail} 条${msg ? `（${msg}）` : ''}`)
  }
}

async function setSelfPrint() {
  const sysTids = selectedSysTids()
  if (!sysTids.length) {
    ElMessage.warning('请先勾选订单')
    return
  }
  loading.agent = true
  try {
    const result = await setOrderAgentType({
      platform: filters.platform,
      tradeStatus: filters.tradeStatus,
      action: 'self_print',
      sysTids,
    })
    showAgentResult(result, '设置自己打单')
    await loadOrders()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '设置自己打单失败')
  } finally {
    loading.agent = false
  }
}

function openPushDialog() {
  const sysTids = selectedSysTids()
  if (!sysTids.length) {
    ElMessage.warning('请先勾选订单')
    return
  }
  if (!kdzsStore.factories.length) {
    void kdzsStore.loadFactories(filters.platform)
  }
  selectedFactoryId.value = kdzsStore.factories[0]?.factoryId || ''
  pushDialogVisible.value = true
}

async function pushToFactory() {
  const sysTids = selectedSysTids()
  if (!sysTids.length || !selectedFactoryId.value) {
    ElMessage.warning('请选择厂家和订单')
    return
  }
  loading.agent = true
  try {
    const result = await setOrderAgentType({
      platform: filters.platform,
      tradeStatus: filters.tradeStatus,
      action: 'push_factory',
      factoryId: selectedFactoryId.value,
      sysTids,
    })
    pushDialogVisible.value = false
    showAgentResult(result, '推送给厂家')
    await loadOrders()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '推送失败')
  } finally {
    loading.agent = false
  }
}

function formatRemarks(row: Order) {
  const parts: string[] = []
  if (row.buyerMemo) parts.push(`买家留言：${row.buyerMemo}`)
  if (row.sellerMemo) parts.push(`卖家备注：${row.sellerMemo}`)
  if (row.fenFaMemo) parts.push(`分发备注：${row.fenFaMemo}`)
  if (row.printerMemo) parts.push(`打单备注：${row.printerMemo}`)
  return parts
}

function compareNullableNumber(a: Order, b: Order, pick: (row: Order) => number | undefined) {
  const va = pick(a)
  const vb = pick(b)
  if (va == null && vb == null) return 0
  if (va == null) return 1
  if (vb == null) return -1
  return va - vb
}

function compareNullableTime(a: Order, b: Order, pick: (row: Order) => string | undefined) {
  const va = pick(a)?.trim() || ''
  const vb = pick(b)?.trim() || ''
  if (!va && !vb) return 0
  if (!va) return 1
  if (!vb) return -1
  return va.localeCompare(vb)
}

function sortByPayment(a: Order, b: Order) {
  return compareNullableNumber(a, b, (row) => row.payment)
}

function sortByCreateTime(a: Order, b: Order) {
  return compareNullableTime(a, b, (row) => row.createTime)
}

function sortByPayTime(a: Order, b: Order) {
  return compareNullableTime(a, b, (row) => row.payTime)
}

async function refreshForAccountSwitch() {
  filters.shopId = ''
  filters.pageNo = 1
  selectedOrders.value = []
  await kdzsStore.loadFactories(filters.platform)
  await loadOrders()
}

useAccountRefresh(refreshForAccountSwitch)

onMounted(async () => {
  const tradeStatus = route.query.tradeStatus
  if (typeof tradeStatus === 'string' && statusOptions.some((o) => o.value === tradeStatus)) {
    filters.tradeStatus = tradeStatus
  }
  if (!kdzsStore.shops.length) {
    await kdzsStore.loadShops()
  }
  if (!kdzsStore.factories.length) {
    await kdzsStore.loadFactories(filters.platform)
  }
  await loadOrders()
})
</script>

<template>
  <div class="order-page">
    <el-card shadow="never" class="page-card" v-if="orderStats">
      <div class="stats-row">
        <div
          v-for="card in statCards"
          :key="card.tradeStatus"
          class="stat-item clickable"
          :class="{ active: filters.tradeStatus === card.tradeStatus }"
          @click="onStatCardClick(card.tradeStatus)"
        >
          <div class="stat-label">{{ card.label }}（{{ card.subLabel }}）</div>
          <div class="stat-value" :style="{ color: filters.tradeStatus === card.tradeStatus ? card.color : undefined }">
            {{ card.value ?? '—' }}
          </div>
          <div
            class="stat-sub muted"
            v-if="card.tradeStatus === 'wait_audit' && orderStats.waitingPushByPlatform?.FXG"
          >
            抖店 {{ orderStats.waitingPushByPlatform.FXG }}
          </div>
        </div>
      </div>
      <el-alert v-if="orderHint" type="warning" :title="orderHint" show-icon :closable="false" class="hint" />
    </el-card>

    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">订单列表 <span class="count">({{ orderTotal }})</span></div>
          <div class="toolbar">
            <el-button :disabled="!selectedOrders.length" :loading="loading.agent" @click="setSelfPrint">设置自己打单</el-button>
            <el-button type="primary" plain :disabled="!selectedOrders.length" :loading="loading.agent" @click="openPushDialog">
              推送给厂家
            </el-button>
            <el-button type="warning" plain :loading="loading.decrypt" :disabled="!orders.length" @click="decryptAll">
              一键解密收件信息
            </el-button>
          </div>
        </div>
      </template>

      <div class="filter-panel">
        <div class="filter-row">
          <span class="filter-label">时间类型</span>
          <el-radio-group v-model="filters.timeType" @change="onFilterChange">
            <el-radio-button v-for="opt in timeTypeOptions" :key="opt.value" :label="opt.value">
              {{ opt.label }}
            </el-radio-button>
          </el-radio-group>
        </div>
        <div class="filter-row">
          <span class="filter-label">时间范围</span>
          <el-date-picker
            v-model="filters.dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            :shortcuts="dateShortcuts"
            style="width: 420px"
            @change="onFilterChange"
          />
        </div>
        <div class="filter-row">
          <span class="filter-label">其他条件</span>
          <div class="filters">
            <el-select v-model="filters.platform" placeholder="平台" clearable style="width: 120px" @change="onFilterChange">
              <el-option v-for="opt in platformOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <el-select v-model="filters.shopId" placeholder="店铺" clearable style="width: 200px" @change="onFilterChange">
              <el-option v-for="shop in kdzsStore.shops" :key="shop.mallUserId" :label="shop.mallUserName" :value="shop.mallUserId" />
            </el-select>
            <el-select v-model="filters.tradeStatus" style="width: 120px" @change="onFilterChange">
              <el-option v-for="opt in statusOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
            </el-select>
            <el-button type="primary" :loading="loading.orders" @click="loadOrders">查询</el-button>
          </div>
        </div>
        <div class="filter-summary" v-if="filterSummaryTags.length">
          <span class="filter-label">当前筛选</span>
          <div class="filter-tags">
            <el-tag v-for="tag in filterSummaryTags" :key="tag" type="info" effect="plain">{{ tag }}</el-tag>
          </div>
        </div>
      </div>

      <el-table
        :data="orders"
        v-loading="loading.orders"
        stripe
        border
        empty-text="暂无订单"
        :default-sort="{ prop: 'payTime', order: 'descending' }"
        @selection-change="(rows: Order[]) => (selectedOrders = rows)"
      >
        <el-table-column type="selection" width="48" fixed="left" />
        <el-table-column prop="platformName" label="平台" width="90" />
        <el-table-column label="订单号" min-width="200">
          <template #default="{ row }">
            <div v-if="row.tids?.length">平台：{{ row.tids.join(', ') }}</div>
            <div v-if="row.sysTids?.length" class="muted">系统：{{ row.sysTids.join(', ') }}</div>
            <div v-if="!(row.tids || []).length && !(row.sysTids || []).length && row.togetherId" class="muted">{{ row.togetherId }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="shopName" label="店铺" min-width="140" />
        <el-table-column prop="buyerNick" label="买家" width="120" />
        <el-table-column label="商品" min-width="280">
          <template #default="{ row }">
            <div v-for="(g, idx) in row.goods || []" :key="idx" class="goods-line">
              <div class="goods-row">
                <el-image
                  v-if="g.picUrl"
                  :src="g.picUrl"
                  :preview-src-list="[g.picUrl]"
                  fit="cover"
                  class="goods-pic"
                  preview-teleported
                />
                <div class="goods-info">
                  <div>{{ g.title }}</div>
                  <div v-if="g.skuName" class="muted">{{ g.skuName }}<span v-if="g.num"> x{{ g.num }}</span></div>
                </div>
              </div>
            </div>
            <span v-if="!(row.goods || []).length" class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="留言备注" min-width="220">
          <template #default="{ row }">
            <div v-for="(line, idx) in formatRemarks(row)" :key="idx" class="remark-line">{{ line }}</div>
            <div v-if="row.factoryName" class="muted">厂家：{{ row.factoryName }}</div>
            <span v-if="!formatRemarks(row).length && !row.factoryName" class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="收件信息" min-width="280">
          <template #default="{ row }">
            <template v-if="row.decrypted && row.formattedReceiver">
              <div class="decrypted-line">{{ row.formattedReceiver }}</div>
              <div class="receiver-actions">
                <el-button link type="primary" size="small" @click="copyOrderText(row)">复制</el-button>
              </div>
            </template>
            <template v-else>
              <div>{{ row.receiverName || '-' }}</div>
              <div class="muted">{{ row.receiverMobile }}</div>
              <div class="muted address">{{ row.receiverAddress }}</div>
            </template>
          </template>
        </el-table-column>
        <el-table-column prop="payment" label="金额" width="100" sortable :sort-method="sortByPayment">
          <template #default="{ row }">{{ row.payment ? `¥${row.payment}` : '-' }}</template>
        </el-table-column>
        <el-table-column
          prop="createTime"
          label="下单时间"
          width="170"
          sortable
          :sort-method="sortByCreateTime"
        />
        <el-table-column
          prop="payTime"
          label="付款时间"
          width="170"
          sortable
          :sort-method="sortByPayTime"
        />
        <el-table-column prop="statusText" label="状态" width="100">
          <template #default="{ row }">{{ row.statusText || row.tradeStatus || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="!row.decrypted"
              link
              type="primary"
              size="small"
              :loading="loading.decryptRow[orderSysTid(row)]"
              @click="decryptOne(row)"
            >
              解密
            </el-button>
            <el-button
              v-else-if="row.formattedReceiver"
              link
              type="primary"
              size="small"
              @click="copyOrderText(row)"
            >
              复制
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="orderTotal"
          :page-size="filters.pageSize"
          :current-page="filters.pageNo"
          @current-change="onPageChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="pushDialogVisible" title="推送给厂家" width="420px">
      <el-form label-width="88px">
        <el-form-item label="选择厂家">
          <el-select v-model="selectedFactoryId" placeholder="请选择厂家" style="width: 100%">
            <el-option
              v-for="f in kdzsStore.factories"
              :key="f.factoryId"
              :label="`${f.remark || f.factoryName} (${f.factoryId})`"
              :value="f.factoryId"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="订单数">
          <span>{{ selectedOrders.length }} 条</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pushDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="loading.agent" @click="pushToFactory">确认推送</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.filters {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.filter-panel {
  margin-bottom: 16px;
  padding: 16px;
  background: #f8f9fb;
  border-radius: 8px;
  border: 1px solid #ebeef5;
}
.filter-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.filter-row:last-child {
  margin-bottom: 0;
}
.filter-label {
  width: 72px;
  flex-shrink: 0;
  color: #606266;
  font-size: 13px;
}
.filter-summary {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding-top: 12px;
  margin-top: 4px;
  border-top: 1px dashed #dcdfe6;
}
.filter-tags {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.remark-line {
  line-height: 1.5;
  font-size: 12px;
}
.goods-line {
  line-height: 1.5;
}
.goods-line + .goods-line {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed #ebeef5;
}
.goods-row {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}
.goods-pic {
  width: 56px;
  height: 56px;
  flex-shrink: 0;
  border-radius: 4px;
  border: 1px solid #ebeef5;
}
.goods-info {
  flex: 1;
  min-width: 0;
}
.address {
  margin-top: 2px;
  line-height: 1.4;
}
.decrypted-line {
  line-height: 1.5;
  word-break: break-all;
}
.receiver-actions {
  margin-top: 4px;
}
.stats-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 12px;
}
.stat-item {
  min-width: 120px;
  padding: 8px 12px;
  border-radius: 8px;
  background: #fafafa;
  border: 1px solid transparent;
  transition: background 0.15s, border-color 0.15s, box-shadow 0.15s;
}
.stat-item.clickable {
  cursor: pointer;
}
.stat-item.clickable:hover {
  background: #f0f7ff;
  border-color: #c6e2ff;
}
.stat-item.active {
  background: #ecf5ff;
  border-color: #409eff;
  box-shadow: 0 0 0 1px #409eff inset;
}
.stat-item.active .stat-label {
  color: #409eff;
}
.stat-label {
  color: #606266;
  font-size: 13px;
}
.stat-value {
  font-size: 28px;
  font-weight: 600;
  line-height: 1.2;
  margin-top: 4px;
}
.stat-sub {
  margin-top: 4px;
}
.hint {
  margin-top: 8px;
}
</style>
