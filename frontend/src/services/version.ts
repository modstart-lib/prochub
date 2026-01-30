import { Modal, message } from 'ant-design-vue'
import { CheckVersion, GetAppVersion } from '../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { useAppStore } from '../stores/app'

// Cache for app version
let cachedAppVersion: string | null = null

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

    if (newVersion === currentVersion) {
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
