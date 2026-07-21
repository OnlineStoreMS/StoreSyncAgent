function pad(n: number) {
  return String(n).padStart(2, '0')
}

export function formatDateTime(d: Date) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** 订单解密复制用：2026 07/11 12:42 */
export function formatOrderCopyDateTime(d: Date): string {
  return `${d.getFullYear()} ${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function defaultDateRange(): [string, string] {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - 29)
  start.setHours(0, 0, 0, 0)
  end.setHours(23, 59, 59, 0)
  return [formatDateTime(start), formatDateTime(end)]
}

/** 「全部」状态默认近 30 天（各状态合集） */
export function allOrdersDateRange(): [string, string] {
  return defaultDateRange()
}

export const dateShortcuts = [
  {
    text: '近7天',
    value: () => {
      const end = new Date()
      end.setHours(23, 59, 59, 0)
      const start = new Date()
      start.setDate(start.getDate() - 6)
      start.setHours(0, 0, 0, 0)
      return [start, end]
    },
  },
  {
    text: '近30天',
    value: () => {
      const end = new Date()
      end.setHours(23, 59, 59, 0)
      const start = new Date()
      start.setDate(start.getDate() - 29)
      start.setHours(0, 0, 0, 0)
      return [start, end]
    },
  },
  {
    text: '近1年',
    value: () => {
      const end = new Date()
      end.setHours(23, 59, 59, 0)
      const start = new Date()
      start.setFullYear(start.getFullYear() - 1)
      start.setHours(0, 0, 0, 0)
      return [start, end]
    },
  },
  {
    text: '近2年',
    value: () => {
      const end = new Date()
      end.setHours(23, 59, 59, 0)
      const start = new Date()
      start.setFullYear(start.getFullYear() - 2)
      start.setHours(0, 0, 0, 0)
      return [start, end]
    },
  },
]
