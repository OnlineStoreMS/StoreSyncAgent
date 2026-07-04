import axios from 'axios'

const http = axios.create({ baseURL: '/api/v1', timeout: 60000 })

export interface Shop {
  id: number
  platform: string
  platformName: string
  mallUserId: string
  mallUserName: string
  bindTime: string
  expireTime: string
  tokenValid: boolean
}

export interface TradeGoods {
  title?: string
  skuName?: string
  picUrl?: string
  num?: number
  outerId?: string
  price?: number
}

export interface Order {
  platform: string
  platformName: string
  togetherId?: string
  sysTids?: string[]
  tids?: string[]
  buyerNick?: string
  receiverName?: string
  receiverMobile?: string
  receiverAddress?: string
  buyerMemo?: string
  sellerMemo?: string
  fenFaMemo?: string
  printerMemo?: string
  agentType?: number
  factoryId?: string
  factoryName?: string
  decrypted?: boolean
  formattedReceiver?: string
  payment?: number
  tradeStatus?: string
  statusText?: string
  createTime?: string
  payTime?: string
  shopName?: string
  shopId?: string
  goods?: TradeGoods[]
}

export interface KdzsAccount {
  id: string
  name: string
  role: string
  roleLabel: string
  mobile: string
  active: boolean
}

export interface FactoryItem {
  id: number
  factoryId: string
  factoryName: string
  factoryNick?: string
  remark?: string
  bindStatus?: number
  bindTime?: string
  hasPrePushTrade?: boolean
  supportBindItem?: boolean
}

export interface AgentTypeResult {
  successList?: string[]
  failList?: string[]
  failMessageMap?: Record<string, string>
}

export interface OrderFilters {
  timeType: number
  timeTypeLabel: string
  startDateTime: string
  endDateTime: string
  platform?: string
  platformName?: string
  shopId?: string
  tradeStatus: string
  statusLabel: string
}

export interface OrderListResponse {
  total: number
  pageNo: number
  pageSize: number
  items: Order[]
  filters?: OrderFilters
  stats?: {
    waitingPushTotal: number
    waitingSendTotal: number
    waitingPushByPlatform?: Record<string, number>
    tabWaitAudit?: number
    tabWaitSend?: number
  }
  hint?: string
}

export async function getLoginStatus() {
  const { data } = await http.get('/kdzs/status')
  return data
}

export async function listShops() {
  const { data } = await http.get<{ items: Shop[]; total: number }>('/shops')
  return data
}

export async function listOrders(params: {
  platform?: string
  shopId?: string
  tradeStatus?: string
  pageNo?: number
  pageSize?: number
  timeType?: number
  startDateTime?: string
  endDateTime?: string
}) {
  const { data } = await http.get<OrderListResponse>('/orders', { params })
  return data
}

export async function decryptOrders(body: {
  platform: string
  tradeStatus?: string
  sysTids: string[]
}) {
  const { data } = await http.post<{ items: Order[] }>('/orders/decrypt', body)
  return data
}

export async function listAccounts() {
  const { data } = await http.get<{ items: KdzsAccount[] }>('/kdzs/accounts')
  return data
}

export async function switchAccount(accountId: string) {
  const { data } = await http.post('/kdzs/accounts/switch', { accountId })
  return data
}

export async function listFactories(params: { platform?: string; pageNo?: number; pageSize?: number }) {
  const { data } = await http.get<{ total: number; pageNo: number; pageSize: number; items: FactoryItem[] }>('/factories', { params })
  return data
}

export async function setOrderAgentType(body: {
  platform: string
  tradeStatus?: string
  action: 'self_print' | 'push_factory'
  factoryId?: string
  sysTids: string[]
}) {
  const { data } = await http.post<AgentTypeResult>('/orders/agent-type', body)
  return data
}

export interface RefundGoods {
  title?: string
  skuName?: string
  picUrl?: string
  num?: number
  refundAmount?: string
}

export interface RefundSLA {
  scenario?: string
  deadlineAt?: string
  remainingSeconds?: number
  remainingText?: string
  urgency?: 'imminent' | 'critical' | 'warning' | 'normal' | 'expired' | 'unknown' | 'none'
  source?: string
  hint?: string
  logisticsStatus?: string
  logisticsStatusDesc?: string
  signTime?: string
  acceptTime?: string
  inboundTime?: string
  isSigned?: boolean
  isInbound?: boolean
  isPickupPending?: boolean
  pickupHint?: string
  important?: boolean
}

export interface RefundItem {
  platform: string
  platformName: string
  refundId: string
  tid?: string
  sysTid?: string
  afterSaleStatus: string
  afterSaleStatusText: string
  afterSaleType: number
  afterSaleTypeText: string
  refundReason?: string
  refundAmount?: string
  confirmTime?: string
  created?: string
  buyerNick?: string
  shopName?: string
  shopId?: string
  sid?: string
  sidCode?: string
  factoryName?: string
  orderSent?: boolean
  goods?: RefundGoods[]
  sla?: RefundSLA
}

export interface RefundStats {
  total: number
  waitSellerConfirmReceive: number
  waitSellerAgree: number
  refundOnlyPending: number
  exchangePending: number
  waitSendExchange: number
  returnSigned: number
  pickupPending: number
  urgent: number
  imminent: number
  critical: number
  expired: number
}

export interface RefundListResponse {
  total: number
  pageNo: number
  pageSize: number
  items: RefundItem[]
  filters?: {
    dateType: number
    dateTypeLabel: string
    startDateTime: string
    endDateTime: string
    platform?: string
    platformName?: string
    shopId?: string
    scenario?: string
    sid?: string
  }
  stats?: RefundStats
}

export interface LogisticsTrace {
  time?: string
  desc?: string
  logisticsStatus?: string
}

export interface LogisticsDetail {
  mailNo?: string
  cpCode?: string
  logisticsCompanyName?: string
  logisticsStatus?: string
  logisticsStatusDesc?: string
  traceList?: LogisticsTrace[]
  signTime?: string
  inboundTime?: string
  isSigned?: boolean
  isInbound?: boolean
  isPickupPending?: boolean
  pickupHint?: string
}

export async function listRefunds(params: {
  platform?: string
  shopId?: string
  pageNo?: number
  pageSize?: number
  dateType?: number
  startDateTime?: string
  endDateTime?: string
  afterSaleStatus?: string
  afterSaleType?: string
  sid?: string
  refundId?: string
  tid?: string
  sysTid?: string
  scenario?: string
  enrichLogistics?: boolean
}) {
  const { data } = await http.get<RefundListResponse>('/refunds', { params })
  return data
}

export async function getRefundStats(params: {
  platform?: string
  shopId?: string
  startDateTime?: string
  endDateTime?: string
}) {
  const { data } = await http.get<RefundStats>('/refunds/stats', { params })
  return data
}

export async function getRefundLogistics(params: { platform?: string; sid: string; sidCode?: string }) {
  const { data } = await http.get<LogisticsDetail>('/refunds/logistics', { params })
  return data
}

export interface ReturnExchangeRecord {
  id: string
  seqNo?: number
  buyerNick?: string
  afterSaleType?: string
  returnTrackingNo?: string
  spec?: string
  feedbackTime?: string
  submitTime?: string
  orderNo?: string
  recipientInfo?: string
  parsedRecipientInfo?: string
  outboundTrackingNo?: string
  remark?: string
  platform?: string
  sysTid?: string
  shopName?: string
  goods?: TradeGoods[]
  goodsTitle?: string
  originalRecipientInfo?: string
  payment?: number
  payTime?: string
  statusText?: string
  createdAt?: string
  updatedAt?: string
}

export interface OrderLookup {
  found: boolean
  orderNo: string
  platform?: string
  sysTid?: string
  shopName?: string
  goods?: TradeGoods[]
  goodsTitle?: string
  originalRecipientInfo?: string
  payment?: number
  payTime?: string
  statusText?: string
  source?: string
}

export async function listReturnExchanges() {
  const { data } = await http.get<{ items: ReturnExchangeRecord[]; total: number }>('/return-exchanges')
  return data
}

export async function createReturnExchange(body: Partial<ReturnExchangeRecord>) {
  const { data } = await http.post<ReturnExchangeRecord>('/return-exchanges', body)
  return data
}

export async function updateReturnExchange(id: string, body: Partial<ReturnExchangeRecord>) {
  const { data } = await http.put<ReturnExchangeRecord>(`/return-exchanges/${id}`, body)
  return data
}

export async function deleteReturnExchange(id: string) {
  const { data } = await http.delete<{ ok: boolean }>(`/return-exchanges/${id}`)
  return data
}

export async function lookupOrder(params: { orderNo: string; platform?: string }) {
  const { data } = await http.get<OrderLookup>('/orders/lookup', { params })
  return data
}

export interface NotificationScenarioOption {
  key: string
  label: string
}

export interface NotificationConfig {
  enabled: boolean
  webhookUrl: string
  secret?: string
  secretSet?: boolean
  appId?: string
  appSecret?: string
  appSecretSet?: boolean
  platform: string
  pollIntervalMinutes: number
  dateRangeDays: number
  scenarios: string[]
  /** 空数组表示全部 config 中的 accounts */
  accountIds?: string[]
}

export interface NotificationState {
  lastRunAt?: string
  lastRunOk?: boolean
  lastError?: string
  lastSentCount?: number
  lastBarcodeError?: string
}

export interface NotificationView {
  config: NotificationConfig
  state: NotificationState
  scenarios: NotificationScenarioOption[]
  accounts: KdzsAccount[]
}

export async function getNotification() {
  const { data } = await http.get<NotificationView>('/notifications')
  return data
}

export async function saveNotification(body: NotificationConfig) {
  const { data } = await http.put<NotificationView>('/notifications', body)
  return data
}

export async function testNotification(text?: string) {
  const { data } = await http.post<{ ok: boolean }>('/notifications/test', { text })
  return data
}

export async function testBarcodeNotification() {
  const { data } = await http.post<{ ok: boolean }>('/notifications/test-barcode')
  return data
}

export async function runNotification() {
  const { data } = await http.post<{ sent: number; skipped: number; barcodeWarnings?: number; lastBarcodeError?: string; error?: string }>('/notifications/run')
  return data
}

export async function resetNotificationState() {
  const { data } = await http.post<{ cleared: number; view: NotificationView }>('/notifications/reset-state')
  return data
}
