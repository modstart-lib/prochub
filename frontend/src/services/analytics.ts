const BATCH_DELAY = 5000 // 5 seconds delay for batch sending

export type AnalyticsEvent = 'Open' | 'Visit' | 'Error'

export interface AnalyticsData {
  page?: string
  error?: string
  [key: string]: unknown
}

interface CollectPayload {
  name: AnalyticsEvent
  data?: AnalyticsData
}

// Event queue for batch sending
const eventQueue: CollectPayload[] = []
let batchTimer: ReturnType<typeof setTimeout> | null = null

// Extend window type for Wails bindings
declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          SendAnalytics?: (events: CollectPayload[]) => void
        }
      }
    }
  }
}

/**
 * Call the Go backend to send analytics
 */
function sendAnalyticsViaBackend(events: CollectPayload[]): void {
  // Use the Wails binding to call Go backend
  if (window.go?.main?.App?.SendAnalytics) {
    console.log('[Analytics] Sending via Go backend:', events)
    window.go.main.App.SendAnalytics(events)
  } else {
    console.warn('[Analytics] Go backend not available')
  }
}

/**
 * Flush all queued events to the server
 */
async function flushEvents(): Promise<void> {
  if (eventQueue.length === 0) {
    return
  }

  // Copy and clear the queue
  const eventsToSend = [...eventQueue]
  eventQueue.length = 0
  batchTimer = null

  console.log('[Analytics] Flushing events:', eventsToSend)
  sendAnalyticsViaBackend(eventsToSend)
}

/**
 * Queue an analytics event for batch sending
 */
function queueEvent(name: AnalyticsEvent, data?: AnalyticsData): void {
  const payload: CollectPayload = { name }
  if (data) {
    payload.data = data
  }

  console.log('[Analytics] Queuing event:', JSON.stringify(payload))
  eventQueue.push(payload)

  // Reset the timer if already running
  if (batchTimer) {
    clearTimeout(batchTimer)
  }

  // Set a new timer to flush events after delay
  batchTimer = setTimeout(() => {
    flushEvents()
  }, BATCH_DELAY)
}

/**
 * Track app open event
 * Called when the application is first loaded
 */
export function trackOpen(): void {
  console.log('[Analytics] trackOpen called')
  queueEvent('Open')
}

/**
 * Track page/view visit event
 * @param page - The name of the page or view being visited
 */
export function trackVisit(page: string): void {
  console.log('[Analytics] trackVisit called:', page)
  queueEvent('Visit', { page })
}

/**
 * Track error event
 * @param error - The error message or description
 */
export function trackError(error: string): void {
  console.log('[Analytics] trackError called:', error)
  queueEvent('Error', { error })
}

export default {
  trackOpen,
  trackVisit,
  trackError,
}
