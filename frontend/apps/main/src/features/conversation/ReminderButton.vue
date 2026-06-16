<template>
  <DropdownMenu v-model:open="open">
    <DropdownMenuTrigger asChild>
      <Toggle
        class="px-2 py-2 border-0 relative text-muted-foreground hover:text-foreground"
        variant="outline"
        :pressed="false"
        title="Set a personal reminder"
        aria-label="Set a personal reminder"
      >
        <Clock class="h-4 w-4" />
        <span
          v-if="pending.length"
          class="absolute -top-0.5 -right-0.5 bg-primary text-primary-foreground rounded-full w-4 h-4 text-[10px] flex items-center justify-center"
        >{{ pending.length }}</span>
      </Toggle>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="start" class="w-80 p-3 space-y-2">
      <div class="text-xs font-semibold text-muted-foreground">Remind me about this ticket</div>

      <!-- Presets -->
      <div class="grid grid-cols-2 gap-1.5">
        <button
          v-for="preset in PRESETS"
          :key="preset.key"
          @click="setPreset(preset.minutes)"
          class="text-xs text-left px-2 py-1.5 border rounded hover:bg-accent cursor-pointer disabled:opacity-50"
          :disabled="busy"
        >{{ preset.label }}</button>
      </div>

      <!-- Custom date + time -->
      <div class="border-t pt-2 space-y-1.5">
        <label class="text-xs text-muted-foreground">Pick a date and time</label>
        <input
          v-model="customAt"
          type="datetime-local"
          :min="minDateTimeLocal"
          class="reminder-datetime w-full h-8 px-2 text-xs border rounded bg-transparent outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      <!-- Note (shared across presets + custom) -->
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

      <Button
        v-if="customAt"
        size="sm"
        class="w-full h-8 text-xs"
        :disabled="busy || !customAtValid"
        @click="setCustom"
      >Set reminder</Button>

      <!-- Pending list -->
      <div v-if="pending.length" class="border-t pt-2">
        <div class="text-xs font-semibold text-muted-foreground mb-1.5">Your pending reminders</div>
        <div class="space-y-1">
          <div
            v-for="r in pending"
            :key="r.id"
            class="flex items-start justify-between gap-2 text-xs py-1 px-1.5 rounded hover:bg-accent/40"
          >
            <div class="flex-1 min-w-0">
              <div class="font-medium">{{ formatRemindAt(r.remind_at) }}</div>
              <div v-if="r.note" class="text-muted-foreground truncate">{{ r.note }}</div>
            </div>
            <button
              @click="remove(r.id)"
              class="text-muted-foreground hover:text-destructive shrink-0"
              :disabled="busy"
              title="Delete reminder"
            ><X class="h-3.5 w-3.5" /></button>
          </div>
        </div>
      </div>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Clock, X } from 'lucide-vue-next'
import { Toggle } from '@shared-ui/components/ui/toggle'
import { Button } from '@shared-ui/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const props = defineProps({
  conversationUUID: { type: String, required: true }
})

const emitter = useEmitter()

// Presets in minutes — kept simple and readable. Wesley's repaired-board case
// (one week) is the most common; tighter presets cover follow-ups within the
// week and the month preset covers longer-tail check-ins.
const PRESETS = [
  { key: '1d', label: 'In 1 day', minutes: 60 * 24 },
  { key: '3d', label: 'In 3 days', minutes: 60 * 24 * 3 },
  { key: '1w', label: 'In 1 week', minutes: 60 * 24 * 7 },
  { key: '1mo', label: 'In 1 month', minutes: 60 * 24 * 30 }
]

const open = ref(false)
const busy = ref(false)
const note = ref('')
const customAt = ref('') // datetime-local string in the browser's tz
const pending = ref([])

// `min` for the datetime-local input — now() rounded down to the minute, so
// the browser blocks past picks. Server also rejects, this is just a hint.
const minDateTimeLocal = computed(() => {
  const d = new Date()
  d.setSeconds(0, 0)
  // datetime-local wants "YYYY-MM-DDTHH:MM" in local time, no timezone.
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
})

const customAtValid = computed(() => {
  if (!customAt.value) return false
  const t = new Date(customAt.value)
  return !isNaN(t.valueOf()) && t.getTime() > Date.now() - 60_000
})

// Fetch existing reminders when the popover first opens — saves a request on
// every conversation switch for agents who never use this feature.
let loadedOnce = false
watch(open, async (isOpen) => {
  if (isOpen && !loadedOnce) {
    await refresh()
    loadedOnce = true
  }
})

// Reset state when the conversation changes.
watch(() => props.conversationUUID, () => {
  loadedOnce = false
  pending.value = []
  note.value = ''
  customAt.value = ''
})

async function refresh() {
  try {
    const { data } = await api.listConversationReminders(props.conversationUUID)
    pending.value = data?.data || []
  } catch (e) {
    // Soft-fail: if the table doesn't exist yet (migration not applied),
    // don't spam the user — just leave the list empty.
    pending.value = []
  }
}

async function setPreset(minutes) {
  const at = new Date(Date.now() + minutes * 60_000)
  await create(at)
}

async function setCustom() {
  if (!customAtValid.value) return
  const at = new Date(customAt.value)
  await create(at)
  customAt.value = ''
}

async function create(at) {
  busy.value = true
  try {
    await api.createReminder(props.conversationUUID, {
      remind_at: at.toISOString(),
      note: note.value.trim()
    })
    note.value = ''
    await refresh()
    // Tell the sidebar Reminders panel to re-fetch.
    emitter.emit('reminders-changed', { uuid: props.conversationUUID })
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
    emitter.emit('reminders-changed', { uuid: props.conversationUUID })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    busy.value = false
  }
}

// React to changes made via the sidebar Reminders panel so the badge count
// stays in sync without the agent having to re-open the popover.
function handleRemindersChanged(payload) {
  if (!payload || payload.uuid !== props.conversationUUID) return
  refresh()
}
emitter.on('reminders-changed', handleRemindersChanged)
onBeforeUnmount(() => emitter.off('reminders-changed', handleRemindersChanged))

// Format like "Tue 23 Jun, 2:15 pm" — concise enough for the popover list.
function formatRemindAt(iso) {
  const d = new Date(iso)
  if (isNaN(d.valueOf())) return iso
  const opts = { weekday: 'short', day: '2-digit', month: 'short', hour: 'numeric', minute: '2-digit' }
  return d.toLocaleString(undefined, opts)
}

// Pre-fetch on mount so the badge count shows up without waiting for the
// agent to open the popover. Cheap query.
onMounted(refresh)
</script>

<!-- Global (unscoped) so the ::-webkit-calendar-picker-indicator pseudo-element
     is reachable. The native widget's calendar glyph defaults to follow the
     page's color-scheme, which in our app stays ambiguous and ended up either
     black-on-dark (dark mode) or white-on-white (light sidebar). Force a
     filter that biases the icon to a mid-grey, visible against either bg. -->
<style>
input.reminder-datetime::-webkit-calendar-picker-indicator {
  filter: brightness(0.5);
  cursor: pointer;
}
.dark input.reminder-datetime::-webkit-calendar-picker-indicator {
  filter: invert(0.85);
}
</style>
