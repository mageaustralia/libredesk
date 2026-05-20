import { ref } from 'vue'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'

/**
 * useAsyncAction — wraps the try/isRunning/toast/error pattern that
 * shows up identically in every settings page (and most save/test
 * handlers across the admin UI).
 *
 * Each instance tracks ONE in-flight action; spin up multiple instances
 * if a page has independent load + save + test calls (so e.g. clicking
 * Test while Save is still spinning doesn't interleave their flags).
 *
 * @param {object} [opts]
 * @param {string} [opts.defaultSuccessToast] - shown on success unless overridden per-call
 * @param {string} [opts.defaultErrorPrefix]  - prepended to the parsed HTTP error
 *
 * Returns:
 *  - isRunning: reactive boolean, true while the action is in-flight
 *  - error: reactive ref to the thrown error (null on success)
 *  - run(fn, opts?): executes fn(); resolves to fn's return value on
 *      success, or null on error. Errors are toasted, not rethrown —
 *      check error.value if the caller needs to branch on outcome.
 *      Per-call opts can override successToast (pass null to suppress).
 */
export function useAsyncAction ({ defaultSuccessToast = null, defaultErrorPrefix = '' } = {}) {
  const emitter = useEmitter()
  const isRunning = ref(false)
  const error = ref(null)

  const showToast = (description, variant) =>
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, variant ? { variant, description } : { description })

  const run = async (fn, opts = {}) => {
    const successToast = opts.successToast === undefined ? defaultSuccessToast : opts.successToast
    const silentOnError = opts.silentOnError === true
    isRunning.value = true
    error.value = null
    try {
      const result = await fn()
      if (successToast) showToast(successToast)
      return result
    } catch (err) {
      error.value = err
      if (!silentOnError) {
        const msg = handleHTTPError(err).message
        showToast(defaultErrorPrefix ? `${defaultErrorPrefix}: ${msg}` : msg, 'destructive')
      }
      return null
    } finally {
      isRunning.value = false
    }
  }

  return { isRunning, error, run }
}
