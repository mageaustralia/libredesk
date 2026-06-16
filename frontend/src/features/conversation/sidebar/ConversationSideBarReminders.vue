<template>
  <div class="space-y-3">
    <!-- Existing pending reminders -->
    <div v-if="loading" class="text-xs text-muted-foreground">Loading…</div>
    <div v-else-if="pending.length" class="space-y-1.5">
      <div
        v-for="r in pending"
        :key="r.id"
        class="flex items-start justify-between gap-2 text-xs py-1.5 px-2 bg-muted/30 rounded border border-muted-foreground/10"
      >
        <div class="flex-1 min-w-0">
          <div class="font-medium">{{ formatRemindAt(r.remind_at) }}</div>
          <div v-if="r.note" class="text-muted-foreground mt-0.5 break-words">{{ r.note }}</div>
        </div>
        <button
          @click="remove(r.id)"
          :disabled="busy"
          class="text-muted-foreground hover:text-destructive shrink-0 mt-0.5"
          title="Delete reminder"
        >
          <X class="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
    <div v-else class="text-xs text-muted-foreground italic">No reminders set on this ticket.</div>

    <!-- Add form, collapsible to avoid cluttering the sidebar -->
    <Button
      v-if="!formOpen"
      size="sm"
      variant="outline"
      class="w-full h-8 text-xs"
      @click="formOpen = true"
    >
      <Plus class="h-3.5 w-3.5 mr-1" />
      Set a reminder
    </Button>

    <div v-else class="space-y-2 border rounded p-2 bg-background">
      <div class="text-xs font-semibold text-muted-foreground">When?</div>
      <div class="grid grid-cols-2 gap-1.5">
        <button
          v-for="preset in PRESETS"
          :key="preset.key"
          @click="setPreset(preset.minutes)"
          class="text-xs text-left px-2 py-1.5 border rounded hover:bg-accent cursor-pointer disabled:opacity-50"
          :disabled="busy"
        >{{ preset.label }}</button>
      </div>

      <div class="space-y-1.5">
        <label class="text-xs text-muted-foreground">Or pick a date and time</label>
        <input
          v-model="customAt"
          type="datetime-local"
          :min="minDateTimeLocal"
          class="reminder-datetime w-full h-8 px-2 text-xs border rounded bg-transparent outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      <div class="space-y-1.5">
        <label class="text-xs text-muted-foreground">Note (optional)</label>
        <input
          v-model="note"
          type="text"
          maxlength="500"
          placeholder="e.g. Check repaired board OK"
          class="w-full h-8 px-2 text-xs border rounded bg-transparent outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      <div class="flex gap-1.5">
        <Button
          v-if="customAt"
          size="sm"
          class="flex-1 h-8 text-xs"
          :disabled="busy || !customAtValid"
          @click="setCustom"
        >Set reminder</Button>
        <Button
          size="sm"
          variant="ghost"
          class="h-8 text-xs"
          @click="resetForm"
        >Cancel</Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Plus, X } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import api from '@/api'
import { useConversationStore } from '@/stores/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@/utils/http'

const conversationStore = useConversationStore()
const emitter = useEmitter()

// Same presets as the reply-box button — keep these in sync (ReminderButton.vue).
const PRESETS = [
  { key: '1d', label: 'In 1 day', minutes: 60 * 24 },
  { key: '3d', label: 'In 3 days', minutes: 60 * 24 * 3 },
  { key: '1w', label: 'In 1 week', minutes: 60 * 24 * 7 },
  { key: '1mo', label: 'In 1 month', minutes: 60 * 24 * 30 }
]

const pending = ref([])
const loading = ref(false)
const busy = ref(false)
const formOpen = ref(false)
const note = ref('')
const customAt = ref('')

const conversationUUID = computed(() => conversationStore.current?.uuid || '')

const minDateTimeLocal = computed(() => {
  const d = new Date()
  d.setSeconds(0, 0)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
})

const customAtValid = computed(() => {
  if (!customAt.value) return false
  const t = new Date(customAt.value)
  return !isNaN(t.valueOf()) && t.getTime() > Date.now() - 60_000
})

// Expose count so the sidebar trigger can render a badge.
defineExpose({ count: computed(() => pending.value.length) })

async function refresh() {
  if (!conversationUUID.value) {
    pending.value = []
    return
  }
  loading.value = true
  try {
    const { data } = await api.listConversationReminders(conversationUUID.value)
    pending.value = data?.data || []
  } catch (_) {
    pending.value = []
  } finally {
    loading.value = false
  }
}

function resetForm() {
  formOpen.value = false
  note.value = ''
  customAt.value = ''
}

async function setPreset(minutes) {
  const at = new Date(Date.now() + minutes * 60_000)
  await create(at)
}

async function setCustom() {
  if (!customAtValid.value) return
  await create(new Date(customAt.value))
}

async function create(at) {
  if (!conversationUUID.value) return
  busy.value = true
  try {
    await api.createReminder(conversationUUID.value, {
      remind_at: at.toISOString(),
      note: note.value.trim()
    })
    resetForm()
    await refresh()
    // Notify the ReminderButton in the reply box so its badge re-fetches.
    emitter.emit('reminders-changed', { uuid: conversationUUID.value })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'default',
      description: `Reminder set for ${formatRemindAt(at.toISOString())}`
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    busy.value = false
  }
}

async function remove(id) {
  busy.value = true
  try {
    await api.deleteReminder(id)
    await refresh()
    emitter.emit('reminders-changed', { uuid: conversationUUID.value })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    busy.value = false
  }
}

function formatRemindAt(iso) {
  const d = new Date(iso)
  if (isNaN(d.valueOf())) return iso
  return d.toLocaleString(undefined, {
    weekday: 'short',
    day: '2-digit',
    month: 'short',
    hour: 'numeric',
    minute: '2-digit'
  })
}

// React to conversation switches.
watch(conversationUUID, refresh, { immediate: true })

// External refresh trigger (from ReminderButton in the reply box).
function handleRemindersChanged(payload) {
  if (!payload || payload.uuid !== conversationUUID.value) return
  refresh()
}

onMounted(() => {
  emitter.on('reminders-changed', handleRemindersChanged)
})
onBeforeUnmount(() => {
  emitter.off('reminders-changed', handleRemindersChanged)
})
</script>
