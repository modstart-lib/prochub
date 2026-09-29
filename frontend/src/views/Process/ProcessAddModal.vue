<script lang="ts" setup>
import { Modal, message } from 'ant-design-vue'
import { reactive, ref, watch } from 'vue'
import { process as ProcessModels } from '../../../wailsjs/go/models'
import { trackVisit } from '../../services/analytics'
import { useAppStore } from '../../stores/app'
import { testActionSet } from '../../utils/test'
import ProcessFormTabs, { type ProcessForm } from './ProcessFormTabs.vue'

const props = defineProps<{ visible: boolean }>()

// Track visit when modal opens
watch(() => props.visible, (visible) => {
  if (visible) {
    trackVisit('AddProcess')
  }
})
const emit = defineEmits<{ 'update:visible': [boolean] }>()
const appStore = useAppStore()

const form = reactive<ProcessForm>({
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

const handleOk = async () => {
  if (!form.command) {
    message.warning(appStore.t('validation.commandRequired'))
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
    message.success(appStore.t('messages.processAdded'))
    emit('update:visible', false)
  } catch (error) {
    message.error(appStore.t('messages.operationFailed'))
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

// ── 自动化测试 action ──────────────────────────────────────────────────────
testActionSet('ProcessAdd.getForm', () => ({ ...form, env: form.env.map((e) => ({ ...e })) }))
testActionSet('ProcessAdd.getTab', () => activeTab.value)
testActionSet('ProcessAdd.setTab', (params: unknown) => {
  const { tab } = params as { tab: string }
  activeTab.value = tab
})
testActionSet('ProcessAdd.fill', (params: unknown) => {
  const p = params as {
    name?: string
    command?: string
    args?: string
    workingDir?: string
    autoStart?: boolean
    restartPolicy?: string
    maxRetries?: number
    env?: Array<{ key: string; value: string }>
  }
  if (p.name !== undefined) form.name = p.name
  if (p.command !== undefined) form.command = p.command
  if (p.args !== undefined) form.args = p.args
  if (p.workingDir !== undefined) form.workingDir = p.workingDir
  if (p.autoStart !== undefined) form.autoStart = p.autoStart
  if (p.restartPolicy !== undefined) form.restartPolicy = p.restartPolicy
  if (p.maxRetries !== undefined) form.maxRetries = p.maxRetries
  if (p.env !== undefined) {
    form.env = p.env.length ? p.env.map((e) => ({ key: e.key, value: e.value })) : [{ key: '', value: '' }]
  }
})
testActionSet('ProcessAdd.addEnv', (params: unknown) => {
  const { times } = (params ?? {}) as { times?: number }
  const n = Math.max(1, times ?? 1)
  for (let i = 0; i < n; i++) form.env.push({ key: '', value: '' })
})
testActionSet('ProcessAdd.removeEnv', (params: unknown) => {
  const { index } = params as { index: number }
  form.env.splice(index, 1)
  if (form.env.length === 0) form.env.push({ key: '', value: '' })
})
testActionSet('ProcessAdd.getEnvCount', () => form.env.length)
testActionSet('ProcessAdd.submit', async () => {
  await handleOk()
  return { visible: props.visible }
})
testActionSet('ProcessAdd.cancel', () => {
  emit('update:visible', false)
})

// ── 截图专用 action（prepare / cleanup）─────────────────────────────────────

</script>

<template>
  <Modal
    :open="props.visible"
    :title="appStore.t('processes.addTitle')"
    :ok-text="appStore.t('actions.save')"
    :cancel-text="appStore.t('actions.cancel')"
    width="min(600px, 90vw)"
    @ok="handleOk"
    @cancel="emit('update:visible', false)"
  >
    <ProcessFormTabs :form="form" :active-tab="activeTab" @update:active-tab="activeTab = $event" />
  </Modal>
</template>
