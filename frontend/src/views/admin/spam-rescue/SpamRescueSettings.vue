<script setup>
import { ref, onMounted } from 'vue'
import { toast } from 'vue-sonner'
import api from '@/api'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import { MailCheck } from 'lucide-vue-next'

const loading = ref(true)
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

onMounted(fetchSettings)

async function fetchSettings() {
  loading.value = true
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
    console.error('Failed to load spam rescue settings', err)
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
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
    toast.success('Spam rescue settings saved')
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
              <MailCheck class="h-5 w-5" />
              <CardTitle>Spam Rescue</CardTitle>
            </div>
            <CardDescription>
              Emails your mail provider files in its spam folder are marked as spam. Spam rescue
              keeps the legitimate ones in the inbox instead. Senders an agent has replied to
              before are always kept.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="flex items-center justify-between max-w-2xl">
              <div class="space-y-1">
                <Label>Enable spam rescue</Label>
                <p class="text-xs text-muted-foreground">
                  When off, only the always-on rule (a previous agent reply) applies.
                </p>
              </div>
              <Switch v-model:checked="enabled" />
            </div>

            <div class="space-y-2">
              <Label>Mode</Label>
              <Select v-model="mode">
                <SelectTrigger class="max-w-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="shadow">Shadow (record only, change nothing)</SelectItem>
                  <SelectItem value="live">Live (keep matches in the inbox)</SelectItem>
                </SelectContent>
              </Select>
              <p class="text-xs text-muted-foreground">
                In shadow mode, a private note on the conversation records what would have been
                rescued and why. Review those for a while before switching to live.
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Rules</CardTitle>
            <CardDescription>
              Checked in this order. The first match keeps the email; nothing here is tied to a
              particular business.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="space-y-2">
              <Label>Trusted email addresses</Label>
              <Textarea
                v-model="trustedEmails"
                class="max-w-2xl font-mono text-sm"
                rows="3"
                placeholder="partner@example.com"
              />
              <p class="text-xs text-muted-foreground">One address per line.</p>
            </div>

            <div class="space-y-2">
              <Label>Trusted domains</Label>
              <Textarea
                v-model="trustedDomains"
                class="max-w-2xl font-mono text-sm"
                rows="3"
                placeholder="example.org"
              />
              <p class="text-xs text-muted-foreground">
                One per line. Includes subdomains.
              </p>
            </div>

            <div class="flex items-center justify-between max-w-2xl">
              <div class="space-y-1">
                <Label>Reply to something we sent</Label>
                <p class="text-xs text-muted-foreground">
                  The email's reply headers point at a message an agent sent.
                </p>
              </div>
              <Switch v-model:checked="ruleReplyToThread" />
            </div>

            <div class="flex items-center justify-between max-w-2xl">
              <div class="space-y-1">
                <Label>Sender is a known customer</Label>
                <p class="text-xs text-muted-foreground">
                  Looked up in the connected shop (Admin &rarr; Ecommerce). Skipped if no shop is
                  connected.
                </p>
              </div>
              <Switch v-model:checked="ruleEcommerceCustomer" />
            </div>

            <div class="space-y-3 max-w-2xl">
              <div class="flex items-center justify-between">
                <div class="space-y-1">
                  <Label>Mentions an order the sender placed</Label>
                  <p class="text-xs text-muted-foreground">
                    A number matching the pattern below must exist in the shop and belong to the
                    sender's email address.
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
                Regular expression for your order-number format. At most three candidates per
                email are looked up.
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>AI check</CardTitle>
            <CardDescription>
              For emails none of the rules matched, ask the default AI provider (Admin &rarr; AI)
              whether the message looks legitimate. It can only keep an email in the inbox, never
              delete or hide one. Failures and timeouts leave the email as spam.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="flex items-center justify-between max-w-2xl">
              <Label>Use AI for the remaining emails</Label>
              <Switch v-model:checked="aiEnabled" />
            </div>

            <div class="space-y-2">
              <Label>About your business</Label>
              <Textarea
                v-model="aiContext"
                class="max-w-2xl text-sm"
                rows="6"
                placeholder="Describe what you sell or do, and which kinds of enquiries, partnerships or press are relevant to you."
              />
              <p class="text-xs text-muted-foreground">
                The AI uses this to judge relevance. Be specific: topics, products, and the kind of
                sender you want to hear from.
              </p>
            </div>

            <div class="space-y-2">
              <Label for="min-confidence">Minimum confidence (0&ndash;1)</Label>
              <Input
                id="min-confidence"
                v-model="aiMinConfidence"
                type="number"
                min="0"
                max="1"
                step="0.05"
                class="max-w-xs"
              />
              <p class="text-xs text-muted-foreground">
                The AI must call the email legitimate with at least this confidence. Higher is
                safer.
              </p>
            </div>
          </CardContent>
        </Card>

        <Button @click="saveSettings" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save' }}
        </Button>
      </div>
    </template>
    <template #help>
      <h4 class="font-medium mb-2">How it works</h4>
      <ul class="text-sm text-muted-foreground list-disc list-inside space-y-1">
        <li>Applies only to emails that arrive through the mail provider's spam or junk folder</li>
        <li>Rescued emails stay Open and are moved back to the provider's inbox</li>
        <li>Each decision is recorded as a private note on the conversation</li>
        <li>Rules run before the AI, so most legitimate mail never reaches it</li>
      </ul>
      <h4 class="font-medium mt-4 mb-2">Tips</h4>
      <ul class="text-sm text-muted-foreground list-disc list-inside space-y-1">
        <li>Start in shadow mode and check the notes for a week or two</li>
        <li>
          Notes are removed by the activity purge on the Trash &amp; Cleanup page, so raise or
          disable it while reviewing
        </li>
      </ul>
    </template>
  </AdminPageWithHelp>
</template>
