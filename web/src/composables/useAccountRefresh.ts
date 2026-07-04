import { watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useKdzsStore } from '../stores/kdzs'

/** 快递助手账号切换成功后，重新加载当前页数据。 */
export function useAccountRefresh(refresh: () => void | Promise<void>) {
  const kdzsStore = useKdzsStore()
  const { accountVersion } = storeToRefs(kdzsStore)
  watch(accountVersion, () => {
    void refresh()
  })
}
