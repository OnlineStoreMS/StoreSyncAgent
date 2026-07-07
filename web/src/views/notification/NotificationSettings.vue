<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Bell, Delete, Promotion, Refresh } from '@element-plus/icons-vue'
import {
  getNotification,
  resetNotificationState,
  runNotification,
  saveNotification,
  testNotification,
  testBarcodeNotification,
  type KdzsAccount,
  type NotificationConfig,
  type NotificationState,
  type NotificationScenarioOption,
} from '../../api'
import { formatAccountTitle } from '../../utils/account'

const loading = reactive({ load: false, save: false, test: false, testBarcode: false, run: false, reset: false })
const scenarioOptions = ref<NotificationScenarioOption[]>([])
const accountOptions = ref<KdzsAccount[]>([])
const state = ref<NotificationState>({})
const secretInput = ref('')
const appSecretInput = ref('')

const form = reactive<NotificationConfig>({
  enabled: false,
  webhookUrl: '',
  appId: '',
  platform: 'FXG',
  pollIntervalMinutes: 15,
  dateRangeDays: 30,
  scenarios: [],
  accountIds: [],
})

async function load() {
  loading.load = true
  try {
    const data = await getNotification()
    Object.assign(form, data.config)
    form.scenarios = [...(data.config.scenarios || [])]
    form.accountIds = [...(data.config.accountIds || [])]
    state.value = data.state || {}
    scenarioOptions.value = data.scenarios || []
    accountOptions.value = data.accounts || []
    secretInput.value = ''
    appSecretInput.value = ''
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '加载失败')
  } finally {
    loading.load = false
  }
}

async function onSave() {
  loading.save = true
  try {
    const payload: NotificationConfig = {
      ...form,
      secret: secretInput.value || undefined,
      appSecret: appSecretInput.value || undefined,
    }
    const data = await saveNotification(payload)
    Object.assign(form, data.config)
    form.scenarios = [...(data.config.scenarios || [])]
    form.accountIds = [...(data.config.accountIds || [])]
    state.value = data.state || {}
    secretInput.value = ''
    appSecretInput.value = ''
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
    await testNotification('【电商店铺同步】飞书通知测试消息')
    ElMessage.success('测试消息已发送')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '测试失败')
  } finally {
    loading.test = false
  }
}

async function onTestBarcode() {
  loading.testBarcode = true
  try {
    await testBarcodeNotification()
    ElMessage.success('条形码测试卡片已发送')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '条形码测试失败')
  } finally {
    loading.testBarcode = false
  }
}

async function onRunNow() {
  loading.run = true
  try {
    const result = await runNotification()
    let msg = `检查完成：推送 ${result.sent} 条，跳过 ${result.skipped} 条`
    if (result.lastBarcodeError) {
      msg += `；条形码异常：${result.lastBarcodeError}`
      ElMessage.warning(msg)
    } else {
      ElMessage.success(msg)
    }
    await load()
  } catch (e: any) {
    const result = e?.response?.data?.result
    const msg = e?.response?.data?.error || e.message || '执行失败'
    if (result?.sent > 0) {
      ElMessage.warning(`${msg}（已推送 ${result.sent} 条）`)
      await load()
    } else {
      ElMessage.error(msg)
    }
  } finally {
    loading.run = false
  }
}

async function onResetState() {
  try {
    await ElMessageBox.confirm(
      '将清空已推送去重记录与运行状态，之后「立即检查」或定时任务会重新推送符合条件的售后通知。通知配置不会改动。',
      '重置通知记录',
      { type: 'warning', confirmButtonText: '确认重置', cancelButtonText: '取消' },
    )
  } catch {
    return
  }
  loading.reset = true
  try {
    const data = await resetNotificationState()
    if (data.view) {
      Object.assign(form, data.view.config)
      form.scenarios = [...(data.view.config.scenarios || [])]
      form.accountIds = [...(data.view.config.accountIds || [])]
      state.value = data.view.state || {}
    }
    ElMessage.success(`已重置，清除 ${data.cleared} 条去重记录`)
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e.message || '重置失败')
  } finally {
    loading.reset = false
  }
}

onMounted(load)
</script>

<template>
  <div class="notification-page" v-loading="loading.load">
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">
            <el-icon class="title-icon"><Bell /></el-icon>
            通知管理
          </div>
          <div class="actions">
            <el-button :icon="Refresh" @click="load">刷新</el-button>
            <el-button :loading="loading.test" @click="onTest">测试推送</el-button>
            <el-button :loading="loading.testBarcode" @click="onTestBarcode">测试条形码</el-button>
            <el-button type="warning" plain :loading="loading.run" @click="onRunNow">立即检查</el-button>
            <el-button type="primary" :loading="loading.save" @click="onSave">保存配置</el-button>
          </div>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        class="hint"
        title="通过飞书群机器人 Webhook 推送售后提醒。配置与去重状态保存在 data/notifications.json，无需数据库；服务会按设定间隔定时拉取售后并推送。"
      />

      <el-form label-width="120px" class="form">
        <el-form-item label="启用通知">
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
        <el-form-item label="飞书应用 ID">
          <el-input v-model="form.appId" placeholder="cli_xxx（用于上传物流单号条形码图片）" />
        </el-form-item>
        <el-form-item label="飞书应用 Secret">
          <el-input
            v-model="appSecretInput"
            type="password"
            show-password
            :placeholder="form.appSecretSet ? '已配置，留空则不修改' : '企业自建应用 App Secret'"
          />
          <div class="field-tip muted">需开启应用「机器人」能力，并开通「获取与上传图片或文件资源」权限且发布版本；有退货物流单号时卡片底部显示 Code128 条形码</div>
        </el-form-item>
        <el-form-item label="定时拉取">
          <div class="inline-field">
            每
            <el-input-number v-model="form.pollIntervalMinutes" :min="5" :max="1440" :step="5" />
            分钟检查一次（最小 5 分钟）
          </div>
        </el-form-item>
        <el-form-item label="扫描范围">
          <div class="inline-field">
            近
            <el-input-number v-model="form.dateRangeDays" :min="7" :max="90" />
            天申请的售后单
          </div>
        </el-form-item>
        <el-form-item label="平台">
          <el-select v-model="form.platform" style="width: 120px">
            <el-option label="抖店" value="FXG" />
          </el-select>
        </el-form-item>
        <el-form-item label="通知账号">
          <div class="account-field">
            <el-checkbox-group v-model="form.accountIds">
              <el-checkbox v-for="acc in accountOptions" :key="acc.id" :label="acc.id">
                {{ formatAccountTitle(acc) }}
              </el-checkbox>
            </el-checkbox-group>
            <div class="field-tip muted">不勾选任何账号时，默认扫描 config 中的全部快递助手账号</div>
          </div>
        </el-form-item>
        <el-form-item label="通知场景">
          <el-checkbox-group v-model="form.scenarios">
            <el-checkbox v-for="opt in scenarioOptions" :key="opt.key" :label="opt.key">
              {{ opt.label }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">运行状态</div>
          <el-button
            type="danger"
            plain
            :icon="Delete"
            :loading="loading.reset"
            @click="onResetState"
          >
            重置通知记录
          </el-button>
        </div>
      </template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="上次检查">{{ state.lastRunAt || '—' }}</el-descriptions-item>
        <el-descriptions-item label="结果">
          <el-tag v-if="!state.lastRunAt" type="info">未运行</el-tag>
          <el-tag v-else-if="state.lastRunOk" type="success">成功</el-tag>
          <el-tag v-else type="danger">失败</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="上次推送">{{ state.lastSentCount ?? '—' }} 条</el-descriptions-item>
        <el-descriptions-item label="错误信息">{{ state.lastError || '—' }}</el-descriptions-item>
        <el-descriptions-item label="条形码" :span="2">{{ state.lastBarcodeError || '—' }}</el-descriptions-item>
      </el-descriptions>
      <div class="status-tip muted">
        <el-icon><Promotion /></el-icon>
        同一账号下同一场景的同一售后单只推送一次；「时效紧迫」在 urgency 升级时会再次提醒（warning → critical 4h → imminent 30m → expired）。有物流单号且配置了飞书应用凭证时，卡片底部会显示条形码。多账号去重互不影响。
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.notification-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.title-icon {
  vertical-align: -2px;
  margin-right: 6px;
}
.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.hint {
  margin-bottom: 20px;
}
.form {
  max-width: 860px;
}
.inline-field {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.account-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.field-tip {
  font-size: 12px;
  line-height: 1.5;
}
.status-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 16px;
  font-size: 13px;
}
</style>
