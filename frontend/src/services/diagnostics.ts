import * as AppAPI from '../../wailsjs/go/main/App'
import { trackError } from './analytics'

type RendererLevel = 'info' | 'error'

// Persist a renderer event to the backend system log. This is the local
// counterpart of the remote analytics report: renderer errors must reach the
// log file even when offline or when the analytics endpoint is unavailable.
export function logRenderer(level: RendererLevel, label: string, data?: Record<string, unknown>): void {
  try {
    void AppAPI.LogFrontendError(level, label, data ?? {}).catch(() => {})
  } catch {
    // Wails bindings are not ready yet (very early boot errors): fall back to
    // the console, which the developer can still read.
  }
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

function stackOf(err: unknown): string | undefined {
  return err instanceof Error ? err.stack : undefined
}

// Register the boot marker and the global renderer error handlers. The handlers
// are installed before the app is mounted on purpose: startup / mount errors are
// the most common cause of a white screen and were previously missed.
export function installRendererDiagnostics(): void {
  logRenderer('info', 'boot', { href: location.href, ua: navigator.userAgent })

  // Uncaught JS errors.
  window.addEventListener('error', (event: ErrorEvent) => {
    const message = event.message || String(event)
    logRenderer('error', 'window.error', {
      message,
      stack: event.error?.stack,
      filename: event.filename,
      lineno: event.lineno,
      colno: event.colno,
    })
    console.error('[renderer.window.error]', event.error ?? message)
    trackError(`Uncaught Error: ${message}`)
  })

  // Resource (script / module / stylesheet) load failures do not bubble, so they
  // must be caught in the capture phase. This is the local analog of a router
  // lazy-chunk load failure: a missing bundle turns the page blank.
  window.addEventListener(
    'error',
    (event: Event) => {
      const target = event.target as HTMLElement | null
      // JS error events are dispatched on window and handled by the listener
      // above; only element targets mean a resource failed to load.
      if (!target || target === (window as unknown as EventTarget)) return
      const src = (target as HTMLScriptElement).src || (target as HTMLLinkElement).href || ''
      logRenderer('error', 'resource.error', { tag: target.tagName, src })
      console.error('[renderer.resource.error]', target.tagName, src)
      trackError(`Resource Load Error: ${target.tagName} ${src}`)
    },
    true,
  )

  // Unhandled promise rejections.
  window.addEventListener('unhandledrejection', (event: PromiseRejectionEvent) => {
    const message = messageOf(event.reason)
    const stack = stackOf(event.reason)
    logRenderer('error', 'window.unhandledrejection', { message, stack })
    console.error('[renderer.unhandledrejection]', event.reason)
    trackError(`Unhandled Promise Rejection: ${message}`)
  })
}

// logMounted writes the positive marker that tells a successful mount apart
// from a crash during mount (boot present but mounted missing).
export function logMounted(): void {
  logRenderer('info', 'mounted', { href: location.href })
}
