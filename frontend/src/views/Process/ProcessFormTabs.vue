<script lang="ts" setup>
import { Button, Divider, Form, FormItem, Input, InputNumber, Select, Switch, TabPane, Tabs } from 'ant-design-vue'
import { FileSearch, Folder, FolderOpen, Minus, Plus, Settings2, Terminal, Variable } from 'lucide-vue-next'
import { computed } from 'vue'
import * as AppAPI from '../../../wailsjs/go/main/App'
import { useAppStore } from '../../stores/app'

export interface ProcessForm {
  name: string
  command: string
  args: string
  workingDir: string
  autoStart: boolean
  restartPolicy: string
  maxRetries: number
  env: Array<{ key: string; value: string }>
}

const props = defineProps<{ form: ProcessForm; activeTab: string }>()
const emit = defineEmits<{ 'update:activeTab': [string] }>()

const appStore = useAppStore()

const restartPolicyOptions = computed(() => [
  { value: 'always', label: appStore.t('processes.restartPolicies.always') },
  { value: 'on_failure', label: appStore.t('processes.restartPolicies.onFailure') },
  { value: 'never', label: appStore.t('processes.restartPolicies.never') },
])

// 选择工作目录
const selectWorkingDir = async () => {
  try {
    const dir = await AppAPI.SelectDirectory()
    if (dir) {
      props.form.workingDir = dir
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
      props.form.command = file
    }
  } catch (error) {
    console.error('Failed to select file:', error)
  }
}

const addEnv = () => {
  props.form.env.push({ key: '', value: '' })
}

const removeEnv = (index: number) => {
  props.form.env.splice(index, 1)
  if (props.form.env.length === 0) {
    props.form.env.push({ key: '', value: '' })
  }
}
</script>

<template>
  <Tabs
    :active-key="activeTab"
    class="modal-tabs"
    @change="(key: string | number) => emit('update:activeTab', String(key))"
  >
    <!-- 基础设置 -->
    <TabPane key="basic">
      <template #tab>
        <span class="tab-label">
          <Terminal class="w-4 h-4" aria-hidden="true" />
          {{ appStore.t('tabs.basic') }}
        </span>
      </template>
      <Form layout="vertical" class="modal-form">
        <FormItem :label="appStore.t('processes.fields.name')">
          <Input v-model:value="form.name" :placeholder="appStore.t('processes.placeholders.name')" />
        </FormItem>
        <FormItem :label="appStore.t('processes.fields.command')" required>
          <div class="input-with-button">
            <Input v-model:value="form.command" :placeholder="appStore.t('processes.placeholders.command')">
              <template #prefix>
                <Terminal class="w-4 h-4 input-icon" aria-hidden="true" />
              </template>
            </Input>
            <Button :aria-label="appStore.t('processes.pickCommand')" @click="selectCommand">
              <FileSearch class="w-4 h-4" aria-hidden="true" />
            </Button>
          </div>
        </FormItem>
        <FormItem :label="appStore.t('processes.fields.args')">
          <Input v-model:value="form.args" :placeholder="appStore.t('processes.placeholders.args')" />
        </FormItem>
        <FormItem :label="appStore.t('processes.fields.workingDir')">
          <div class="input-with-button">
            <Input v-model:value="form.workingDir" :placeholder="appStore.t('processes.placeholders.workingDir')">
              <template #prefix>
                <Folder class="w-4 h-4 input-icon" aria-hidden="true" />
              </template>
            </Input>
            <Button :aria-label="appStore.t('processes.pickWorkingDir')" @click="selectWorkingDir">
              <FolderOpen class="w-4 h-4" aria-hidden="true" />
            </Button>
          </div>
        </FormItem>
      </Form>
    </TabPane>

    <!-- 高级设置 -->
    <TabPane key="advanced">
      <template #tab>
        <span class="tab-label">
          <Settings2 class="w-4 h-4" aria-hidden="true" />
          {{ appStore.t('tabs.advanced') }}
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
          <Select v-model:value="form.restartPolicy" :options="restartPolicyOptions" />
        </FormItem>
        <FormItem :label="appStore.t('processes.fields.maxRetries')">
          <InputNumber v-model:value="form.maxRetries" :min="0" :max="100" class="w-full" />
        </FormItem>
      </Form>
    </TabPane>

    <!-- 环境变量 -->
    <TabPane key="env">
      <template #tab>
        <span class="tab-label">
          <Variable class="w-4 h-4" aria-hidden="true" />
          {{ appStore.t('tabs.environment') }}
        </span>
      </template>
      <div class="env-section">
        <div class="env-header">
          <span class="env-title">{{ appStore.t('processes.fields.env') }}</span>
          <Button type="dashed" @click="addEnv">
            <template #icon><Plus class="w-4 h-4" aria-hidden="true" /></template>
            {{ appStore.t('actions.addEnv') }}
          </Button>
        </div>
        <Divider class="my-3" />
        <div class="env-list">
          <div class="env-columns">
            <span class="env-key-column">{{ appStore.t('processes.env.key') }}</span>
            <span class="env-value-column">{{ appStore.t('processes.env.value') }}</span>
          </div>
          <div v-for="(item, index) in form.env" :key="index" class="env-row">
            <Input v-model:value="item.key" placeholder="KEY" class="env-key" />
            <span class="env-separator">=</span>
            <Input v-model:value="item.value" placeholder="VALUE" class="env-value" />
            <Button
              danger
              :aria-label="appStore.t('actions.remove')"
              :disabled="form.env.length === 1 && !item.key && !item.value"
              @click="removeEnv(index)"
            >
              <Minus class="w-4 h-4" aria-hidden="true" />
            </Button>
          </div>
        </div>
      </div>
    </TabPane>
  </Tabs>
</template>

<style scoped>
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

.env-columns {
  @apply flex items-center gap-2 px-0.5 text-xs font-medium text-slate-500 dark:text-slate-400;
}

.env-key-column,
.env-value-column {
  @apply flex-1;
}

.env-row {
  @apply flex items-center gap-2;
}

.env-key {
  @apply flex-1;
}

.env-separator {
  @apply font-mono text-slate-400;
}

.env-value {
  @apply flex-1;
}

.input-with-button {
  @apply flex items-center gap-2;
}

.input-with-button :deep(.ant-input-affix-wrapper) {
  @apply flex-1;
}

.w-full {
  width: 100%;
}
</style>
