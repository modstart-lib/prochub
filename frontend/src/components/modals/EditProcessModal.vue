<script lang="ts" setup>
import { Button, Divider, Form, FormItem, Input, InputNumber, Modal, Popconfirm, Select, Switch, TabPane, Tabs, message } from 'ant-design-vue'
import { Folder, FolderOpen, Minus, Plus, Settings2, Terminal, Trash2, Variable } from 'lucide-vue-next'
import { reactive, ref, watch } from 'vue'
import * as AppAPI from '../../../wailsjs/go/main/App'
import { process as ProcessModels } from '../../../wailsjs/go/models'
import { trackVisit } from '../../services/analytics'
import { useAppStore, type ProcessItem } from '../../stores/app'

const props = defineProps<{ visible: boolean; process: ProcessItem | null }>()

// Track visit when modal opens
watch(() => props.visible, (visible) => {
  if (visible) {
    trackVisit('EditProcess')
  }
})
const emit = defineEmits<{ 'update:visible': [boolean] }>()
const appStore = useAppStore()

const form = reactive({
  id: '',
  name: '',
  command: '',
  args: '',
  workingDir: '',
  autoStart: false,
  restartPolicy: 'on_failure',
  maxRetries: 5,
  env: [{ key: '', value: '' }],
})

const activeTab = ref('basic')

const resetForm = () => {
  form.id = ''
  form.name = ''
  form.command = ''
  form.args = ''
  form.workingDir = ''
  form.autoStart = false
  form.restartPolicy = 'on_failure'
  form.maxRetries = 5
  form.env = [{ key: '', value: '' }]
  activeTab.value = 'basic'
}

const hydrateForm = (process: ProcessItem | null) => {
  if (!process) {
    resetForm()
    return
  }

  form.id = process.id
  form.name = process.name
  form.command = process.command
  form.args = process.args.join(' ')
  form.workingDir = process.workingDir
  form.autoStart = process.autoStart
  form.restartPolicy = process.restartPolicy
  form.maxRetries = process.maxRetries
  const envEntries = Object.entries(process.env || {})
  form.env = envEntries.length
    ? envEntries.map(([key, value]) => ({ key, value }))
    : [{ key: '', value: '' }]
}

// 选择工作目录
const selectWorkingDir = async () => {
  try {
    const dir = await AppAPI.SelectDirectory()
    if (dir) {
      form.workingDir = dir
    }
  } catch (error) {
    console.error('Failed to select directory:', error)
  }
}

// 选择命令文件
const selectCommand = async () => {
  try {
    const file = await AppAPI.SelectFile()
    if (file) {
      form.command = file
    }
  } catch (error) {
    console.error('Failed to select file:', error)
  }
}

const addEnv = () => {
  form.env.push({ key: '', value: '' })
}

const removeEnv = (index: number) => {
  form.env.splice(index, 1)
  if (form.env.length === 0) {
    form.env.push({ key: '', value: '' })
  }
}

const handleOk = async () => {
  if (!form.command || !form.id) {
    message.warning(appStore.t('validation.commandRequired') || 'Please enter a command')
    return
  }

  const envMap: Record<string, string> = {}
  form.env.forEach((item) => {
    if (item.key) {
      envMap[item.key] = item.value
    }
  })

  const argsArray = form.args ? form.args.split(' ').filter(arg => arg.trim()) : []

  const definition = new ProcessModels.Definition({
    id: form.id,
    name: form.name || appStore.t('processes.unnamed'),
    command: form.command,
    args: argsArray,
    workingDir: form.workingDir,
    env: envMap,
    autoStart: form.autoStart,
    autoRestart: form.restartPolicy !== 'never',
    restartPolicy: form.restartPolicy,
    maxRetries: form.maxRetries,
  })

  try {
    await appStore.updateProcess(form.id, definition)
    message.success(appStore.t('messages.processUpdated') || 'Process updated successfully')
    emit('update:visible', false)
  } catch (error) {
    message.error(appStore.t('messages.operationFailed') || 'Failed to update process')
  }
}

const handleDelete = async () => {
  if (!form.id) return
  
  try {
    await appStore.removeProcess(form.id)
    message.success(appStore.t('messages.processRemoved') || 'Process removed successfully')
    emit('update:visible', false)
  } catch (error) {
    message.error(appStore.t('messages.operationFailed') || 'Failed to remove process')
  }
}

watch(
  () => props.visible,
  (next) => {
    if (!next) {
      resetForm()
      return
    }
    hydrateForm(props.process)
  },
)

watch(
  () => props.process,
  (next) => {
    if (props.visible) {
      hydrateForm(next)
    }
  },
)

const restartPolicyOptions = [
  { value: 'always', label: 'Always' },
  { value: 'on_failure', label: 'On Failure' },
  { value: 'never', label: 'Never' },
]
</script>

<template>
  <Modal
    :open="props.visible"
    :title="appStore.t('processes.editTitle')"
    width="560px"
    class="process-modal"
    @cancel="emit('update:visible', false)"
  >
    <template #footer>
      <div class="modal-footer">
        <Popconfirm
          :title="appStore.t('messages.confirmDelete') || 'Are you sure you want to delete this process?'"
          :ok-text="appStore.t('actions.confirm') || 'Yes'"
          :cancel-text="appStore.t('actions.cancel') || 'No'"
          placement="topLeft"
          @confirm="handleDelete"
        >
          <Button danger type="primary">
            <template #icon><Trash2 :size="14" /></template>
            {{ appStore.t('actions.delete') || 'Delete' }}
          </Button>
        </Popconfirm>
        <div class="footer-right">
          <Button @click="emit('update:visible', false)">
            {{ appStore.t('actions.cancel') }}
          </Button>
          <Button type="primary" @click="handleOk">
            {{ appStore.t('actions.save') }}
          </Button>
        </div>
      </div>
    </template>

    <Tabs v-model:activeKey="activeTab" class="modal-tabs">
      <!-- 基础设置 -->
      <TabPane key="basic">
        <template #tab>
          <span class="tab-label">
            <Terminal :size="14" />
            {{ appStore.t('tabs.basic') || 'Basic' }}
          </span>
        </template>
        <Form layout="vertical" class="modal-form">
          <FormItem :label="appStore.t('processes.fields.name')">
            <Input 
              v-model:value="form.name" 
              :placeholder="appStore.t('processes.placeholders.name')"
            />
          </FormItem>
          <FormItem :label="appStore.t('processes.fields.command')" required>
            <div class="input-with-button">
              <Input 
                v-model:value="form.command" 
                :placeholder="appStore.t('processes.placeholders.command')"
              >
                <template #prefix>
                  <Terminal :size="14" class="input-icon" />
                </template>
              </Input>
              <Button @click="selectCommand">
                <template #icon>
                  <FileSearch :size="14" />
                </template>
              </Button>
            </div>
          </FormItem>
          <FormItem :label="appStore.t('processes.fields.args')">
            <Input 
              v-model:value="form.args" 
              :placeholder="appStore.t('processes.placeholders.args')"
            />
          </FormItem>
          <FormItem :label="appStore.t('processes.fields.workingDir')">
            <div class="input-with-button">
              <Input 
                v-model:value="form.workingDir" 
                :placeholder="appStore.t('processes.placeholders.workingDir')"
              >
                <template #prefix>
                  <Folder :size="14" class="input-icon" />
                </template>
              </Input>
              <Button @click="selectWorkingDir">
                <template #icon>
                  <FolderOpen :size="14" />
                </template>
              </Button>
            </div>
          </FormItem>
        </Form>
      </TabPane>

      <!-- 高级设置 -->
      <TabPane key="advanced">
        <template #tab>
          <span class="tab-label">
            <Settings2 :size="14" />
            {{ appStore.t('tabs.advanced') || 'Advanced' }}
          </span>
        </template>
        <Form layout="vertical" class="modal-form">
          <FormItem :label="appStore.t('processes.fields.autoStart')">
            <div class="switch-wrapper">
              <Switch v-model:checked="form.autoStart" />
              <span class="switch-label">{{ form.autoStart ? appStore.t('actions.enabled') : appStore.t('actions.disabled') }}</span>
            </div>
          </FormItem>
          <FormItem :label="appStore.t('processes.fields.restartPolicy')">
            <Select 
              v-model:value="form.restartPolicy" 
              :options="restartPolicyOptions"
            />
          </FormItem>
          <FormItem :label="appStore.t('processes.fields.maxRetries')">
            <InputNumber 
              v-model:value="form.maxRetries" 
              :min="0" 
              :max="100"
              class="w-full"
            />
          </FormItem>
        </Form>
      </TabPane>

      <!-- 环境变量 -->
      <TabPane key="env">
        <template #tab>
          <span class="tab-label">
            <Variable :size="14" />
            {{ appStore.t('tabs.environment') || 'Environment' }}
          </span>
        </template>
        <div class="env-section">
          <div class="env-header">
            <span class="env-title">{{ appStore.t('processes.fields.env') }}</span>
            <Button type="dashed" size="small" @click="addEnv">
              <template #icon><Plus :size="14" /></template>
              {{ appStore.t('actions.addEnv') }}
            </Button>
          </div>
          <Divider class="my-3" />
          <div class="env-list">
            <div v-for="(item, index) in form.env" :key="index" class="env-row">
              <Input 
                v-model:value="item.key" 
                placeholder="KEY" 
                class="env-key"
              />
              <span class="env-separator">=</span>
              <Input 
                v-model:value="item.value" 
                placeholder="VALUE" 
                class="env-value"
              />
              <Button 
                type="text" 
                danger 
                size="small" 
                @click="removeEnv(index)"
                :disabled="form.env.length === 1 && !item.key && !item.value"
              >
                <template #icon><Minus :size="14" /></template>
              </Button>
            </div>
          </div>
        </div>
      </TabPane>
    </Tabs>
  </Modal>
</template>

<style scoped>
.process-modal :deep(.ant-modal-content) {
  @apply rounded-xl;
}

.modal-tabs :deep(.ant-tabs-nav) {
  @apply mb-4;
}

.tab-label {
  @apply flex items-center gap-1.5;
}

.modal-form {
  @apply space-y-4;
}

.input-icon {
  @apply text-slate-400;
}

.switch-wrapper {
  @apply flex items-center gap-3;
}

.switch-label {
  @apply text-sm text-slate-600 dark:text-slate-400;
}

.env-section {
  @apply py-2;
}

.env-header {
  @apply flex items-center justify-between;
}

.env-title {
  @apply text-sm font-medium text-slate-700 dark:text-slate-300;
}

.env-list {
  @apply space-y-2;
}

.env-row {
  @apply flex items-center gap-2;
}

.env-key {
  @apply flex-1;
}

.env-separator {
  @apply text-slate-400 font-mono;
}

.env-value {
  @apply flex-1;
}

.input-with-button {
  @apply flex items-center gap-2;
}

.input-with-button .ant-input-affix-wrapper {
  @apply flex-1;
}

.modal-footer {
  @apply flex items-center justify-between;
}

.footer-right {
  @apply flex items-center gap-2;
}

.w-full {
  width: 100%;
}
</style>
