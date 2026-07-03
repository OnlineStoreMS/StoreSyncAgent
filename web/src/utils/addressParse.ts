/** 将手填的新收件地址解析为统一格式（用于展示与复制补发） */
export function parseRecipientAddress(raw: string): string {
  const text = raw.trim()
  if (!text) return ''

  const labeled = parseLabeledFormat(text)
  if (labeled) return labeled

  const phone = extractMobile(text)
  let body = text
  if (phone) {
    body = body.replace(new RegExp(phone.replace(/(\d)/g, '$1\\s*'), 'g'), ' ')
    body = body.replace(/[,，\s]*$/, '').replace(/^[,，\s]*/, '')
  }

  const nameFromShou = body.match(/([\u4e00-\u9fa5A-Za-z·]{1,8})收/)
  let name = nameFromShou?.[1] || ''
  if (nameFromShou) {
    body = body.replace(nameFromShou[0], '')
  }

  if (!name && phone) {
    const nameBeforePhone = text.match(/[，,]\s*([\u4e00-\u9fa5A-Za-z·]{1,8})(?:收)?\s*[,，]?\s*1[3-9]\d{9}/)
    if (nameBeforePhone) name = nameBeforePhone[1]
  }

  body = body
    .replace(/[，,]\s*[\u4e00-\u9fa5A-Za-z·]{1,8}(?:收)?/g, '')
    .replace(/[,，]+/g, '，')
    .replace(/\n+/g, ' ')
    .trim()

  const lines: string[] = []
  if (name) lines.push(`收件人: ${name}`)
  if (phone) lines.push(`手机号码: ${phone}`)
  if (body) lines.push(`详细地址: ${body}`)

  return lines.join('\n') || text
}

function parseLabeledFormat(text: string): string | null {
  const normalized = text.replace(/\r\n/g, '\n')
  const name =
    pickLabel(normalized, ['收货人', '收件人']) ||
    pickLabel(normalized, ['联系人'])
  const mobile = pickLabel(normalized, ['手机号码', '手机号', '电话', '联系电话'])
  const region = pickLabel(normalized, ['所在地区', '地区'])
  const address = pickLabel(normalized, ['详细地址', '地址'])

  if (!name && !mobile && !region && !address) {
    return null
  }

  const lines: string[] = []
  if (name) lines.push(`收件人: ${name}`)
  if (mobile) lines.push(`手机号码: ${normalizeMobile(mobile)}`)
  if (region) lines.push(`所在地区: ${region}`)
  if (address) lines.push(`详细地址: ${address}`)
  return lines.join('\n')
}

function pickLabel(text: string, labels: string[]): string {
  for (const label of labels) {
    const re = new RegExp(`${label}\\s*[：:]\\s*([^\\n,，]+)`, 'i')
    const m = text.match(re)
    if (m?.[1]) return m[1].trim()
    const reMultiline = new RegExp(`${label}\\s*[：:]\\s*([^\\n]+)`, 'i')
    const m2 = text.match(reMultiline)
    if (m2?.[1]) return m2[1].trim()
  }
  return ''
}

function extractMobile(text: string): string {
  const m = text.match(/1[3-9]\d{9}/)
  return m?.[0] || ''
}

function normalizeMobile(v: string): string {
  const m = v.match(/1[3-9]\d{9}/)
  return m?.[0] || v.trim()
}

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
