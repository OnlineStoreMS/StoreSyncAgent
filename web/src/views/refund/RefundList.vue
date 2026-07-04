<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useAccountRefresh } from '../../composables/useAccountRefresh'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  getRefundLogistics,
  listRefunds,
  type LogisticsDetail,
  type RefundItem,
  type RefundStats,
} from '../../api'
import { useKdzsStore } from '../../stores/kdzs'
import { dateShortcuts, defaultDateRange } from '../../utils/date'

const route = useRoute()
const kdzsStore = useKdzsStore()

const loading = reactive({ list: false, logistics: false })
const refunds = ref<RefundItem[]>([])
const refundTotal = ref(0)
const refundStats = ref<RefundStats>()
const logisticsVisible = ref(false)
const logisticsDetail = ref<LogisticsDetail>()
const logisticsTarget = ref<RefundItem>()

const [defaultStart, defaultEnd] = defaultDateRange()

const filters = reactive({
  platform: 'FXG',
  shopId: '',
  afterSaleStatus: '',
  afterSaleType: '',
  sid: '',
  tid: '',
  scenario: '',
  dateRange: [defaultStart, defaultEnd] as [string, string],
  pageNo: 1,
  pageSize: 20,
})

const scenarioTabs = [
  { key: '', label: '全部售后' },
  { key: 'confirm_receive', label: '待卖家确认收货' },
  { key: 'wait_agree', label: '等待卖家同意' },
  { key: 'wait_return', label: '待买家退货' },
  { key: 'refund_only', label: '仅退款提醒' },
  { key: 'exchange', label: '换货待处理' },
  { key: 'wait_send_exchange', label: '待发出换货商品' },
  { key: 'return_signed', label: '退回已签收' },
  { key: 'pickup_pending', label: '驿站待取件' },
  { key: 'seller_refuse', label: '卖家拒绝退款' },
  { key: 'refund_close_with_sid', label: '退款关闭(有物流)' },
  { key: 'refund_success', label: '退货退款·退款成功' },
  { key: 'urgent', label: '时效紧迫' },
]

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '等待卖家同意', value: 'WAIT_SELLER_AGREE' },
  { label: '等待买家退货', value: 'WAIT_BUYER_RETURN_ITEM' },
  { label: '待卖家确认收货', value: 'WAIT_SELLER_CONFIRM_RECEIVE' },
  { label: '待发出换货商品', value: 'WAIT_SEND_EXCHANGE_ITEM' },
  { label: '换货补寄待收货', value: 'WAIT_RECEIVE_EXCHANGE_ITEM' },
  { label: '卖家拒绝退款', value: 'SELLER_REFUSAL_REFUND' },
  { label: '退款成功', value: 'REFUND_SUCCESS' },
  { label: '售后关闭', value: 'REFUND_CLOSE' },
]

const typeOptions = [
  { label: '全部类型', value: '' },
  { label: '仅退款', value: '1' },
  { label: '退货退款', value: '2' },
  { label: '换货', value: '3' },
  { label: '补差价', value: '4' },
  { label: '补发', value: '5' },
]

const urgencyType = (urgency?: string) => {
  switch (urgency) {
    case 'expired':
    case 'imminent':
    case 'critical':
      return 'danger'
    case 'warning':
      return 'warning'
    case 'normal':
      return 'success'
    default:
      return 'info'
  }
}

const urgencyLabel = (urgency?: string) => {
  switch (urgency) {
    case 'expired':
      return '已超时'
    case 'imminent':
      return '极急'
    case 'critical':
      return '紧急'
    case 'warning':
      return '临近'
    case 'normal':
      return '正常'
    default:
      return '—'
  }
}

const statCards = computed(() => [
  { label: '待确认收货', value: refundStats.value?.waitSellerConfirmReceive, color: '#409eff', scenario: 'confirm_receive' },
  { label: '待卖家同意', value: refundStats.value?.waitSellerAgree, color: '#e6a23c', scenario: 'wait_agree' },
  { label: '仅退款待处理', value: refundStats.value?.refundOnlyPending, color: '#f56c6c', scenario: 'refund_only' },
  { label: '换货待处理', value: refundStats.value?.exchangePending, color: '#b88230', scenario: 'exchange' },
  { label: '待发出换货商品', value: refundStats.value?.waitSendExchange, color: '#9c6ade', scenario: 'wait_send_exchange' },
  { label: '退回已签收', value: refundStats.value?.returnSigned, color: '#67c23a', scenario: 'return_signed' },
  { label: '驿站待取件', value: refundStats.value?.pickupPending, color: '#909399', scenario: 'pickup_pending' },
  { label: '待买家退货', value: refundStats.value?.waitBuyerReturn, color: '#909399', scenario: 'wait_return' },
  { label: '卖家拒绝退款', value: refundStats.value?.sellerRefuse, color: '#e6a23c', scenario: 'seller_refuse' },
  { label: '退款关闭(有物流)', value: refundStats.value?.refundCloseWithSid, color: '#909399', scenario: 'refund_close_with_sid' },
  { label: '退货退款·退款成功', value: refundStats.value?.refundSuccess, color: '#67c23a', scenario: 'refund_success' },
  { label: '时效紧迫', value: refundStats.value?.urgent, color: '#f56c6c', highlight: true, scenario: 'urgent' },
])

async function loadRefunds() {
  loading.list = true
  try {
    const [startDateTime, endDateTime] = filters.dateRange
    const data = await listRefunds({
      platform: filters.platform || undefined,
      shopId: filters.shopId || undefined,
      afterSaleStatus: filters.afterSaleStatus || undefined,
      afterSaleType: filters.afterSaleType || undefined,
      sid: filters.sid.trim() || undefined,
      tid: filters.tid.trim() || undefined,
      scenario: filters.scenario || undefined,
      startDateTime: startDateTime || undefined,
      endDateTime: endDateTime || undefined,
      pageNo: filters.pageNo,
      pageSize: filters.pageSize,
      enrichLogistics: true,
    })
    refunds.value = data.items || []
    refundTotal.value = data.total || 0
    refundStats.value = data.stats
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '加载售后失败')
  } finally {
    loading.list = false
  }
}

function onScenarioChange(key: string) {
  filters.scenario = key
  filters.afterSaleStatus = ''
  filters.afterSaleType = ''
  filters.pageNo = 1
  loadRefunds()
}

function onStatCardClick(scenario: string) {
  onScenarioChange(scenario)
}

function onFilterChange() {
  filters.pageNo = 1
  loadRefunds()
}

function onPageChange(page: number) {
  filters.pageNo = page
  loadRefunds()
}

async function showLogistics(row: RefundItem) {
  if (!row.sid) {
    ElMessage.warning('无退货物流单号')
    return
  }
  logisticsTarget.value = row
  loading.logistics = true
  logisticsVisible.value = true
  try {
    logisticsDetail.value = await getRefundLogistics({
      platform: row.platform,
      sid: row.sid,
      sidCode: row.sidCode,
    })
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '查询物流失败')
  } finally {
    loading.logistics = false
  }
}

function firstGoods(row: RefundItem) {
  return row.goods?.[0]
}

const logisticsTraceList = computed(() => {
  const list = logisticsDetail.value?.traceList
  if (!list?.length) return []
  return [...list].reverse()
})

useAccountRefresh(async () => {
  filters.shopId = ''
  filters.pageNo = 1
  await loadRefunds()
})

onMounted(async () => {
  const scenario = route.query.scenario
  if (typeof scenario === 'string' && scenarioTabs.some((tab) => tab.key === scenario)) {
    filters.scenario = scenario
  }
  if (!kdzsStore.shops.length) {
    await kdzsStore.loadShops()
  }
  await loadRefunds()
})
</script>

<template>
  <div class="refund-page">
    <el-card shadow="never" class="page-card" v-if="refundStats">
      <div class="stats-row">
        <div
          v-for="card in statCards"
          :key="card.label"
          class="stat-item"
          :class="{
            highlight: card.highlight && (card.value || 0) > 0,
            active: filters.scenario === card.scenario,
            clickable: true,
          }"
          @click="onStatCardClick(card.scenario)"
        >
          <div class="stat-label">{{ card.label }}</div>
          <div class="stat-value" :style="{ color: card.color }">{{ card.value ?? '—' }}</div>
        </div>
      </div>
      <div class="stats-sub muted" v-if="refundStats.imminent || refundStats.critical || refundStats.expired">
        其中已超时 {{ refundStats.expired || 0 }} 条，30分钟内到期 {{ refundStats.imminent || 0 }} 条，4小时内到期 {{ refundStats.critical || 0 }} 条
      </div>
    </el-card>

    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">售后管理 <span class="count">({{ refundTotal }})</span></div>
          <el-button type="primary" :loading="loading.list" @click="loadRefunds">刷新</el-button>
        </div>
      </template>

      <div class="scenario-tabs">
        <el-tag
          v-for="tab in scenarioTabs"
          :key="tab.key"
          :type="filters.scenario === tab.key ? 'primary' : 'info'"
          :effect="filters.scenario === tab.key ? 'dark' : 'plain'"
          class="scenario-tag"
          @click="onScenarioChange(tab.key)"
        >
          {{ tab.label }}
        </el-tag>
      </div>

      <el-form inline class="filter-form" @submit.prevent="onFilterChange">
        <el-form-item label="平台">
          <el-select v-model="filters.platform" style="width: 120px" @change="onFilterChange">
            <el-option label="抖店" value="FXG" />
          </el-select>
        </el-form-item>
        <el-form-item label="店铺">
          <el-select v-model="filters.shopId" clearable placeholder="全部店铺" style="width: 180px" @change="onFilterChange">
            <el-option
              v-for="shop in kdzsStore.shops.filter((s) => s.platform === filters.platform)"
              :key="shop.mallUserId"
              :label="shop.mallUserName"
              :value="shop.mallUserId"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="售后类型">
          <el-select v-model="filters.afterSaleType" clearable style="width: 130px" @change="onFilterChange">
            <el-option v-for="opt in typeOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="平台状态">
          <el-select v-model="filters.afterSaleStatus" clearable style="width: 160px" @change="onFilterChange">
            <el-option v-for="opt in statusOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="申请时间">
          <el-date-picker
            v-model="filters.dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始"
            end-placeholder="结束"
            value-format="YYYY-MM-DD HH:mm:ss"
            :shortcuts="dateShortcuts"
            @change="onFilterChange"
          />
        </el-form-item>
        <el-form-item label="订单号">
          <el-input
            v-model="filters.tid"
            clearable
            placeholder="平台订单号（需配合申请时间）"
            style="width: 220px"
            @keyup.enter="onFilterChange"
          />
        </el-form-item>
        <el-form-item label="退货物流">
          <el-input
            v-model="filters.sid"
            clearable
            placeholder="快递单号（需配合申请时间）"
            style="width: 220px"
            @keyup.enter="onFilterChange"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="onFilterChange">查询</el-button>
        </el-form-item>
      </el-form>

      <el-table :data="refunds" v-loading="loading.list" stripe border class="refund-table">
        <el-table-column label="商品" min-width="220">
          <template #default="{ row }">
            <div class="goods-cell" v-if="firstGoods(row)">
              <el-image
                v-if="firstGoods(row)?.picUrl"
                :src="firstGoods(row)?.picUrl"
                fit="cover"
                class="goods-thumb"
                :preview-src-list="[firstGoods(row)!.picUrl!]"
                preview-teleported
              />
              <div class="goods-info">
                <div class="goods-title">{{ firstGoods(row)?.title }}</div>
                <div class="muted goods-sku">{{ firstGoods(row)?.skuName }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="售后信息" min-width="180">
          <template #default="{ row }">
            <div>
              <el-tag size="small" :type="row.afterSaleType === 1 ? 'danger' : 'info'">{{ row.afterSaleTypeText }}</el-tag>
              <el-tag size="small" class="ml-4">{{ row.afterSaleStatusText }}</el-tag>
            </div>
            <div class="muted mt-4">{{ row.refundReason }}</div>
            <div class="amount">¥{{ row.refundAmount }}</div>
          </template>
        </el-table-column>
        <el-table-column label="订单/买家" min-width="160">
          <template #default="{ row }">
            <div>{{ row.shopName }}</div>
            <div class="muted">{{ row.buyerNick }}</div>
            <div class="mono muted">{{ row.tid }}</div>
          </template>
        </el-table-column>
        <el-table-column label="申请时间" width="170">
          <template #default="{ row }">
            <div>{{ row.confirmTime || row.created }}</div>
          </template>
        </el-table-column>
        <el-table-column label="退货物流" min-width="150">
          <template #default="{ row }">
            <div v-if="row.sid">
              <el-link type="primary" @click="showLogistics(row)">{{ row.sid }}</el-link>
              <div class="muted">{{ row.sla?.logisticsStatusDesc }}</div>
              <div v-if="row.sla?.isPickupPending" class="pickup-tag">驿站/柜待取件</div>
            </div>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="处理时效" min-width="200">
          <template #default="{ row }">
            <template v-if="row.sla">
              <el-tag v-if="row.sla.important" type="danger" size="small" effect="dark" class="mb-4">仅退款</el-tag>
              <el-tag v-if="row.sla.isPickupPending && !row.sla.isSigned" type="warning" size="small" class="mb-4">待取件</el-tag>
              <div v-if="row.sla.remainingText" class="remaining" :class="row.sla.urgency">
                {{ row.sla.remainingText }}
              </div>
              <div v-if="row.sla.deadlineAt" class="muted deadline">截止 {{ row.sla.deadlineAt }}</div>
              <div v-if="row.sla.acceptTime" class="muted">揽收 {{ row.sla.acceptTime }}</div>
              <div v-if="row.sla.signTime" class="muted">签收 {{ row.sla.signTime }}</div>
              <div v-if="row.sla.pickupHint" class="hint-text pickup-hint">{{ row.sla.pickupHint }}</div>
              <div v-if="row.sla.hint" class="hint-text">{{ row.sla.hint }}</div>
              <el-tag v-if="row.sla.urgency" size="small" :type="urgencyType(row.sla.urgency)">
                {{ urgencyLabel(row.sla.urgency) }}
              </el-tag>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="refundTotal"
          :page-size="filters.pageSize"
          :current-page="filters.pageNo"
          @current-change="onPageChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="logisticsVisible" title="退货物流详情" width="640px">
      <div v-loading="loading.logistics">
        <template v-if="logisticsDetail">
          <div class="logistics-head">
            <div>{{ logisticsDetail.logisticsCompanyName }} · {{ logisticsDetail.mailNo }}</div>
            <div class="logistics-tags">
              <el-tag>{{ logisticsDetail.logisticsStatusDesc || logisticsDetail.logisticsStatus }}</el-tag>
              <el-tag v-if="logisticsDetail.isPickupPending" type="warning">识别为待取件</el-tag>
            </div>
          </div>
          <p v-if="logisticsDetail.isPickupPending && logisticsDetail.logisticsStatusDesc === '派件中'" class="pickup-note">
            快递助手显示「派件中」，轨迹含驿站/取件码等关键词，已按待取件处理。
          </p>
          <el-timeline class="trace-list">
            <el-timeline-item
              v-for="(trace, idx) in logisticsTraceList"
              :key="idx"
              :timestamp="trace.time"
            >
              {{ trace.desc }}
            </el-timeline-item>
          </el-timeline>
        </template>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.refund-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.page-card {
  border-radius: 8px;
}
.stats-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}
.stat-item {
  min-width: 110px;
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
.stat-item.highlight.active {
  background: #fef0f0;
  border-color: #f56c6c;
  box-shadow: 0 0 0 1px #f56c6c inset;
}
.stat-item.highlight.active .stat-label {
  color: #f56c6c;
}
.stat-item.highlight {
  background: #fef0f0;
  border: 1px solid #fbc4c4;
}
.stat-label {
  font-size: 12px;
  color: #909399;
}
.stat-value {
  font-size: 22px;
  font-weight: 600;
  margin-top: 4px;
}
.stats-sub {
  margin-top: 8px;
  font-size: 12px;
}
.row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.card-title {
  font-weight: 600;
}
.count {
  color: #909399;
  font-weight: 400;
}
.scenario-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}
.scenario-tag {
  cursor: pointer;
}
.filter-form {
  margin-bottom: 12px;
}
.goods-cell {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}
.goods-thumb {
  width: 48px;
  height: 48px;
  border-radius: 4px;
  flex-shrink: 0;
}
.goods-title {
  font-size: 13px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.goods-sku {
  font-size: 12px;
}
.muted {
  color: #909399;
  font-size: 12px;
}
.mono {
  font-family: ui-monospace, monospace;
}
.mt-4 {
  margin-top: 4px;
}
.ml-4 {
  margin-left: 4px;
}
.mb-4 {
  margin-bottom: 4px;
}
.amount {
  color: #f56c6c;
  font-weight: 600;
  margin-top: 4px;
}
.remaining {
  font-weight: 600;
  font-size: 14px;
}
.remaining.expired,
.remaining.imminent,
.remaining.critical {
  color: #f56c6c;
}
.remaining.warning {
  color: #e6a23c;
}
.remaining.normal {
  color: #67c23a;
}
.deadline,
.hint-text {
  font-size: 12px;
  margin: 2px 0;
}
.hint-text {
  color: #606266;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.logistics-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  gap: 12px;
}
.logistics-tags {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.pickup-tag {
  color: #e6a23c;
  font-size: 12px;
  margin-top: 2px;
}
.pickup-hint {
  color: #e6a23c;
}
.pickup-note {
  font-size: 12px;
  color: #909399;
  margin: 0 0 12px;
}
.trace-list {
  max-height: 400px;
  overflow-y: auto;
}
</style>
