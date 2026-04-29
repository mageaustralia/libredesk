import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from './useEmitter'

/**
 * Thin wrapper around the SHOW_TOAST emitter event.
 *
 * Centralises the emitter+payload boilerplate that was previously duplicated
 * across ~27 sites in the conversation feature. Two main flavours:
 *
 *   toast.success(description)            // default variant
 *   toast.error(errOrMessage, fallback?)  // destructive variant; if passed an
 *                                         // axios error, runs handleHTTPError
 *                                         // and uses .message
 *
 * `warning` is currently an alias for `error` (same destructive variant) — the
 * shadcn toast only ships default + destructive, so we don't have a real
 * warning palette to differentiate against. Kept as a separate name so call
 * sites can express intent.
 */
export function useToast () {
  const emitter = useEmitter()

  const success = (description) => {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'default', description })
  }

  const error = (errOrMessage, fallbackMessage = '') => {
    const description = typeof errOrMessage === 'string'
      ? errOrMessage
      : (handleHTTPError(errOrMessage).message || fallbackMessage)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description })
  }

  const warning = (description) => {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description })
  }

  return { success, error, warning }
}
