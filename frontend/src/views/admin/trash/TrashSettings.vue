<script setup>
import { ref, onMounted } from 'vue'
import { toast } from 'vue-sonner'
import api from '@/api'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import { Trash2 } from 'lucide-vue-next'

const loading = ref(true)
const saving = ref(false)

const autoTrashResolvedDays = ref(90)
const autoTrashSpamDays = ref(30)
const autoDeleteDays = ref(30)

onMounted(async () => {
  await fetchSettings()
})

async function fetchSettings() {
  loading.value = true
  try {
    const res = await api.getSettings('trash')
    const data = res.data?.data || {}
    if (data['trash.auto_trash_resolved_days'] !== undefined) {
      autoTrashResolvedDays.value = data['trash.auto_trash_resolved_days']
    }
    if (data['trash.auto_trash_spam_days'] !== undefined) {
      autoTrashSpamDays.value = data['trash.auto_trash_spam_days']
    }
    if (data['trash.auto_delete_days'] !== undefined) {
      autoDeleteDays.value = data['trash.auto_delete_days']
    }
  } catch (err) {
    console.error('Failed to load trash settings', err)
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  try {
    await api.updateSettings('trash', {
      'trash.auto_trash_resolved_days': parseInt(autoTrashResolvedDays.value) || 0,
      'trash.auto_trash_spam_days': parseInt(autoTrashSpamDays.value) || 0,
      'trash.auto_delete_days': parseInt(autoDeleteDays.value) || 0
    })
    toast.success('Trash settings saved')
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to save settings')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <AdminPageWithHelp>
    <template #content>
      <div v-if="loading" class="flex justify-center py-12">
        <Spinner />
      </div>

      <div v-else class="space-y-6">
        <Card>
          <CardHeader>
            <div class="flex items-center gap-2">
              <Trash2 class="h-5 w-5" />
              <CardTitle>Trash &amp; Cleanup</CardTitle>
            </div>
            <CardDescription>
              Configure automatic cleanup of resolved conversations, spam, and permanently trashed items.
              Settings take effect on the next hourly cleanup cycle.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="space-y-2">
              <Label for="auto-trash-resolved">Auto-trash resolved conversations after (days)</Label>
              <Input
                id="auto-trash-resolved"
                v-model="autoTrashResolvedDays"
                type="number"
                min="0"
                placeholder="90"
                class="max-w-xs"
              />
              <p class="text-xs text-muted-foreground">
                Resolved and closed conversations older than this many days will be automatically moved to trash.
                Set to 0 to disable.
              </p>
            </div>

            <div class="space-y-2">
              <Label for="auto-trash-spam">Auto-trash spam after (days)</Label>
              <Input
                id="auto-trash-spam"
                v-model="autoTrashSpamDays"
                type="number"
                min="0"
                placeholder="30"
                class="max-w-xs"
              />
              <p class="text-xs text-muted-foreground">
                Spam conversations older than this many days will be automatically moved to trash.
                Set to 0 to disable.
              </p>
            </div>

            <div class="space-y-2">
              <Label for="auto-delete">Permanently delete trashed items after (days)</Label>
              <Input
                id="auto-delete"
                v-model="autoDeleteDays"
                type="number"
                min="0"
                placeholder="30"
                class="max-w-xs"
              />
              <p class="text-xs text-muted-foreground">
                Items in trash older than this many days will be permanently deleted and cannot be recovered.
                Set to 0 to disable permanent deletion (items remain in trash indefinitely).
              </p>
            </div>

            <div class="pt-4">
              <Button @click="saveSettings" :disabled="saving">
                {{ saving ? 'Saving...' : 'Save' }}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>
    <template #help>
      <h4 class="font-medium mb-2">Automatic Cleanup</h4>
      <p class="text-sm text-muted-foreground mb-4">
        The cleanup process runs automatically every hour and applies these rules to keep your inbox tidy.
      </p>
      <h4 class="font-medium mb-2">How It Works</h4>
      <ul class="text-sm text-muted-foreground list-disc list-inside space-y-1">
        <li><strong>Resolved/Closed</strong> conversations are moved to trash after the configured number of days</li>
        <li><strong>Spam</strong> conversations are moved to trash after the configured number of days</li>
        <li><strong>Trashed</strong> items are permanently deleted after the configured number of days</li>
      </ul>
      <h4 class="font-medium mt-4 mb-2">Tips</h4>
      <ul class="text-sm text-muted-foreground list-disc list-inside space-y-1">
        <li>Set any value to 0 to disable that cleanup rule</li>
        <li>Changes apply on the next hourly cycle (no restart needed)</li>
        <li>Permanently deleted items cannot be recovered</li>
      </ul>
    </template>
  </AdminPageWithHelp>
</template>
