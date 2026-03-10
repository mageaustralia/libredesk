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
import { GripVertical, Trash2 } from 'lucide-vue-next'
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
