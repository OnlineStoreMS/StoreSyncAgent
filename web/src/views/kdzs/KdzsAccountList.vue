<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createKdzsAccount,
  deleteKdzsAccount,
  listKdzsAccountDetails,
  setDefaultKdzsAccount,
  switchAccount,
  updateKdzsAccount,
  type KdzsAccountDetail,
} from '../../api'
import { useKdzsStore } from '../../stores/kdzs'

const kdzsStore = useKdzsStore()
const loading = ref(false)
const items = ref<KdzsAccountDetail[]>([])
const dialogVisible = ref(false)
const editingCode = ref<string | null>(null)
const form = reactive({
  code: '',
  name: '',
  role: 'merchant',
  mobile: '',
  password: '',
  enabled: true,
})

async function load() {
  loading.value = true
  try {
    const data = await listKdzsAccountDetails()
    items.value = data.items || []
  } catch (e: any) {
    ElMessage.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingCode.value = null
  form.code = ''
  form.name = ''
  form.role = 'merchant'
  form.mobile = ''
  form.password = ''
  form.enabled = true
  dialogVisible.value = true
}

function openEdit(row: KdzsAccountDetail) {
  editingCode.value = row.code
  form.code = row.code
  form.name = row.name
  form.role = row.role
  form.mobile = row.mobile
  form.password = ''
  form.enabled = row.enabled
  dialogVisible.value = true
}

async function submit() {
  try {
    if (editingCode.value) {
      await updateKdzsAccount(editingCode.value, {
        name: form.name,
        role: form.role,
        mobile: form.mobile,
        password: form.password,
        enabled: form.enabled,
      })
      ElMessage.success('已更新')
    } else {
      await createKdzsAccount({
        code: form.code,
        name: form.name,
        role: form.role,
        mobile: form.mobile,
        password: form.password,
        enabled: form.enabled,
      })
      ElMessage.success('已添加')
    }
    dialogVisible.value = false
    await Promise.all([load(), kdzsStore.loadAccounts(), kdzsStore.loadStatus()])
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  }
}

async function onDelete(row: KdzsAccountDetail) {
  try {
    await ElMessageBox.confirm(`确定删除账号 ${row.name}？`, '删除确认', { type: 'warning' })
    await deleteKdzsAccount(row.code)
    ElMessage.success('已删除')
    await Promise.all([load(), kdzsStore.loadAccounts()])
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error(e.message || '删除失败')
  }
}

async function onSetDefault(row: KdzsAccountDetail) {
  try {
    await setDefaultKdzsAccount(row.code)
    ElMessage.success('已设为默认账号')
    await load()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function onSwitch(row: KdzsAccountDetail) {
  try {
    await switchAccount(row.code)
    ElMessage.success('已切换当前账号')
    await Promise.all([load(), kdzsStore.loadStatus(), kdzsStore.loadAccounts()])
  } catch (e: any) {
    ElMessage.error(e.message || '切换失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="account-page">
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="row-between">
          <div class="card-title">
            快递助手账号
            <span class="hint">按当前租户隔离存储，密码保存在数据库</span>
          </div>
          <el-button type="primary" @click="openCreate">添加账号</el-button>
        </div>
      </template>

      <el-table :data="items" v-loading="loading" stripe border empty-text="暂无账号，请先添加">
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="code" label="账号 ID" min-width="160" />
        <el-table-column prop="mobile" label="手机号" width="130" />
        <el-table-column prop="roleLabel" label="类型" width="90" />
        <el-table-column label="状态" width="180">
          <template #default="{ row }">
            <el-tag v-if="row.active" type="success" size="small">当前使用</el-tag>
            <el-tag v-if="row.isDefault" type="primary" size="small" effect="plain">默认</el-tag>
            <el-tag v-if="!row.enabled" type="info" size="small">已禁用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="onSwitch(row)">切换</el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-if="!row.isDefault" link type="primary" @click="onSetDefault(row)">设默认</el-button>
            <el-button link type="danger" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="editingCode ? '编辑账号' : '添加快递助手账号'"
      width="520px"
      destroy-on-close
    >
      <el-form label-width="96px">
        <el-form-item label="账号 ID" required>
          <el-input v-model="form.code" :disabled="!!editingCode" placeholder="如 account_13107749258" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="显示名称，默认同手机号" />
        </el-form-item>
        <el-form-item label="手机号" required>
          <el-input v-model="form.mobile" />
        </el-form-item>
        <el-form-item label="密码" :required="!editingCode">
          <el-input v-model="form.password" type="password" show-password :placeholder="editingCode ? '留空则不修改' : ''" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.role" style="width: 100%">
            <el-option label="商家版" value="merchant" />
            <el-option label="厂家版" value="factory" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.card-title {
  font-weight: 600;
}
.hint {
  margin-left: 8px;
  font-size: 12px;
  color: #909399;
  font-weight: normal;
}
.row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
