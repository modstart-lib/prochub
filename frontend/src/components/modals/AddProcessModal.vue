<script lang="ts" setup>
import { Button, Divider, Form, FormItem, Input, InputNumber, Modal, Select, Switch, TabPane, Tabs, message } from 'ant-design-vue'
import { FileSearch, Folder, FolderOpen, Minus, Plus, Settings2, Terminal, Variable } from 'lucide-vue-next'
import { reactive, ref, watch } from 'vue'
import * as AppAPI from '../../../wailsjs/go/main/App'
import { process as ProcessModels } from '../../../wailsjs/go/models'
import { useAppStore } from '../../stores/app'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': [boolean] }>()
const appStore = useAppStore()

const form = reactive({
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

// 从命令路径提取工作目录
const extractWorkingDir = (command: string): string => {
  if (!command) return ''
  // 检查是否是绝对路径
  if (command.startsWith('/') || command.match(/^[A-Za-z]:\\/)) {
    const lastSlash = Math.max(command.lastIndexOf('/'), command.lastIndexOf('\\'))
    if (lastSlash > 0) {
      return command.substring(0, lastSlash)
    }
  }
  return ''
}

// 监听命令变化，自动填充工作目录
watch(() => form.command, (newCommand) => {
  if (!form.workingDir) {
    const dir = extractWorkingDir(newCommand)
    if (dir) {
      form.workingDir = dir
    }
  }
})

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
  if (!form.command) {
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
    id: '',
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
    await appStore.addProcess(definition)
    message.success(appStore.t('messages.processAdded') || 'Process added successfully')
    emit('update:visible', false)
  } catch (error) {
    message.error(appStore.t('messages.operationFailed') || 'Failed to add process')
  }
}

watch(
  () => props.visible,
  (next) => {
    if (!next) {
      resetForm()
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
    :title="appStore.t('processes.addTitle')"
    :ok-text="appStore.t('actions.save')"
    :cancel-text="appStore.t('actions.cancel')"
    width="560px"
    class="process-modal"
    @ok="handleOk"
    @cancel="emit('update:visible', false)"
  >
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

.w-full {
  width: 100%;
}
</style>
