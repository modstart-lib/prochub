import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as AppAPI from '../../wailsjs/go/main/App'
import { process as ProcessModels } from '../../wailsjs/go/models'
import { i18n } from '../plugins/i18n'
import { trackError } from '../services/analytics'

export type ProcessStatus = 'running' | 'stopped' | 'errored' | 'starting'

export interface ProcessItem {
  id: string
  name: string
  command: string
  args: string[]
  workingDir: string
  status: ProcessStatus
  autoStart: boolean
  autoRestart: boolean
  restartPolicy: string
  maxRetries: number
  env: Record<string, string>
  pid: number
  restarts: number
  lastError: string
}

export const useAppStore = defineStore('app', () => {
  const locale = ref<'zh' | 'en'>('zh')
  const isDark = ref(false)
  const processes = ref<ProcessItem[]>([])
  const logs = ref<string[]>([])

  const t = (key: string, params?: Record<string, unknown>) => i18n.global.t(key, params || {})

  const setLocale = async (nextLocale: 'zh' | 'en') => {
    locale.value = nextLocale
    i18n.global.locale.value = nextLocale
    // Save to backend config
    try {
      const config = await AppAPI.GetConfig()
      config.locale = nextLocale
      await AppAPI.UpdateConfig(config)
    } catch (error) {
      console.error('Failed to save locale setting:', error)
    }
  }

  const setTheme = async (nextIsDark: boolean) => {
    isDark.value = nextIsDark
    document.documentElement.classList.toggle('dark', nextIsDark)
    // Save theme preference to localStorage for now
    // (backend config doesn't have a theme field yet)
    localStorage.setItem('theme', nextIsDark ? 'dark' : 'light')
  }

  // Load processes from backend
  const loadProcesses = async () => {
    try {
      const snapshots = await AppAPI.ListProcesses()
      processes.value = snapshots.map((snap: ProcessModels.Snapshot) => ({
        id: snap.definition.id,
        name: snap.definition.name,
        command: snap.definition.command,
        args: snap.definition.args || [],
        workingDir: snap.definition.workingDir || '',
        status: snap.status as ProcessStatus,
        autoStart: snap.definition.autoStart || false,
        autoRestart: snap.definition.autoRestart || false,
        restartPolicy: snap.definition.restartPolicy || 'on_failure',
        maxRetries: snap.definition.maxRetries || 0,
        env: snap.definition.env || {},
        pid: snap.pid,
        restarts: snap.restarts,
        lastError: snap.lastError || '',
      }))
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to load processes: ${errorMsg}`)
      console.error('Failed to load processes:', error)
    }
  }

  // Add a new process
  const addProcess = async (definition: ProcessModels.Definition) => {
    try {
      await AppAPI.AddProcess(definition)
      await loadProcesses()
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to add process: ${errorMsg}`)
      console.error('Failed to add process:', error)
      throw error
    }
  }

  // Remove a process
  const removeProcess = async (id: string) => {
    try {
      await AppAPI.RemoveProcess(id)
      await loadProcesses()
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to remove process: ${errorMsg}`)
      console.error('Failed to remove process:', error)
      throw error
    }
  }

  // Update a process
  const updateProcess = async (id: string, definition: ProcessModels.Definition) => {
    try {
      await AppAPI.UpdateProcess(id, definition)
      await loadProcesses()
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to update process: ${errorMsg}`)
      console.error('Failed to update process:', error)
      throw error
    }
  }

  // Start a process
  const startProcess = async (id: string) => {
    try {
      await AppAPI.StartProcess(id)
      // Refresh after a short delay to get updated status
      setTimeout(loadProcesses, 500)
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to start process: ${errorMsg}`)
      console.error('Failed to start process:', error)
      throw error
    }
  }

  // Stop a process
  const stopProcess = async (id: string) => {
    try {
      await AppAPI.StopProcess(id)
      setTimeout(loadProcesses, 500)
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to stop process: ${errorMsg}`)
      console.error('Failed to stop process:', error)
      throw error
    }
  }

  // Restart a process
  const restartProcess = async (id: string) => {
    try {
      await AppAPI.RestartProcess(id)
      setTimeout(loadProcesses, 500)
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to restart process: ${errorMsg}`)
      console.error('Failed to restart process:', error)
      throw error
    }
  }

  // Load logs for a specific process
  const loadProcessLogs = async (id: string) => {
    try {
      const entries = await AppAPI.GetProcessLogs(id)
      logs.value = entries.map((entry: any) => {
        const date = new Date(entry.timestamp)
        const timestamp = date.toLocaleString('zh-CN', {
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit',
          hour12: false
        })
        return `[${timestamp}] ${entry.stream}: ${entry.line}`
      })
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to load logs: ${errorMsg}`)
      console.error('Failed to load logs:', error)
    }
  }

  const runningCount = computed(() => processes.value.filter((item) => item.status === 'running').length)
  const stoppedCount = computed(() => processes.value.filter((item) => item.status === 'stopped').length)
  const failedCount = computed(() => processes.value.filter((item) => item.status === 'errored').length)

  // Initialize app settings from backend
  const initSettings = async () => {
    try {
      const config = await AppAPI.GetConfig()
      // Load locale from backend
      if (config.locale) {
        const localeValue = config.locale as 'zh' | 'en'
        locale.value = localeValue
        i18n.global.locale.value = localeValue
      }
      // Load theme from localStorage
      const savedTheme = localStorage.getItem('theme')
      if (savedTheme) {
        const dark = savedTheme === 'dark'
        isDark.value = dark
        document.documentElement.classList.toggle('dark', dark)
      }
    } catch (error) {
      console.error('Failed to load settings:', error)
    }
  }

  return {
    locale,
    isDark,
    processes,
    logs,
    runningCount,
    stoppedCount,
    failedCount,
    setLocale,
    setTheme,
    t,
    initSettings,
    loadProcesses,
    addProcess,
    removeProcess,
    updateProcess,
    startProcess,
    stopProcess,
    restartProcess,
    loadProcessLogs,
  }
})

