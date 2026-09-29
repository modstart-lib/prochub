<script lang="ts" setup>
import { Button, Modal, Popconfirm, message } from 'ant-design-vue'
import { Trash2 } from 'lucide-vue-next'
import { reactive, ref, watch } from 'vue'
import { process as ProcessModels } from '../../../wailsjs/go/models'
import { trackVisit } from '../../services/analytics'
import { useAppStore, type ProcessItem } from '../../stores/app'
import { testActionSet } from '../../utils/test'
import ProcessFormTabs, { type ProcessForm } from './ProcessFormTabs.vue'

const props = defineProps<{ visible: boolean; process: ProcessItem | null }>()

// Track visit when modal opens
watch(() => props.visible, (visible) => {
  if (visible) {
    trackVisit('EditProcess')
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

const hydrateForm = (process: ProcessItem | null) => {
  if (!process) {
    resetForm()
    return
  }

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

const handleOk = async () => {
  if (!form.command) {
    message.warning(appStore.t('validation.commandRequired'))
    return
  }

  const processId = props.process?.id
  if (!processId) return

  const envMap: Record<string, string> = {}
  form.env.forEach((item) => {
    if (item.key) {
      envMap[item.key] = item.value
    }
  })

  const argsArray = form.args ? form.args.split(' ').filter(arg => arg.trim()) : []

  const definition = new ProcessModels.Definition({
    id: processId,
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
    await appStore.updateProcess(processId, definition)
    message.success(appStore.t('messages.processUpdated'))
    emit('update:visible', false)
  } catch (error) {
    message.error(appStore.t('messages.operationFailed'))
  }
}

const handleDelete = async () => {
  if (!props.process?.id) return

  try {
    await appStore.removeProcess(props.process.id)
    message.success(appStore.t('messages.processRemoved'))
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

// ── 自动化测试 action ──────────────────────────────────────────────────────
testActionSet('ProcessEdit.getForm', () => ({ ...form, env: form.env.map((e) => ({ ...e })) }))
testActionSet('ProcessEdit.getTab', () => activeTab.value)
testActionSet('ProcessEdit.setTab', (params: unknown) => {
  const { tab } = params as { tab: string }
  activeTab.value = tab
})
testActionSet('ProcessEdit.fill', (params: unknown) => {
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
testActionSet('ProcessEdit.addEnv', (params: unknown) => {
  const { times } = (params ?? {}) as { times?: number }
  const n = Math.max(1, times ?? 1)
  for (let i = 0; i < n; i++) form.env.push({ key: '', value: '' })
})
testActionSet('ProcessEdit.getEnvCount', () => form.env.length)
testActionSet('ProcessEdit.submit', async () => {
  await handleOk()
  return { visible: props.visible }
})
testActionSet('ProcessEdit.delete', async () => {
  await handleDelete()
  return { visible: props.visible }
})
testActionSet('ProcessEdit.cancel', () => {
  emit('update:visible', false)
})
</script>

<template>
  <Modal
    :open="props.visible"
    :title="appStore.t('processes.editTitle')"
    width="min(600px, 90vw)"
    @cancel="emit('update:visible', false)"
  >
    <template #footer>
      <div class="modal-footer">
        <Popconfirm
          :title="appStore.t('messages.confirmDelete')"
          :ok-text="appStore.t('actions.confirm')"
          :cancel-text="appStore.t('actions.cancel')"
          placement="topLeft"
          @confirm="handleDelete"
        >
          <Button danger>
            <template #icon><Trash2 class="w-4 h-4" aria-hidden="true" /></template>
            {{ appStore.t('actions.delete') }}
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

    <ProcessFormTabs :form="form" :active-tab="activeTab" @update:active-tab="activeTab = $event" />
  </Modal>
</template>

<style scoped>
.modal-footer {
  @apply flex items-center justify-between;
}

.footer-right {
  @apply flex items-center gap-2;
}
</style>
