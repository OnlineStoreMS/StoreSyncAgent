import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getLoginStatus,
  listAccounts,
  listFactories,
  listOrders,
  listShops,
  switchAccount,
  type FactoryItem,
  type KdzsAccount,
  type OrderListResponse,
  type Shop,
} from '../api'
import { defaultDateRange } from '../utils/date'
import { formatAccountTitle } from '../utils/account'

export const useKdzsStore = defineStore('kdzs', () => {
  const loginInfo = ref<{
    userId?: string
    mobile?: string
    loggedIn?: boolean
    accountId?: string
    accountName?: string
    accountRole?: string
    accountRoleLabel?: string
  }>({})
  const accounts = ref<KdzsAccount[]>([])
  const shops = ref<Shop[]>([])
  const factories = ref<FactoryItem[]>([])
  const stats = ref<OrderListResponse['stats']>()
  /** 账号切换成功后递增，供各数据页监听并刷新。 */
  const accountVersion = ref(0)
  const loading = ref({ status: false, shops: false, overview: false, accounts: false, factories: false, switch: false })

  async function loadStatus() {
    loading.value.status = true
    try {
      loginInfo.value = await getLoginStatus()
    } finally {
      loading.value.status = false
    }
  }

  async function loadAccounts() {
    loading.value.accounts = true
    try {
      const data = await listAccounts()
      accounts.value = data.items || []
    } finally {
      loading.value.accounts = false
    }
  }

  async function switchKdzsAccount(accountId: string) {
    if (accountId === loginInfo.value.accountId) return
    loading.value.switch = true
    try {
      loginInfo.value = await switchAccount(accountId)
      shops.value = []
      factories.value = []
      stats.value = undefined
      await Promise.all([loadAccounts(), loadShops(), loadOverviewStats()])
      accountVersion.value++
      ElMessage.success(`已切换到 ${formatAccountTitle({
        name: loginInfo.value.accountName,
        mobile: loginInfo.value.mobile,
        roleLabel: loginInfo.value.accountRoleLabel,
      })}`)
    } catch (e: any) {
      ElMessage.error(e?.response?.data?.error || e.message || '切换账号失败')
      throw e
    } finally {
      loading.value.switch = false
    }
  }

  async function loadShops() {
    loading.value.shops = true
    try {
      const data = await listShops()
      shops.value = data.items || []
    } finally {
      loading.value.shops = false
    }
  }

  async function loadFactories(platform = 'FXG') {
    loading.value.factories = true
    try {
      const data = await listFactories({ platform, pageNo: 1, pageSize: 100 })
      factories.value = data.items || []
    } finally {
      loading.value.factories = false
    }
  }

  async function loadOverviewStats() {
    loading.value.overview = true
    try {
      const [startDateTime, endDateTime] = defaultDateRange()
      const data = await listOrders({
        platform: 'FXG',
        tradeStatus: 'wait_audit',
        timeType: 0,
        startDateTime,
        endDateTime,
        pageNo: 1,
        pageSize: 1,
      })
      stats.value = data.stats
    } finally {
      loading.value.overview = false
    }
  }

  async function refreshAll() {
    await Promise.all([loadStatus(), loadAccounts(), loadShops(), loadOverviewStats()])
  }

  return {
    loginInfo,
    accounts,
    shops,
    factories,
    stats,
    accountVersion,
    loading,
    loadStatus,
    loadAccounts,
    loadShops,
    loadFactories,
    loadOverviewStats,
    switchKdzsAccount,
    refreshAll,
  }
})
