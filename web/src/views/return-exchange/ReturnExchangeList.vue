<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Search } from '@element-plus/icons-vue'
import {
  createReturnExchange,
  deleteReturnExchange,
  listReturnExchanges,
  lookupOrder,
  updateReturnExchange,
  type OrderLookup,
  type ReturnExchangeRecord,
} from '../../api'

const loading = reactive({ list: false, lookup: false, save: false })
const rows = ref<ReturnExchangeRecord[]>([])
const lookupRowId = ref<string>()
const afterSaleTypes = ['补发', '换货', '退货退款', '仅退款', '其他']

async function loadList() {
  loading.list = true
  try {
    const data = await listReturnExchanges()
    rows.value = data.items || []
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '加载失败')
  } finally {
    loading.list = false
  }
}

function blankRow(): ReturnExchangeRecord {
  return {
    id: '',
    seqNo: 0,
    buyerNick: '',
    afterSaleType: '补发',
    returnTrackingNo: '',
    spec: '',
    feedbackTime: '',
    submitTime: '',
    orderNo: '',
    recipientInfo: '',
    outboundTrackingNo: '',
    remark: '',
    shopName: '',
    orderBuyerNick: '',
    goodsSummary: '',
    memoNotes: '',
    originalRecipientInfo: '',
    payment: 0,
    payTime: '',
    statusText: '',
  }
}

function formatMoney(v?: number) {
  if (!v || v <= 0) return ''
  return v.toFixed(2)
}

async function addRow() {
  loading.save = true
  try {
    const item = await createReturnExchange(blankRow())
    rows.value.push(item)
    ElMessage.success('已新增一行')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '新增失败')
  } finally {
    loading.save = false
  }
}

async function saveRow(row: ReturnExchangeRecord) {
  if (!row.id) return
  loading.save = true
  try {
    const updated = await updateReturnExchange(row.id, row)
    const idx = rows.value.findIndex((r) => r.id === row.id)
    if (idx >= 0) rows.value[idx] = updated
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '保存失败')
  } finally {
    loading.save = false
  }
}

async function removeRow(row: ReturnExchangeRecord) {
  if (!row.id) return
  try {
    await ElMessageBox.confirm('确定删除这条记录？', '删除确认', { type: 'warning' })
    await deleteReturnExchange(row.id)
    rows.value = rows.value.filter((r) => r.id !== row.id)
    ElMessage.success('已删除')
  } catch (e: any) {
    if (e === 'cancel' || e?.message === 'cancel') return
    ElMessage.error(e?.response?.data?.error || e.message || '删除失败')
  }
}

function applyLookup(row: ReturnExchangeRecord, data: OrderLookup) {
  if (!data.found) {
    ElMessage.warning('未找到该订单，请检查订单号或平台')
    return
  }
  // 仅填充订单侧字段；客户昵称、新收件地址保持手填不动
  row.shopName = data.shopName || ''
  row.orderBuyerNick = data.orderBuyerNick || ''
  row.goodsSummary = data.goodsSummary || ''
  row.goodsTitle = data.goodsTitle || ''
  row.memoNotes = data.memoNotes || ''
  row.originalRecipientInfo = data.originalRecipientInfo || ''
  row.payment = data.payment || 0
  row.payTime = data.payTime || ''
  row.statusText = data.statusText || ''
  row.platform = data.platform || row.platform
  row.sysTid = data.sysTid || ''
  ElMessage.success('已补充订单信息（店铺/买家/商品/原收件等）')
}

async function onOrderNoBlur(row: ReturnExchangeRecord) {
  const orderNo = row.orderNo?.trim()
  if (!orderNo || orderNo.length < 10) return
  lookupRowId.value = row.id
  loading.lookup = true
  try {
    const data = await lookupOrder({ orderNo, platform: row.platform || 'FXG' })
    applyLookup(row, data)
    await saveRow(row)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '检索订单失败')
  } finally {
    loading.lookup = false
    lookupRowId.value = undefined
  }
}

async function manualLookup(row: ReturnExchangeRecord) {
  if (!row.orderNo?.trim()) {
    ElMessage.warning('请先填写订单号')
    return
  }
  await onOrderNoBlur(row)
}

onMounted(loadList)
</script>

<template>
  <div class="page">
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar">
        <div>
          <div class="title">退换货管理</div>
          <div class="desc">
            填写订单号后自动补充店铺、买家、商品、留言、原收件信息、金额、付款时间、状态；
            <strong>客户昵称</strong>与<strong>新收件地址</strong>请手动填写
          </div>
        </div>
        <div class="actions">
          <el-button :icon="Refresh" :loading="loading.list" @click="loadList">刷新</el-button>
          <el-button type="primary" :icon="Plus" :loading="loading.save" @click="addRow">新增一行</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="table-card">
      <el-table
        v-loading="loading.list"
        :data="rows"
        border
        stripe
        empty-text="暂无记录，点击「新增一行」开始维护"
        style="width: 100%"
        :header-cell-style="{ background: '#fafafa' }"
      >
        <el-table-column label="序号" width="64" fixed>
          <template #default="{ row }">
            <el-input-number v-model="row.seqNo" :min="0" :controls="false" size="small" @change="saveRow(row)" />
          </template>
        </el-table-column>

        <el-table-column label="订单号" width="190" fixed>
          <template #default="{ row }">
            <div class="order-cell">
              <el-input
                v-model="row.orderNo"
                size="small"
                placeholder="输入后失焦检索"
                @blur="onOrderNoBlur(row)"
              />
              <el-button
                link
                type="primary"
                :icon="Search"
                :loading="loading.lookup && lookupRowId === row.id"
                @click="manualLookup(row)"
              />
            </div>
          </template>
        </el-table-column>

        <el-table-column label="店铺" width="120">
          <template #default="{ row }">
            <span class="auto-field">{{ row.shopName || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="买家" width="110">
          <template #default="{ row }">
            <span class="auto-field">{{ row.orderBuyerNick || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="商品" min-width="180">
          <template #default="{ row }">
            <span class="auto-field multiline">{{ row.goodsSummary || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="留言备注" min-width="140">
          <template #default="{ row }">
            <span class="auto-field multiline">{{ row.memoNotes || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="原收件信息" min-width="200">
          <template #default="{ row }">
            <span class="auto-field multiline">{{ row.originalRecipientInfo || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="金额" width="80" align="right">
          <template #default="{ row }">
            <span class="auto-field">{{ formatMoney(row.payment) || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="付款时间" width="150">
          <template #default="{ row }">
            <span class="auto-field">{{ row.payTime || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="auto-field">{{ row.statusText || '—' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="客户昵称" width="120">
          <template #header>
            <span>客户昵称</span>
            <span class="manual-tag">手填</span>
          </template>
          <template #default="{ row }">
            <el-input v-model="row.buyerNick" size="small" placeholder="手动填写" @blur="saveRow(row)" />
          </template>
        </el-table-column>
        <el-table-column label="售后类型" width="100">
          <template #default="{ row }">
            <el-select v-model="row.afterSaleType" size="small" filterable allow-create @change="saveRow(row)">
              <el-option v-for="t in afterSaleTypes" :key="t" :label="t" :value="t" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="退回单号" width="130">
          <template #default="{ row }">
            <el-input v-model="row.returnTrackingNo" size="small" placeholder="补发可不填" @blur="saveRow(row)" />
          </template>
        </el-table-column>
        <el-table-column label="规格" min-width="140">
          <template #default="{ row }">
            <el-input v-model="row.spec" size="small" type="textarea" :autosize="{ minRows: 1, maxRows: 3 }" @blur="saveRow(row)" />
          </template>
        </el-table-column>
        <el-table-column label="顾客反馈时间" width="120">
          <template #default="{ row }">
            <el-input v-model="row.feedbackTime" size="small" placeholder="如 2026.7.1" @blur="saveRow(row)" />
          </template>
        </el-table-column>
        <el-table-column label="提交时间" width="120">
          <template #default="{ row }">
            <el-input v-model="row.submitTime" size="small" @blur="saveRow(row)" />
          </template>
        </el-table-column>
        <el-table-column label="新收件地址" min-width="200">
          <template #header>
            <span>新收件地址</span>
            <span class="manual-tag">手填</span>
          </template>
          <template #default="{ row }">
            <el-input
              v-model="row.recipientInfo"
              size="small"
              type="textarea"
              placeholder="顾客提供的新地址"
              :autosize="{ minRows: 2, maxRows: 6 }"
              @blur="saveRow(row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="发出快递单号" width="130">
          <template #default="{ row }">
            <el-input v-model="row.outboundTrackingNo" size="small" @blur="saveRow(row)" />
          </template>
        </el-table-column>
        <el-table-column label="备注" min-width="100">
          <template #default="{ row }">
            <el-input v-model="row.remark" size="small" @blur="saveRow(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="72" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" @click="removeRow(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.toolbar-card :deep(.el-card__body) {
  padding: 16px 20px;
}
.toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
.title {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}
.desc {
  margin-top: 4px;
  font-size: 13px;
  color: #909399;
  line-height: 1.5;
}
.desc strong {
  color: #606266;
  font-weight: 600;
}
.actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.table-card :deep(.el-card__body) {
  padding: 0;
}
.order-cell {
  display: flex;
  align-items: center;
  gap: 4px;
}
.order-cell .el-input {
  flex: 1;
}
.auto-field {
  font-size: 12px;
  color: #606266;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-word;
}
.auto-field.multiline {
  display: block;
  max-height: 120px;
  overflow: auto;
}
.manual-tag {
  display: inline-block;
  margin-left: 4px;
  padding: 0 4px;
  font-size: 10px;
  color: #e6a23c;
  background: #fdf6ec;
  border-radius: 3px;
  vertical-align: middle;
}
</style>
