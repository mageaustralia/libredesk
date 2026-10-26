<template>
  <AdminSplitLayout>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }" class="space-y-6">
        <form @submit.prevent="onSubmit" class="space-y-8 w-full max-w-2xl">
          <div class="flex items-start justify-between gap-4">
            <div class="space-y-1">
              <Label>{{ t('admin.spamRescue.enabled') }}</Label>
              <p class="text-xs text-muted-foreground">
                {{ t('admin.spamRescue.enabledDescription') }}
              </p>
            </div>
            <Switch v-model:checked="enabled" />
          </div>

          <div class="space-y-2">
            <Label for="spam-rescue-mode">{{ t('admin.spamRescue.mode') }}</Label>
            <Select v-model="mode">
              <SelectTrigger id="spam-rescue-mode" class="max-w-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="shadow">{{ t('admin.spamRescue.modeShadow') }}</SelectItem>
                <SelectItem value="live">{{ t('admin.spamRescue.modeLive') }}</SelectItem>
              </SelectContent>
            </Select>
            <p class="text-xs text-muted-foreground">{{ t('admin.spamRescue.modeDescription') }}</p>
          </div>

          <div class="space-y-6">
            <div class="space-y-1">
              <h3 class="text-sm font-medium">{{ t('admin.spamRescue.rules') }}</h3>
              <p class="text-xs text-muted-foreground">
                {{ t('admin.spamRescue.rulesDescription') }}
              </p>
            </div>

            <div class="space-y-2">
              <Label for="spam-rescue-emails">{{ t('admin.spamRescue.trustedEmails') }}</Label>
              <Textarea
                id="spam-rescue-emails"
                v-model="trustedEmails"
                class="font-mono text-sm"
                rows="3"
                placeholder="partner@example.com"
              />
              <p class="text-xs text-muted-foreground">
                {{ t('admin.spamRescue.trustedEmailsDescription') }}
              </p>
            </div>

            <div class="space-y-2">
              <Label for="spam-rescue-domains">{{ t('admin.spamRescue.trustedDomains') }}</Label>
              <Textarea
                id="spam-rescue-domains"
                v-model="trustedDomains"
                class="font-mono text-sm"
                rows="3"
                placeholder="example.org"
              />
              <p class="text-xs text-muted-foreground">
                {{ t('admin.spamRescue.trustedDomainsDescription') }}
              </p>
            </div>

            <div class="flex items-start justify-between gap-4">
              <div class="space-y-1">
                <Label>{{ t('admin.spamRescue.ruleReplyToThread') }}</Label>
                <p class="text-xs text-muted-foreground">
                  {{ t('admin.spamRescue.ruleReplyToThreadDescription') }}
                </p>
              </div>
              <Switch v-model:checked="ruleReplyToThread" />
            </div>

            <div class="flex items-start justify-between gap-4">
              <div class="space-y-1">
                <Label>{{ t('admin.spamRescue.ruleEcommerceCustomer') }}</Label>
                <p class="text-xs text-muted-foreground">
                  {{ t('admin.spamRescue.ruleEcommerceCustomerDescription') }}
                </p>
              </div>
              <Switch v-model:checked="ruleEcommerceCustomer" />
            </div>

            <div class="space-y-3">
              <div class="flex items-start justify-between gap-4">
                <div class="space-y-1">
                  <Label>{{ t('admin.spamRescue.ruleOrderNumber') }}</Label>
                  <p class="text-xs text-muted-foreground">
                    {{ t('admin.spamRescue.ruleOrderNumberDescription') }}
                  </p>
                </div>
                <Switch v-model:checked="ruleOrderNumber" />
              </div>
              <Input
                v-model="orderNumberPattern"
                class="font-mono text-sm"
                placeholder="\b\d{5,8}\b"
              />
              <p class="text-xs text-muted-foreground">
                {{ t('admin.spamRescue.orderNumberPatternDescription') }}
              </p>
            </div>
          </div>

          <div class="space-y-6">
            <div class="space-y-1">
              <h3 class="text-sm font-medium">{{ t('admin.spamRescue.ai') }}</h3>
              <p class="text-xs text-muted-foreground">{{ t('admin.spamRescue.aiDescription') }}</p>
            </div>

            <div class="flex items-center justify-between gap-4">
              <Label>{{ t('admin.spamRescue.aiEnabled') }}</Label>
              <Switch v-model:checked="aiEnabled" />
            </div>

            <div class="space-y-2">
              <Label for="spam-rescue-context">{{ t('admin.spamRescue.aiContext') }}</Label>
              <Textarea
                id="spam-rescue-context"
                v-model="aiContext"
                class="text-sm"
                rows="6"
                :placeholder="t('admin.spamRescue.aiContextPlaceholder')"
              />
              <p class="text-xs text-muted-foreground">
                {{ t('admin.spamRescue.aiContextDescription') }}
              </p>
            </div>

            <div class="space-y-2">
              <Label for="spam-rescue-confidence">{{ t('admin.spamRescue.aiMinConfidence') }}</Label>
              <Input
                id="spam-rescue-confidence"
                v-model="aiMinConfidence"
                type="number"
                min="0"
                max="1"
                step="0.05"
                class="max-w-xs"
              />
              <p class="text-xs text-muted-foreground">
                {{ t('admin.spamRescue.aiMinConfidenceDescription') }}
              </p>
            </div>
          </div>

          <Button type="submit" :isLoading="saving">
            {{ t('globals.messages.save') }}
          </Button>
        </form>
        <Spinner v-if="isLoading" />
      </div>
    </template>
    <template #help>
      <p>{{ t('admin.spamRescue.help') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { Switch } from '@shared-ui/components/ui/switch'
import { Textarea } from '@shared-ui/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()

const isLoading = ref(true)
const saving = ref(false)

const enabled = ref(false)
const mode = ref('shadow')
const ruleReplyToThread = ref(true)
const ruleEcommerceCustomer = ref(true)
const ruleOrderNumber = ref(false)
const orderNumberPattern = ref('')
const trustedEmails = ref('')
const trustedDomains = ref('')
const aiEnabled = ref(false)
const aiContext = ref('')
const aiMinConfidence = ref(0.9)

const toLines = (list) => (Array.isArray(list) ? list.join('\n') : '')
const fromLines = (text) =>
  text
    .split('\n')
    .map((s) => s.trim())
    .filter((s) => s.length > 0)

const showToast = (description, variant) =>
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, variant ? { variant, description } : { description })

onMounted(async () => {
  try {
    const res = await api.getSettings('spam-rescue')
    const d = res.data?.data || {}
    enabled.value = !!d['spam_rescue.enabled']
    mode.value = d['spam_rescue.mode'] || 'shadow'
    ruleReplyToThread.value = !!d['spam_rescue.rule_reply_to_thread']
    ruleEcommerceCustomer.value = !!d['spam_rescue.rule_ecommerce_customer']
    ruleOrderNumber.value = !!d['spam_rescue.rule_order_number']
    orderNumberPattern.value = d['spam_rescue.order_number_pattern'] || ''
    trustedEmails.value = toLines(d['spam_rescue.trusted_emails'])
    trustedDomains.value = toLines(d['spam_rescue.trusted_domains'])
    aiEnabled.value = !!d['spam_rescue.ai_enabled']
    aiContext.value = d['spam_rescue.ai_context'] || ''
    aiMinConfidence.value = d['spam_rescue.ai_min_confidence'] ?? 0.9
  } catch (err) {
    showToast(handleHTTPError(err).message, 'destructive')
  } finally {
    isLoading.value = false
  }
})

const onSubmit = async () => {
  saving.value = true
  try {
    await api.updateSettings('spam-rescue', {
      'spam_rescue.enabled': enabled.value,
      'spam_rescue.mode': mode.value,
      'spam_rescue.rule_reply_to_thread': ruleReplyToThread.value,
      'spam_rescue.rule_ecommerce_customer': ruleEcommerceCustomer.value,
      'spam_rescue.rule_order_number': ruleOrderNumber.value,
      'spam_rescue.order_number_pattern': orderNumberPattern.value.trim(),
      'spam_rescue.trusted_emails': fromLines(trustedEmails.value),
      'spam_rescue.trusted_domains': fromLines(trustedDomains.value),
      'spam_rescue.ai_enabled': aiEnabled.value,
      'spam_rescue.ai_context': aiContext.value.trim(),
      'spam_rescue.ai_min_confidence': parseFloat(aiMinConfidence.value) || 0
    })
    showToast(t('globals.messages.savedSuccessfully'))
  } catch (err) {
    showToast(handleHTTPError(err).message, 'destructive')
  } finally {
    saving.value = false
  }
}
</script>
