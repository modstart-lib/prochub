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
  const autoStart = ref(false)
  const autoStartWSL = ref(false)
  const processes = ref<ProcessItem[]>([])
  const logs = ref<string[]>([])

  const t = (key: string, params?: Record<string, unknown>) => i18n.global.t(key, params || {})

  // applyLocale updates the in-memory locale without touching the backend.
  const applyLocale = (nextLocale: 'zh' | 'en') => {
    locale.value = nextLocale
    i18n.global.locale.value = nextLocale
  }

  const setLocale = async (nextLocale: 'zh' | 'en') => {
    applyLocale(nextLocale)
    // Persist to backend config so the CLI shares the same source of truth.
    try {
      await AppAPI.SetLocale(nextLocale)
    } catch (error) {
      console.error('Failed to save locale setting:', error)
    }
  }

  // applyTheme applies the theme locally and mirrors it to localStorage so the
  // first-paint script can avoid a light flash.
  const applyTheme = (nextTheme: 'light' | 'dark') => {
    const dark = nextTheme === 'dark'
    isDark.value = dark
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem('theme', nextTheme)
  }

  const setTheme = async (nextIsDark: boolean) => {
    const nextTheme = nextIsDark ? 'dark' : 'light'
    applyTheme(nextTheme)
    try {
      await AppAPI.SetTheme(nextTheme)
    } catch (error) {
      console.error('Failed to save theme setting:', error)
    }
  }

  // applyConfig applies a full configuration snapshot pushed by the backend.
  // It is the single entry point for configuration arriving from any source
  // (GUI, CLI, tray), so the UI never drifts from the persisted state.
  const applyConfig = (config: {
    theme?: string
    locale?: string
    autoStart?: boolean
    autoStartWSL?: boolean
  }) => {
    if (config.theme) applyTheme(config.theme === 'dark' ? 'dark' : 'light')
    if (config.locale) applyLocale(config.locale === 'en' ? 'en' : 'zh')
    if (typeof config.autoStart === 'boolean') autoStart.value = config.autoStart
    if (typeof config.autoStartWSL === 'boolean') autoStartWSL.value = config.autoStartWSL
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

  // Toggle the auto-start flag of a process without stopping it
  const setProcessAutoStart = async (id: string, enabled: boolean) => {
    try {
      await AppAPI.SetProcessAutoStart(id, enabled)
      await loadProcesses()
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error)
      trackError(`Failed to set auto-start: ${errorMsg}`)
      console.error('Failed to set auto-start:', error)
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
      // Load theme from backend config, falling back to localStorage for
      // installs created before the theme field existed.
      const savedTheme = (config.theme as string) || localStorage.getItem('theme') || 'light'
      applyConfig({
        theme: savedTheme,
        locale: (config.locale as string) || 'zh',
        autoStart: config.autoStart,
        autoStartWSL: config.autoStartWSL,
      })
    } catch (error) {
      console.error('Failed to load settings:', error)
    }
  }

  return {
    locale,
    isDark,
    autoStart,
    autoStartWSL,
    processes,
    logs,
    runningCount,
    stoppedCount,
    failedCount,
    setLocale,
    setTheme,
    applyLocale,
    applyTheme,
    applyConfig,
    t,
    initSettings,
    loadProcesses,
    addProcess,
    removeProcess,
    updateProcess,
    setProcessAutoStart,
    startProcess,
    stopProcess,
    restartProcess,
    loadProcessLogs,
  }
})

