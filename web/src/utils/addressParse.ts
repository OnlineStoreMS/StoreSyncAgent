export function parseDateValue(s?: string): number {
  if (!s?.trim()) return 0
  const m = s.trim().match(/(\d{4})[.\-/年](\d{1,2})[.\-/月](\d{1,2})/)
  if (m) {
    return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])).getTime()
  }
  const t = Date.parse(s)
  return Number.isNaN(t) ? 0 : t
}

export function formatCopyDate(d: Date): string {
  return `${d.getFullYear()}.${d.getMonth() + 1}.${d.getDate()}`
}
