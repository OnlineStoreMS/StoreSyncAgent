export interface AccountDisplay {
  name?: string
  mobile?: string
  roleLabel?: string
}

/** 主标题：name 与 mobile 相同时只显示一次；商家版不重复展示角色。 */
export function formatAccountTitle(acc: AccountDisplay) {
  const role = acc.roleLabel?.trim()
  const name = acc.name?.trim()
  const mobile = acc.mobile?.trim()
  const label = name && name !== mobile ? name : mobile || name || '未命名账号'
  if (!role || role === '商家版') {
    return label
  }
  return `${label}（${role}）`
}

/** 副标题：仅当 name 与 mobile 不同时返回手机号。 */
export function formatAccountSubtitle(acc: AccountDisplay) {
  const name = acc.name?.trim()
  const mobile = acc.mobile?.trim()
  if (name && mobile && name !== mobile) {
    return mobile
  }
  return ''
}
