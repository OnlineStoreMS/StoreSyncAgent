<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Bell, Refresh, Warning } from '@element-plus/icons-vue'
import {
  getStockAlert,
  resetStockAlertState,
  runStockAlert,
  saveStockAlert,
  scanStockAlert,
  testStockAlert,
  type Shop,
  type StockAlertConfig,
  type StockAlertHit,
  type StockAlertState,
} from '../../api'

const loading = reactive({ load: false, save: false, test: false, run: false, scan: false, reset: false })
const shops = ref<Shop[]>([])
const state = ref<StockAlertState>({})
const secretInput = ref('')
const alertItems = ref<StockAlertHit[]>([])
const scanMeta = reactive({ threshold: 0, total: 0, scanned: 0 })

const form = reactive<StockAlertConfig>({
  enabled: false,
  webhookUrl: '',
  platform: 'FXG',
  shopIds: [],
  stockThreshold: 10,
  checkLevel: 'sku',
  onlyOnsale: true,
  pollIntervalMinutes: 60,
})

const platformOptions = [
  { label: '抖店', value: 'FXG' },
  { label: '淘宝', value: 'TB' },
  { label: '小红书', value: 'XHS' },
]

const checkLevelOptions = [
  { label: '按 SKU 库存', value: 'sku' },
  { label: '按整款库存', value: 'spu' },
  { label: 'SKU + 整款', value: 'both' },
]

const shopOptions = computed(() =>
  shops.value.filter((s) => !form.platform || s.platform === form.platform),
)

function applyView(data: { config: StockAlertConfig; state: StockAlertState; shops?: Shop[] }) {
  Object.assign(form, data.config)
  form.shopIds = [...(data.config.shopIds || [])]
  form.checkLevel = data.config.checkLevel || 'sku'
  state.value = data.state || {}
  if (data.shops) shops.value = data.shops
  secretInput.value = ''
}

async function load() {
  loading.load = true
  try {
    const data = await getStockAlert()
    applyView(data)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '加载失败')
  } finally {
    loading.load = false
  }
}

async function onSave() {
  loading.save = true
  try {
    const payload: StockAlertConfig = {
      ...form,
      secret: secretInput.value || undefined,
      shopIds: [...form.shopIds],
    }
    const data = await saveStockAlert(payload)
    applyView(data)
    ElMessage.success('已保存')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '保存失败')
  } finally {
    loading.save = false
  }
}

async function onTest() {
  loading.test = true
  try {
    await testStockAlert('【电商店铺同步】线上商品库存预警测试')
    ElMessage.success('测试消息已发送')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '测试失败')
  } finally {
    loading.test = false
  }
}

async function onScan() {
  loading.scan = true
  try {
    const data = await scanStockAlert()
    alertItems.value = data.items || []
    scanMeta.threshold = data.threshold
    scanMeta.total = data.total
    scanMeta.scanned = data.scanned
    ElMessage.success(`扫描完成：共检查 ${data.scanned} 款，低库存 ${data.total} 条`)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '扫描失败')
  } finally {
    loading.scan = false
  }
}

async function onRunNow() {
  loading.run = true
  try {
    const result = await runStockAlert()
    ElMessage.success(
      `检查完成：推送 ${result.sent} 条，跳过 ${result.skipped} 条，当前低库存 ${result.alerted} 条（扫描 ${result.scanned} 款）`,
    )
    await load()
    await onScan()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '执行失败')
  } finally {
    loading.run = false
  }
}

async function onResetState() {
  try {
    await ElMessageBox.confirm(
      '将清空已推送去重记录与运行状态，之后「立即推送」或定时任务会对仍低于阈值的商品重新推送。预警配置不会改动。',
      '重置预警记录',
      { type: 'warning', confirmButtonText: '确认重置', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  loading.reset = true
  try {
    const data = await resetStockAlertState()
    if (data.view) applyView(data.view)
    ElMessage.success(`已重置，清除 ${data.cleared} 条去重记录`)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '重置失败')
  } finally {
    loading.reset = false
  }
}

function onPlatformChange() {
  form.shopIds = []
}

onMounted(load)
</script>

<template>
  <div class="stock-alert-page" v-loading="loading.load">
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">
            <el-icon class="title-icon"><Warning /></el-icon>
            线上商品库存预警
          </div>
          <div class="actions">
            <el-button :icon="Refresh" @click="load">刷新</el-button>
            <el-button :loading="loading.test" @click="onTest">测试推送</el-button>
            <el-button type="warning" plain :loading="loading.scan" @click="onScan">扫描低库存</el-button>
            <el-button type="warning" :loading="loading.run" @click="onRunNow">立即推送</el-button>
            <el-button type="primary" :loading="loading.save" @click="onSave">保存配置</el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        class="hint"
        title="按快递助手同步的线上商品库存做阈值预警。库存来自平台实时数据（只读），低于阈值时通过飞书群机器人推送；同一 SKU 去重，库存回升后再次跌破会重新通知。"
      />

      <el-form label-width="120px" class="form">
        <el-form-item label="启用预警">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="Webhook 地址" required>
          <el-input v-model="form.webhookUrl" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..." />
        </el-form-item>
        <el-form-item label="签名校验">
          <el-input
            v-model="secretInput"
            type="password"
            show-password
            :placeholder="form.secretSet ? '已配置，留空则不修改' : '机器人安全设置中的签名校验密钥'"
          />
        </el-form-item>
        <el-form-item label="平台">
          <el-select v-model="form.platform" style="width: 160px" @change="onPlatformChange">
            <el-option v-for="opt in platformOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="店铺范围">
          <el-select
            v-model="form.shopIds"
            multiple
            clearable
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="全部店铺"
            style="width: 420px"
          >
            <el-option
              v-for="shop in shopOptions"
              :key="shop.mallUserId"
              :label="shop.mallUserName"
              :value="shop.mallUserId"
            />
          </el-select>
          <div class="field-tip muted">留空表示当前平台下全部已授权店铺</div>
        </el-form-item>
        <el-form-item label="库存阈值">
          <div class="inline-field">
            ≤
            <el-input-number v-model="form.stockThreshold" :min="0" :max="999999" />
            件时预警
          </div>
        </el-form-item>
        <el-form-item label="检查维度">
          <el-radio-group v-model="form.checkLevel">
            <el-radio-button v-for="opt in checkLevelOptions" :key="opt.value" :label="opt.value">
              {{ opt.label }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="仅上架商品">
          <el-switch v-model="form.onlyOnsale" />
        </el-form-item>
        <el-form-item label="定时扫描">
          <div class="inline-field">
            每
            <el-input-number v-model="form.pollIntervalMinutes" :min="15" :max="1440" :step="15" />
            分钟检查一次（最小 15 分钟）
          </div>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="page-card status-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">
            <el-icon class="title-icon"><Bell /></el-icon>
            运行状态
          </div>
          <el-button type="danger" plain :loading="loading.reset" @click="onResetState">重置去重记录</el-button>
        </div>
      </template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="上次运行">{{ state.lastRunAt || '-' }}</el-descriptions-item>
        <el-descriptions-item label="运行结果">
          <el-tag v-if="state.lastRunAt" :type="state.lastRunOk ? 'success' : 'danger'" size="small">
            {{ state.lastRunOk ? '成功' : '失败' }}
          </el-tag>
          <span v-else>-</span>
        </el-descriptions-item>
        <el-descriptions-item label="上次推送">{{ state.lastSentCount ?? 0 }} 条消息</el-descriptions-item>
        <el-descriptions-item label="当时低库存">{{ state.lastAlertCount ?? 0 }} 条</el-descriptions-item>
        <el-descriptions-item label="错误信息" :span="2">
          <span :class="{ danger: !!state.lastError }">{{ state.lastError || '-' }}</span>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">
            低库存列表
            <span v-if="scanMeta.total" class="count">({{ scanMeta.total }})</span>
          </div>
          <span v-if="scanMeta.scanned" class="muted scan-meta">
            阈值 ≤ {{ scanMeta.threshold }} · 已扫描 {{ scanMeta.scanned }} 款
          </span>
        </div>
      </template>
      <el-table :data="alertItems" stripe border empty-text="点击「扫描低库存」查看当前低于阈值的商品" max-height="520">
        <el-table-column label="图片" width="72">
          <template #default="{ row }">
            <el-image
              v-if="row.picUrl"
              :src="row.picUrl"
              fit="cover"
              class="thumb"
              referrerpolicy="no-referrer"
              :preview-src-list="[row.picUrl]"
              preview-teleported
            />
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="商品" min-width="200" show-overflow-tooltip />
        <el-table-column label="规格" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.propertiesName || (row.level === 'spu' ? '整款' : '-') }}
          </template>
        </el-table-column>
        <el-table-column prop="shopName" label="店铺" min-width="120" show-overflow-tooltip />
        <el-table-column prop="quantity" label="库存" width="80">
          <template #default="{ row }">
            <span class="danger">{{ row.quantity }}</span>
          </template>
        </el-table-column>
        <el-table-column label="维度" width="80">
          <template #default="{ row }">
            {{ row.level === 'sku' ? 'SKU' : '整款' }}
          </template>
        </el-table-column>
        <el-table-column prop="itemId" label="商品ID" min-width="130" show-overflow-tooltip />
        <el-table-column prop="skuId" label="SKU ID" min-width="130" show-overflow-tooltip />
        <el-table-column prop="approveStatusLabel" label="状态" width="80" />
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.stock-alert-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.hint {
  margin-bottom: 16px;
}
.form {
  max-width: 860px;
}
.inline-field {
  display: flex;
  align-items: center;
  gap: 8px;
}
.field-tip {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.4;
}
.title-icon {
  margin-right: 6px;
  vertical-align: middle;
}
.count {
  color: #909399;
  font-weight: 400;
  margin-left: 4px;
}
.scan-meta {
  font-size: 13px;
}
.thumb {
  width: 48px;
  height: 48px;
  border-radius: 4px;
}
.muted {
  color: #909399;
}
.danger {
  color: #f56c6c;
}
</style>
