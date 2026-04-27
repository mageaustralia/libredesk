<template>
  <form @submit="onSmtpSubmit" class="space-y-6">
    <FormField name="enabled" v-slot="{ value, handleChange }">
      <FormItem>
        <FormControl>
          <div class="flex items-center space-x-2">
            <Checkbox :checked="value" @update:checked="handleChange" />
            <Label>{{ $t('globals.terms.enabled') }}</Label>
          </div>
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <div class="grid gap-6 md:grid-cols-2">
      <FormField v-slot="{ componentField }" name="host">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.smtpHost') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="smtp.gmail.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="port">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.smtpPort') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="587" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="grid gap-6 md:grid-cols-2">
      <FormField v-slot="{ componentField }" name="username">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.username') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="admin@yourcompany.com" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="password">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.password') }}</FormLabel>
          <FormControl>
            <Input type="password" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="grid gap-6 md:grid-cols-2">
      <FormField v-slot="{ componentField }" name="auth_protocol">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.authProtocol') }}</FormLabel>
          <FormControl>
            <Select v-bind="componentField" v-model="componentField.modelValue">
              <SelectTrigger>
                <SelectValue :placeholder="t('admin.inbox.authProtocol.description')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="plain">Plain</SelectItem>
                  <SelectItem value="login">Login</SelectItem>
                  <SelectItem value="cram">CRAM-MD5</SelectItem>
                  <SelectItem value="none">None</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="tls_type">
        <FormItem>
          <FormLabel>TLS</FormLabel>
          <FormControl>
            <Select v-bind="componentField" v-model="componentField.modelValue">
              <SelectTrigger>
                <SelectValue :placeholder="t('globals.messages.selectTLS')" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value="none">Off</SelectItem>
                  <SelectItem value="tls">SSL/TLS</SelectItem>
                  <SelectItem value="starttls">STARTTLS</SelectItem>
                </SelectGroup>
              </SelectContent>
            </Select>
          </FormControl>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="grid gap-6 md:grid-cols-2">
      <FormField v-slot="{ componentField }" name="email_address">
        <FormItem>
          <FormLabel>{{ $t('globals.terms.fromEmailAddress') }}</FormLabel>
          <FormControl>
            <Input
              type="text"
              :placeholder="t('admin.inbox.fromEmailAddress.placeholder')"
              v-bind="componentField"
            />
          </FormControl>
          <FormMessage />
          <FormDescription> {{ $t('admin.inbox.fromEmailAddress.description') }}</FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="hello_hostname">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.heloHostname') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="" v-bind="componentField" />
          </FormControl>
          <FormDescription>
            {{ $t('admin.inbox.heloHostname.description') }}
          </FormDescription>
          <FormMessage />
        </FormItem>
      </FormField>
    </div>

    <div class="grid gap-6 md:grid-cols-2">
      <FormField v-slot="{ componentField }" name="max_conns">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.maxConnections') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="2" v-bind="componentField" />
          </FormControl>
          <FormMessage />
          <FormDescription>{{ $t('admin.inbox.maxConnections.description') }} </FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="max_msg_retries">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.maxRetries') }}</FormLabel>
          <FormControl>
            <Input type="number" placeholder="3" v-bind="componentField" />
          </FormControl>
          <FormMessage />
          <FormDescription> {{ $t('admin.inbox.maxRetries.description') }} </FormDescription>
        </FormItem>
      </FormField>
    </div>

    <div class="grid gap-6 md:grid-cols-2">
      <FormField v-slot="{ componentField }" name="idle_timeout">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.idleTimeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="15s" v-bind="componentField" />
          </FormControl>
          <FormMessage />
          <FormDescription>
            {{ $t('admin.inbox.idleTimeout.description') }}
          </FormDescription>
        </FormItem>
      </FormField>

      <FormField v-slot="{ componentField }" name="wait_timeout">
        <FormItem>
          <FormLabel>{{ $t('admin.inbox.waitTimeout') }}</FormLabel>
          <FormControl>
            <Input type="text" placeholder="5s" v-bind="componentField" />
          </FormControl>
          <FormMessage />
          <FormDescription>
            {{ $t('admin.inbox.waitTimeout.description') }}
          </FormDescription>
        </FormItem>
      </FormField>
    </div>

    <FormField v-slot="{ componentField, handleChange }" name="tls_skip_verify">
      <FormItem>
        <SwitchField
          :title="$t('admin.inbox.skipTLSVerification')"
          :description="$t('admin.inbox.skipTLSVerification.description')"
          :checked="componentField.modelValue"
          @update:checked="handleChange"
        />
      </FormItem>
    </FormField>

    <Button type="submit" :isLoading="isLoading"> {{ submitLabel }} </Button>

    <!-- Test Connection Section -->
    <div class="border-t pt-6 mt-6 space-y-4">
      <h3 class="text-base font-semibold">{{ $t('admin.inbox.testConnection') }}</h3>
      <div class="space-y-3">
        <div class="flex items-center gap-2">
          <Input
            v-model="testEmail"
            type="email"
            :placeholder="$t('admin.inbox.testConnection.emailPlaceholder')"
            class="flex-1 max-w-xs"
          />
          <Button
            type="button"
            variant="outline"
            @click="runTest"
            :disabled="isTesting || !testEmail"
          >
            <Loader2 v-if="isTesting" class="w-4 h-4 mr-2 animate-spin" />
            {{ isTesting ? $t('admin.inbox.testConnection.testing') : $t('admin.inbox.testConnection.test') }}
          </Button>
        </div>
        <div v-if="testLogs.length > 0">
          <div
            class="bg-muted p-3 rounded-md font-mono text-xs max-h-48 overflow-y-auto"
            :class="testSuccess === true ? 'border-green-500 border' : testSuccess === false ? 'border-red-500 border' : ''"
          >
            <div v-for="(log, index) in testLogs" :key="index" class="py-0.5">{{ log }}</div>
          </div>
        </div>
      </div>
    </div>
  </form>
</template>

<script setup>
import { watch, ref, computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription
} from '@shared-ui/components/ui/form/index.js'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select/index.js'
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { Label } from '@shared-ui/components/ui/label/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { useI18n } from 'vue-i18n'
import { Loader2 } from 'lucide-vue-next'
import api from '@/api'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const isLoading = ref(false)
const { t } = useI18n()
const props = defineProps({
  initialValues: {
    type: Object,
    required: false
  },
  submitForm: {
    type: Function,
    required: true
  },
  submitLabel: {
    type: String,
    required: false,
    default: () => ''
  }
})

const submitLabel = computed(() => {
  if (props.submitLabel) {
    return props.submitLabel
  }
  return t('globals.messages.save')
})

const smtpForm = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const onSmtpSubmit = smtpForm.handleSubmit(async (values) => {
  isLoading.value = true
  try {
    await props.submitForm(values)
  } finally {
    isLoading.value = false
  }
})

// Test connection state
const isTesting = ref(false)
const testEmail = ref('')
const testLogs = ref([])
const testSuccess = ref(null)

const runTest = async () => {
  isTesting.value = true
  testLogs.value = []
  testSuccess.value = null
  try {
    const values = smtpForm.values
    const response = await api.testEmailNotificationSettings({
      'notification.email.host': values.host,
      'notification.email.port': values.port,
      'notification.email.username': values.username,
      'notification.email.password': values.password,
      'notification.email.auth_protocol': values.auth_protocol,
      'notification.email.tls_type': values.tls_type,
      'notification.email.tls_skip_verify': values.tls_skip_verify,
      'notification.email.email_address': values.email_address,
      'notification.email.hello_hostname': values.hello_hostname,
      test_email: testEmail.value
    })
    testLogs.value = response.data.data.logs || []
    testSuccess.value = response.data.data.success
  } catch (error) {
    testLogs.value = [handleHTTPError(error).message]
    testSuccess.value = false
  } finally {
    isTesting.value = false
  }
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.initialValues,
  (newValues) => {
    smtpForm.setValues(newValues)
  },
  { deep: true, immediate: true }
)
</script>
