import { Modal, message } from 'ant-design-vue'
import { CheckVersion, GetAppVersion } from '../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { useAppStore } from '../stores/app'

/**
 * True when the app is built for App Store distribution (VITE_APPSTORE_BUILD=true).
 * Version check UI and auto-check are disabled in this build.
 */
export const isAppStoreBuild = import.meta.env.VITE_APPSTORE_BUILD === 'true'

// Cache for app version
let cachedAppVersion: string | null = null

// Pre-release type weights, larger means newer
const PRE_WEIGHTS: Record<string, number> = {
  dev: 1,
  alpha: 2,
  beta: 3,
  rc: 4,
}

interface ParsedVersion {
  nums: number[]
  pre: { name: string; num: number | null } | null
}

/**
 * Parse a version string (e.g. "v0.2.0", "0.5.0-beta.1") into comparable parts.
 * Returns null when the string is not a semantic version (e.g. "2025-01-01").
 */
function parseVersion(version: string): ParsedVersion | null {
  const cleaned = version.trim().replace(/^v/i, '')
  const match = /^(\d+(?:\.\d+)*)(?:[-_.]([a-zA-Z]+)(?:[._-]?(\d+))?)?$/.exec(cleaned)
  if (!match) return null
  const nums = match[1].split('.').map(Number)
  while (nums.length < 3) nums.push(0)
  let pre: ParsedVersion['pre'] = null
  if (match[2]) {
    pre = {
      name: match[2].toLowerCase(),
      num: match[3] !== undefined && match[3] !== '' ? Number(match[3]) : null,
    }
  }
  return { nums, pre }
}

/**
 * Compare two semantic versions (e.g., "v0.2.0" vs "v0.3.0")
 * Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
 * Pre-release versions sort after their release counterpart (0.5.0 > 0.5.0-beta)
 */
function compareVersions(v1: string, v2: string): number {
  const a = parseVersion(v1)
  const b = parseVersion(v2)
  if (!a || !b) return 0

  const maxLength = Math.max(a.nums.length, b.nums.length)
  for (let i = 0; i < maxLength; i++) {
    const num1 = a.nums[i] || 0
    const num2 = b.nums[i] || 0
    if (num1 > num2) return 1
    if (num1 < num2) return -1
  }

  // Release (no pre-release suffix) is newer than any pre-release
  if (a.pre === null && b.pre === null) return 0
  if (a.pre === null) return 1
  if (b.pre === null) return -1

  const weightA = PRE_WEIGHTS[a.pre.name] ?? 0
  const weightB = PRE_WEIGHTS[b.pre.name] ?? 0
  if (weightA > weightB) return 1
  if (weightA < weightB) return -1

  const preNumA = a.pre.num ?? 0
  const preNumB = b.pre.num ?? 0
  if (preNumA > preNumB) return 1
  if (preNumA < preNumB) return -1
  return 0
}

// Get app version from backend (cached)
export async function getAppVersion(): Promise<string> {
  if (cachedAppVersion) {
    return cachedAppVersion
  }
  cachedAppVersion = await GetAppVersion()
  return cachedAppVersion
}

export interface VersionCheckOptions {
  /** Whether to show message when already on latest version */
  showLatestMessage?: boolean
  /** Whether to show error message on failure */
  showErrorMessage?: boolean
}

/**
 * Check for new version and prompt user to download if available
 */
export async function checkVersionAndPrompt(options: VersionCheckOptions = {}): Promise<boolean> {
  const { showLatestMessage = false, showErrorMessage = false } = options
  const appStore = useAppStore()

  try {
    const currentVersion = await getAppVersion()
    let versionInfo = await CheckVersion()
    
    // Handle case where response is a JSON string
    if (typeof versionInfo === 'string') {
      versionInfo = JSON.parse(versionInfo)
    }
    
    // Get version string, fallback to 'unknown' if empty
    const newVersion = versionInfo.version || 'unknown'

    // Compare versions: only show update if remote version is higher
    const comparison = compareVersions(newVersion, currentVersion)
    
    if (comparison <= 0) {
      // Remote version is same or lower than current version
      if (showLatestMessage) {
        message.success(appStore.t('settings.version.latestVersion'))
      }
      return false
    }

    if (versionInfo.url) {
      Modal.confirm({
        title: appStore.t('settings.version.updateAvailable'),
        content: appStore.t('settings.version.updateConfirm', { version: newVersion }),
        okText: appStore.t('common.yes'),
        cancelText: appStore.t('common.no'),
        onOk() {
          BrowserOpenURL(versionInfo.url!)
        },
      })
    }

    return true
  } catch (e) {
    console.error('Version check failed:', e)
    if (showErrorMessage) {
      message.error(appStore.t('settings.version.checkFailed'))
    }
    return false
  }
}

/**
 * Auto check version after a delay (used on app startup)
 */
export function autoCheckVersion(delayMs: number = 5000): void {
  setTimeout(() => {
    checkVersionAndPrompt({ showLatestMessage: false, showErrorMessage: false })
  }, delayMs)
}
