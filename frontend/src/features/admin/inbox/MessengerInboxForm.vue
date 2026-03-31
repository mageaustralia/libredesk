<template>
  <form @submit="onSubmit" class="space-y-6 max-w-2xl">
    <!-- Inbox Name -->
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input v-bind="componentField" placeholder="e.g. Facebook Page, Instagram DMs" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Page ID (Messenger) -->
    <FormField v-slot="{ componentField }" name="page_id" v-if="channel === 'messenger'">
      <FormItem>
        <FormLabel>Page ID</FormLabel>
        <FormControl>
          <Input v-bind="componentField" placeholder="e.g. 123456789012345" />
        </FormControl>
        <FormDescription>
          Found in your Facebook Page Settings → About → Page ID
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- IG Account ID (Instagram) -->
    <FormField v-slot="{ componentField }" name="ig_account_id" v-if="channel === 'instagram'">
      <FormItem>
        <FormLabel>Instagram Account ID</FormLabel>
        <FormControl>
          <Input v-bind="componentField" placeholder="e.g. 17841400123456789" />
        </FormControl>
        <FormDescription>
          The Instagram Business Account ID linked to your Facebook Page
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Page ID for Instagram (also needed for webhook matching) -->
    <FormField v-slot="{ componentField }" name="page_id" v-if="channel === 'instagram'">
      <FormItem>
        <FormLabel>Facebook Page ID (linked)</FormLabel>
        <FormControl>
          <Input v-bind="componentField" placeholder="e.g. 123456789012345" />
        </FormControl>
        <FormDescription>
          The Facebook Page linked to this Instagram account (used for webhook routing)
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Page Access Token -->
    <FormField v-slot="{ componentField }" name="page_access_token">
      <FormItem>
        <FormLabel>Page Access Token</FormLabel>
        <FormControl>
          <Input v-bind="componentField" type="password" placeholder="Paste your never-expiring Page Access Token" />
        </FormControl>
        <FormDescription>
          Generate a never-expiring token via the
          <a href="https://developers.facebook.com/tools/explorer/" target="_blank" class="text-primary underline">Graph API Explorer</a>
          or extend a short-lived token via the Access Token Debugger.
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- App Secret -->
    <FormField v-slot="{ componentField }" name="app_secret">
      <FormItem>
        <FormLabel>App Secret</FormLabel>
        <FormControl>
          <Input v-bind="componentField" type="password" placeholder="For webhook signature verification" />
        </FormControl>
        <FormDescription>
          Found in your Facebook App → Settings → Basic → App Secret. Used to verify webhook signatures.
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Verify Token -->
    <FormField v-slot="{ componentField }" name="verify_token">
      <FormItem>
        <FormLabel>Verify Token</FormLabel>
        <FormControl>
          <Input v-bind="componentField" placeholder="Any random string for webhook verification" />
        </FormControl>
        <FormDescription>
          A secret string you choose. Enter the same string in your Meta App's webhook configuration.
        </FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <!-- Webhook URL (read-only display) -->
    <div class="rounded-md border border-border p-4 bg-muted/50">
      <p class="text-sm font-medium mb-1">Webhook URL</p>
      <div class="flex items-center gap-2">
        <code class="text-xs bg-background px-2 py-1 rounded flex-1 overflow-x-auto">{{ webhookURL }}</code>
        <Button type="button" variant="outline" size="xs" @click="copyWebhookURL">Copy</Button>
      </div>
      <p class="text-xs text-muted-foreground mt-2">
        Configure this URL in your Meta App → Webhooks → Callback URL
      </p>
    </div>

    <!-- Auto-assign -->
    <FormField v-slot="{ value, handleChange }" name="auto_assign_on_reply">
      <FormItem class="flex items-center gap-3">
        <FormControl>
          <Switch :model-value="value" @update:model-value="handleChange" />
        </FormControl>
        <FormLabel class="!mt-0">Auto-assign conversation when agent replies</FormLabel>
      </FormItem>
    </FormField>

    <!-- Enabled (edit mode only) -->
    <FormField v-slot="{ value, handleChange }" name="enabled" v-if="initialValues?.id">
      <FormItem class="flex items-center gap-3">
        <FormControl>
          <Switch :model-value="value" @update:model-value="handleChange" />
        </FormControl>
        <FormLabel class="!mt-0">{{ $t('globals.terms.enabled') }}</FormLabel>
      </FormItem>
    </FormField>

    <Button type="submit" :disabled="isLoading">
      <Spinner v-if="isLoading" class="mr-2" />
      {{ submitLabel || (initialValues?.id ? $t('globals.messages.update') : $t('globals.messages.create')) }}
    </Button>
  </form>
</template>

<script setup>
import { computed } from 'vue'
import { useForm } from 'vee-validate'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Spinner } from '@/components/ui/spinner'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form'
import { getMessengerTypedSchema } from './messengerFormSchema'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'

const props = defineProps({
  initialValues: { type: Object, default: () => ({}) },
  submitForm: { type: Function, required: true },
  isLoading: { type: Boolean, default: false },
  submitLabel: { type: String, default: '' },
  channel: { type: String, required: true }  // 'messenger' or 'instagram'
})

const { t } = useI18n()
const emitter = useEmitter()

const form = useForm({
  validationSchema: computed(() => getMessengerTypedSchema(t)),
  initialValues: {
    name: props.initialValues?.name || '',
    page_id: props.initialValues?.page_id || props.initialValues?.config?.page_id || '',
    ig_account_id: props.initialValues?.ig_account_id || props.initialValues?.config?.ig_account_id || '',
    page_access_token: props.initialValues?.page_access_token || '',
    app_secret: props.initialValues?.app_secret || '',
    verify_token: props.initialValues?.verify_token || props.initialValues?.config?.verify_token || '',
    auto_assign_on_reply: props.initialValues?.auto_assign_on_reply || props.initialValues?.config?.auto_assign_on_reply || false,
    enabled: props.initialValues?.enabled ?? true
  }
})

const webhookURL = computed(() => {
  const baseURL = window.location.origin
  return `${baseURL}/webhooks/meta`
})

const copyWebhookURL = () => {
  navigator.clipboard.writeText(webhookURL.value)
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: 'Webhook URL copied to clipboard' })
}

const onSubmit = form.handleSubmit((values) => {
  props.submitForm(values)
})
</script>
