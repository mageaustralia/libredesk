<template>
  <div>
    <Spinner v-if="isLoading" />
    <AdminPageWithHelp>
      <template #content>
        <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
          <div class="flex justify-between mb-5">
            <div class="flex justify-end mb-4 w-full">
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child>
                  <Button class="ml-auto">
                    {{
                      $t('globals.messages.new', {
                        name: $t('globals.terms.status')
                      })
                    }}
                  </Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle>
                      {{
                        $t('globals.messages.new', {
                          name: $t('globals.terms.status')
                        })
                      }}
                    </DialogTitle>
                    <DialogDescription>
                      {{ $t('admin.conversationStatus.name.description') }}
                    </DialogDescription>
                  </DialogHeader>
                  <StatusForm @submit.prevent="onSubmit">
                    <template #footer>
                      <DialogFooter class="mt-10">
                        <Button type="submit" :isLoading="isLoading" :disabled="isLoading">
                          {{ $t('globals.messages.save') }}
                        </Button>
                      </DialogFooter>
                    </template>
                  </StatusForm>
                </DialogContent>
              </Dialog>
            </div>
          </div>
          <div class="space-y-1">
            <div
              v-for="(status, index) in statuses"
              :key="status.id"
              class="flex items-center gap-3 px-3 py-2 border rounded-md bg-background hover:bg-muted/50 cursor-grab active:cursor-grabbing"
              draggable="true"
              @dragstart="onDragStart(index, $event)"
              @dragover.prevent="onDragOver(index)"
              @dragend="onDragEnd"
              :class="{ 'opacity-50': dragIndex === index, 'border-primary': dropIndex === index }"
            >
              <GripVertical class="h-4 w-4 text-muted-foreground shrink-0" />
              <span class="flex-1 text-sm">{{ status.name }}</span>
              <Popover>
                <PopoverTrigger asChild>
                  <button
                    class="h-7 px-2.5 text-xs border rounded cursor-pointer shrink-0 flex items-center gap-1.5 min-w-[90px]"
                    :style="colorPreviewStyle(status.color || 'gray')"
                  >
                    <span class="w-2.5 h-2.5 rounded-full shrink-0 border border-black/10" :style="{ backgroundColor: colorDot(status.color || 'gray') }"></span>
                    {{ colorLabel(status.color || 'gray') }}
                    <ChevronDown class="w-3 h-3 ml-auto opacity-50" />
                  </button>
                </PopoverTrigger>
                <PopoverContent class="w-[160px] p-1" align="end">
                  <button
                    v-for="c in colorOptions"
                    :key="c.value"
                    class="flex items-center gap-2 w-full px-2 py-1.5 text-xs rounded hover:bg-muted cursor-pointer"
                    :class="{ 'bg-muted': (status.color || 'gray') === c.value }"
                    @click="updateColor(status, c.value); closePopover($event)"
                  >
                    <span class="w-4 h-4 rounded shrink-0 border border-black/10" :style="{ backgroundColor: c.bg }"></span>
                    <span :style="{ color: c.text, fontWeight: (status.color || 'gray') === c.value ? 600 : 400 }">{{ c.label }}</span>
                  </button>
                </PopoverContent>
              </Popover>
              <label class="flex items-center gap-1.5 text-xs text-muted-foreground cursor-pointer shrink-0" title="Show in Send dropdown">
                <Checkbox
                  :checked="status.show_on_send"
                  @update:checked="toggleShowOnSend(status)"
                  class="h-3.5 w-3.5"
                />
                Send menu
              </label>
              <Button
                v-if="!isDefault(status.name)"
                variant="ghost"
                size="xs"
                @click="deleteStatus(status.id)"
              >
                <Trash2 class="h-3.5 w-3.5 text-muted-foreground" />
              </Button>
              <Button v-else variant="ghost" size="xs" disabled class="opacity-30 cursor-default">
                <Lock class="h-3.5 w-3.5 text-muted-foreground" />
              </Button>
            </div>
          </div>
        </div>
      </template>

      <template #help>
        <p>Create custom conversation statuses to extend default workflow.</p>
      </template>
    </AdminPageWithHelp>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
// DataTable removed — using drag/drop list
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { GripVertical, Trash2, Lock, ChevronDown } from 'lucide-vue-next'
import { Checkbox } from '@/components/ui/checkbox'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import StatusForm from '@/features/admin/status/StatusForm.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '@/features/admin/status/formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

const { t } = useI18n()
const isLoading = ref(false)
const statuses = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getStatuses()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'status') getStatuses()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const getStatuses = async () => {
  try {
    isLoading.value = true
    const resp = await api.getStatuses()
    statuses.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    isLoading.value = true
    await api.createStatus(values)
    dialogOpen.value = false
    getStatuses()
  } catch (error) {
    console.error('Failed to create status:', error)
  } finally {
    isLoading.value = false
  }
})

const defaultStatuses = ['Open', 'Snoozed', 'Resolved', 'Closed', 'Spam', 'Trashed']
const isDefault = (name) => defaultStatuses.includes(name)

const dragIndex = ref(null)
const dropIndex = ref(null)

const onDragStart = (index, event) => {
  dragIndex.value = index
  event.dataTransfer.effectAllowed = 'move'
}

const onDragOver = (index) => {
  if (dragIndex.value === null || dragIndex.value === index) return
  dropIndex.value = index
  const item = statuses.value.splice(dragIndex.value, 1)[0]
  statuses.value.splice(index, 0, item)
  dragIndex.value = index
}

const onDragEnd = async () => {
  dragIndex.value = null
  dropIndex.value = null
  try {
    await api.reorderStatuses(statuses.value.map(s => s.id))
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: 'Failed to save order'
    })
    getStatuses()
  }
}

const toggleShowOnSend = async (status) => {
  const newValue = !status.show_on_send
  status.show_on_send = newValue
  try {
    await api.toggleStatusShowOnSend(status.id, newValue)
  } catch (error) {
    status.show_on_send = !newValue
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: 'Failed to update'
    })
  }
}

const colorOptions = [
  { value: 'gray', label: 'Gray', bg: '#f3f4f6', text: '#4b5563' },
  { value: 'red', label: 'Red', bg: '#fee2e2', text: '#b91c1c' },
  { value: 'orange', label: 'Orange', bg: '#ffedd5', text: '#c2410c' },
  { value: 'amber', label: 'Amber', bg: '#fef3c7', text: '#b45309' },
  { value: 'yellow', label: 'Yellow', bg: '#fef9c3', text: '#a16207' },
  { value: 'lime', label: 'Lime', bg: '#ecfccb', text: '#4d7c0f' },
  { value: 'green', label: 'Green', bg: '#dcfce7', text: '#15803d' },
  { value: 'teal', label: 'Teal', bg: '#ccfbf1', text: '#0f766e' },
  { value: 'cyan', label: 'Cyan', bg: '#cffafe', text: '#0e7490' },
  { value: 'blue', label: 'Blue', bg: '#dbeafe', text: '#1d4ed8' },
  { value: 'indigo', label: 'Indigo', bg: '#e0e7ff', text: '#4338ca' },
  { value: 'purple', label: 'Purple', bg: '#f3e8ff', text: '#7e22ce' },
  { value: 'pink', label: 'Pink', bg: '#fce7f3', text: '#be185d' },
  { value: 'rose', label: 'Rose', bg: '#ffe4e6', text: '#be123c' },
  { value: 'slate', label: 'Slate', bg: '#e2e8f0', text: '#475569' },
]

const colorLabel = (color) => {
  const c = colorOptions.find(o => o.value === color)
  return c ? c.label : 'Gray'
}

const colorDot = (color) => {
  const c = colorOptions.find(o => o.value === color)
  return c ? c.text : '#4b5563'
}

const closePopover = (event) => {
  // Click the popover trigger to close it
  const popover = event.target.closest('[data-radix-popper-content-wrapper]')
  if (popover) {
    const trigger = popover.previousElementSibling || document.querySelector('[data-state="open"]')
    if (trigger) trigger.click()
  }
}

const colorPreviewStyle = (color) => {
  const c = colorOptions.find(o => o.value === color)
  if (!c) return {}
  return { backgroundColor: c.bg, color: c.text, borderColor: c.text + '33' }
}

const updateColor = async (status, color) => {
  const oldColor = status.color
  status.color = color
  try {
    await api.updateStatusColor(status.id, color)
  } catch (error) {
    status.color = oldColor
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: 'Failed to update color'
    })
  }
}

const deleteStatus = async (id) => {
  try {
    await api.deleteStatus(id)
    getStatuses()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: error?.response?.data?.message || 'Failed to delete status'
    })
  }
}
</script>
