<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { List, Shop, Connection, WarningFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getRefundStats, type RefundStats } from '../api'
import { useKdzsStore } from '../stores/kdzs'
import { defaultDateRange } from '../utils/date'
import { formatAccountTitle } from '../utils/account'
import { useAccountRefresh } from '../composables/useAccountRefresh'

const router = useRouter()
const kdzsStore = useKdzsStore()

const refundStats = ref<RefundStats>()
const loadingRefunds = ref(false)

const shopCount = computed(() => kdzsStore.shops.length)
const validShopCount = computed(() => kdzsStore.shops.filter((s) => s.tokenValid).length)

const refundScenarios = computed(() => [
  {
    key: 'confirm_receive',
    label: '待确认收货',
    value: refundStats.value?.waitSellerConfirmReceive,
    color: '#409eff',
    desc: '买家已退货，待确认',
  },
  {
    key: 'return_signed',
    label: '退回已签收',
    value: refundStats.value?.returnSigned,
    color: '#67c23a',
    desc: '签收后 48h 内处理',
    highlight: true,
  },
  {
    key: 'pickup_pending',
    label: '驿站待取件',
    value: refundStats.value?.pickupPending,
    color: '#909399',
    desc: '揽收后 7 天内关注',
  },
  {
    key: 'wait_agree',
    label: '待卖家同意',
    value: refundStats.value?.waitSellerAgree,
    color: '#e6a23c',
    desc: '新申请待审核',
  },
  {
    key: 'refund_only',
    label: '仅退款提醒',
    value: refundStats.value?.refundOnlyPending,
    color: '#f56c6c',
    desc: '建议36h内处理（预留12h缓冲）',
    highlight: true,
  },
  {
    key: 'exchange',
    label: '换货待处理',
    value: refundStats.value?.exchangePending,
    color: '#b88230',
    desc: '换货进行中',
  },
  {
    key: 'wait_send_exchange',
    label: '待发出换货商品',
    value: refundStats.value?.waitSendExchange,
    color: '#9c6ade',
    desc: '需发出换货商品',
  },
  {
    key: 'urgent',
    label: '时效紧迫',
    value: refundStats.value?.urgent,
    color: '#f56c6c',
    desc: '即将或已超时',
    highlight: true,
  },
])

const pendingRefundTotal = computed(() => {
  if (!refundStats.value) return 0
  const s = refundStats.value
  return (
    (s.waitSellerConfirmReceive || 0) +
    (s.waitSellerAgree || 0) +
    (s.refundOnlyPending || 0) +
    (s.exchangePending || 0)
  )
})

async function loadRefundStats() {
  loadingRefunds.value = true
  try {
    const [startDateTime, endDateTime] = defaultDateRange()
    refundStats.value = await getRefundStats({
      platform: 'FXG',
      startDateTime,
      endDateTime,
    })
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '加载售后提醒失败')
  } finally {
    loadingRefunds.value = false
  }
}

function goRefundScenario(scenario: string) {
  router.push({ path: '/refunds', query: scenario ? { scenario } : undefined })
}

async function onSwitchAccount(accountId: string) {
  await kdzsStore.switchKdzsAccount(accountId)
}

useAccountRefresh(loadRefundStats)

onMounted(async () => {
  if (!kdzsStore.accounts.length) {
    await kdzsStore.loadAccounts()
  }
  await kdzsStore.refreshAll()
  await loadRefundStats()
})
</script>

<template>
  <div class="dashboard">
    <el-card
      v-if="kdzsStore.accounts.length"
      shadow="never"
      class="account-switch-card"
      v-loading="kdzsStore.loading.switch || kdzsStore.loading.accounts"
    >
      <div class="account-switch-header">
        <div>
          <div class="account-switch-title">快递助手账号</div>
          <div class="account-switch-sub muted">
            当前：{{ formatAccountTitle({
              name: kdzsStore.loginInfo.accountName,
              mobile: kdzsStore.loginInfo.mobile,
              roleLabel: kdzsStore.loginInfo.accountRoleLabel,
            }) || '未连接' }}
            · {{ kdzsStore.loginInfo.loggedIn ? '已登录' : '未连接' }}
          </div>
        </div>
        <span class="muted account-switch-tip">点击切换账号，数据将重新加载</span>
      </div>
      <div class="account-switch-list">
        <div
          v-for="acc in kdzsStore.accounts"
          :key="acc.id"
          class="account-chip"
          :class="{ active: acc.active }"
          @click="onSwitchAccount(acc.id)"
        >
          <div class="account-chip-top">
            <span class="account-chip-name" :title="formatAccountTitle(acc)">{{ formatAccountTitle(acc) }}</span>
            <el-tag v-if="acc.active" class="account-chip-tag" type="success" size="small" effect="plain">当前</el-tag>
          </div>
        </div>
      </div>
    </el-card>

    <el-row :gutter="16" class="summary-row">
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card" v-loading="kdzsStore.loading.status">
          <div class="stat-inner">
            <el-icon :size="28" color="#409eff"><Connection /></el-icon>
            <div>
              <div class="stat-label">快递助手连接</div>
              <div class="stat-value">{{ kdzsStore.loginInfo.loggedIn ? '已登录' : '未连接' }}</div>
              <div class="stat-sub muted">{{ kdzsStore.loginInfo.mobile || '—' }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card" v-loading="kdzsStore.loading.shops">
          <div class="stat-inner">
            <el-icon :size="28" color="#67c23a"><Shop /></el-icon>
            <div>
              <div class="stat-label">绑定店铺</div>
              <div class="stat-value">{{ shopCount }}</div>
              <div class="stat-sub muted">授权有效 {{ validShopCount }} 家</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card" v-loading="kdzsStore.loading.overview">
          <div class="stat-inner">
            <el-icon :size="28" color="#e6a23c"><List /></el-icon>
            <div>
              <div class="stat-label">待推单</div>
              <div class="stat-value">{{ kdzsStore.stats?.waitingPushTotal ?? '—' }}</div>
              <div class="stat-sub muted" v-if="kdzsStore.stats?.waitingPushByPlatform?.FXG">
                抖店 {{ kdzsStore.stats.waitingPushByPlatform.FXG }}
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card" v-loading="kdzsStore.loading.overview">
          <div class="stat-inner">
            <el-icon :size="28" color="#f56c6c"><List /></el-icon>
            <div>
              <div class="stat-label">待发货</div>
              <div class="stat-value">{{ kdzsStore.stats?.waitingSendTotal ?? '—' }}</div>
              <div class="stat-sub muted">快递助手首页统计</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="refund-reminder-card" v-loading="loadingRefunds">
      <template #header>
        <div class="refund-header">
          <div>
            <div class="refund-title">
              <el-icon color="#f56c6c"><WarningFilled /></el-icon>
              售后提醒
            </div>
            <div class="refund-sub muted">
              近 30 天申请 · 待处理 {{ pendingRefundTotal }} 单
              <template v-if="refundStats && (refundStats.expired || refundStats.imminent || refundStats.critical)">
                · 已超时 {{ refundStats.expired || 0 }} · 30m 内 {{ refundStats.imminent || 0 }} · 4h 内 {{ refundStats.critical || 0 }}
              </template>
            </div>
          </div>
          <el-button type="primary" link @click="goRefundScenario('')">查看全部售后</el-button>
        </div>
      </template>

      <div class="refund-scenarios">
        <div
          v-for="item in refundScenarios"
          :key="item.key"
          class="refund-scenario"
          :class="{ highlight: item.highlight && (item.value || 0) > 0, empty: !item.value }"
          @click="goRefundScenario(item.key)"
        >
          <div class="refund-scenario-label">{{ item.label }}</div>
          <div class="refund-scenario-value" :style="{ color: item.value ? item.color : '#c0c4cc' }">
            {{ item.value ?? '—' }}
          </div>
          <div class="refund-scenario-desc muted">{{ item.desc }}</div>
        </div>
      </div>
    </el-card>

    <el-row :gutter="16">
      <el-col :xs="24" :md="8">
        <el-card shadow="hover" class="nav-card" @click="router.push('/shops')">
          <div class="nav-inner">
            <el-icon :size="32" color="#67c23a"><Shop /></el-icon>
            <div>
              <div class="nav-title">店铺管理</div>
              <div class="nav-desc">查看抖店、淘宝等平台绑定店铺及授权状态</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="8">
        <el-card shadow="hover" class="nav-card" @click="router.push('/orders')">
          <div class="nav-inner">
            <el-icon :size="32" color="#409eff"><List /></el-icon>
            <div>
              <div class="nav-title">订单列表</div>
              <div class="nav-desc">按平台、店铺、时间筛选订单，支持一键解密收件信息</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="8">
        <el-card shadow="hover" class="nav-card" @click="goRefundScenario('urgent')">
          <div class="nav-inner">
            <el-icon :size="32" color="#f56c6c"><WarningFilled /></el-icon>
            <div>
              <div class="nav-title">售后管理</div>
              <div class="nav-desc">按场景筛选售后单，查看物流轨迹与处理时效提醒</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <el-col :xs="24">
        <el-card shadow="never" class="info-card">
          <div class="info-title">说明</div>
          <ul class="info-list">
            <li>数据来自快递助手，按需实时拉取，不持久化到本地</li>
            <li>抖店订单解密返回虚拟号（主机号-分机号），用于发货联系</li>
            <li>默认查询近 30 天、待推单状态的抖店订单</li>
          </ul>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.summary-row {
  margin-bottom: 16px;
}
.account-switch-card {
  margin-bottom: 16px;
  border-radius: 8px;
}
.account-switch-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.account-switch-title {
  font-size: 16px;
  font-weight: 600;
}
.account-switch-sub {
  margin-top: 4px;
  font-size: 12px;
}
.account-switch-tip {
  font-size: 12px;
  white-space: nowrap;
}
.account-switch-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.account-chip {
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid #ebeef5;
  background: #fafafa;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s, border-color 0.15s;
  min-width: 0;
}
.account-chip:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
}
.account-chip.active {
  border-color: #b3e19d;
  background: #f0f9eb;
}
.account-chip-top {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.account-chip-name {
  flex: 1;
  min-width: 0;
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.account-chip-tag {
  flex-shrink: 0;
}
.refund-reminder-card {
  margin-bottom: 16px;
  border-radius: 8px;
}
.refund-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.refund-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
}
.refund-sub {
  margin-top: 4px;
  font-size: 12px;
}
.refund-scenarios {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}
.refund-scenario {
  padding: 12px;
  border-radius: 8px;
  background: #fafafa;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s, background 0.15s;
}
.refund-scenario:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
}
.refund-scenario.highlight {
  background: #fef0f0;
  border: 1px solid #fbc4c4;
}
.refund-scenario.empty {
  opacity: 0.72;
}
.refund-scenario-label {
  font-size: 12px;
  color: #909399;
}
.refund-scenario-value {
  font-size: 26px;
  font-weight: 600;
  line-height: 1.2;
  margin: 6px 0 4px;
}
.refund-scenario-desc {
  font-size: 11px;
  line-height: 1.4;
}
.stat-card,
.nav-card,
.info-card {
  margin-bottom: 16px;
}
.stat-inner,
.nav-inner {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.stat-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 4px;
}
.stat-value {
  font-size: 24px;
  font-weight: 600;
  line-height: 1.2;
}
.stat-sub {
  margin-top: 4px;
}
.nav-card {
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.15s;
}
.nav-card:hover {
  transform: translateY(-2px);
}
.nav-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}
.nav-desc {
  font-size: 13px;
  color: #909399;
  line-height: 1.5;
}
.info-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
}
.info-list {
  margin: 0;
  padding-left: 18px;
  color: #606266;
  font-size: 13px;
  line-height: 1.8;
}
</style>
