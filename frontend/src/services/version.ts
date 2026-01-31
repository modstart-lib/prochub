import { Modal, message } from 'ant-design-vue'
import { CheckVersion, GetAppVersion } from '../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { useAppStore } from '../stores/app'

// Cache for app version
let cachedAppVersion: string | null = null

/**
 * Compare two semantic versions (e.g., "v0.2.0" vs "v0.3.0")
 * Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
 */
function compareVersions(v1: string, v2: string): number {
  // Remove 'v' prefix if present
  const clean1 = v1.replace(/^v/, '')
  const clean2 = v2.replace(/^v/, '')
  
  const parts1 = clean1.split('.').map(Number)
  const parts2 = clean2.split('.').map(Number)
  
  const maxLength = Math.max(parts1.length, parts2.length)
  
  for (let i = 0; i < maxLength; i++) {
    const num1 = parts1[i] || 0
    const num2 = parts2[i] || 0
    
    if (num1 > num2) return 1
    if (num1 < num2) return -1
  }
  
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
