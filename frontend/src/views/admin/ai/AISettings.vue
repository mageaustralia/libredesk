<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { toast } from 'vue-sonner'
import api from '@/api'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import Spinner from '@/components/ui/spinner/Spinner.vue'
import { Bot, CheckCircle, AlertCircle, RefreshCw, Inbox } from 'lucide-vue-next'

const providers = ref([])
const availableModels = ref([])
const loading = ref(true)
const saving = ref(false)
const testing = ref(false)

// Form state
const openaiApiKey = ref('')
const openrouterApiKey = ref('')
const openrouterModel = ref('anthropic/claude-3-haiku')
const defaultProvider = ref('openai')

// RAG AI Settings
const systemPrompt = ref('')
const maxContextChunks = ref(5)
const similarityThreshold = ref(0.7)
const savingRAG = ref(false)

// External Search settings
const externalSearchEnabled = ref(false)
const externalSearchURL = ref('')
const externalSearchMaxResults = ref(3)
const externalSearchEndpoints = ref('')
const externalSearchHeaders = ref('')

// Inbox scope
const inboxes = ref([])
const selectedInboxId = ref('global')
const knowledgeSources = ref([])
const selectedKnowledgeSourceIds = ref([])
const isInboxScope = computed(() => selectedInboxId.value !== 'global')
const hasInboxOverride = ref(false)

const hasOpenAIKey = computed(() => {
  const p = providers.value.find(p => p.provider === 'openai')
  return p?.has_api_key || false
})

const hasOpenRouterKey = computed(() => {
  const p = providers.value.find(p => p.provider === 'openrouter')
  return p?.has_api_key || false
})

const currentDefaultProvider = computed(() => {
  const p = providers.value.find(p => p.is_default)
  return p?.provider || 'openai'
})

async function fetchProviders() {
  try {
    const res = await api.getAIProviders()
    providers.value = res.data.data || []

    // Set default provider
    const defaultP = providers.value.find(p => p.is_default)
    if (defaultP) {
      defaultProvider.value = defaultP.provider
    }

    // Get current OpenRouter model
    const openrouter = providers.value.find(p => p.provider === 'openrouter')
    if (openrouter?.model) {
      openrouterModel.value = openrouter.model
    }
  } catch (err) {
    console.error('Error fetching providers:', err)
    toast.error('Failed to load AI providers')
  }
}

async function fetchModels() {
  try {
    const res = await api.getAvailableModels()
    availableModels.value = res.data.data || []
  } catch (err) {
    console.error('Error fetching models:', err)
  }
}

async function fetchInboxes() {
  try {
    const res = await api.getInboxes()
    inboxes.value = res.data.data || []
  } catch (err) {
    console.error('Error fetching inboxes:', err)
  }
}

async function fetchKnowledgeSources() {
  try {
    const res = await api.getRAGSources()
    knowledgeSources.value = res.data.data || []
  } catch (err) {
    console.error('Error fetching knowledge sources:', err)
  }
}

async function saveOpenAI() {
  if (!openaiApiKey.value) {
    toast.error('Please enter an API key')
    return
  }

  saving.value = true
  try {
    await api.updateAIProvider({
      provider: 'openai',
      api_key: openaiApiKey.value,
      model: ''
    })
    toast.success('OpenAI API key saved')
    openaiApiKey.value = ''
    await fetchProviders()
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to save')
  } finally {
    saving.value = false
  }
}

async function saveOpenRouter() {
  if (!openrouterApiKey.value && !hasOpenRouterKey.value) {
    toast.error('Please enter an API key')
    return
  }

  saving.value = true
  try {
    await api.updateAIProvider({
      provider: 'openrouter',
      api_key: openrouterApiKey.value || '',
      model: openrouterModel.value
    })
    toast.success('OpenRouter settings saved')
    openrouterApiKey.value = ''
    await fetchProviders()
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to save')
  } finally {
    saving.value = false
  }
}

async function setDefaultProvider(provider) {
  try {
    await api.setDefaultAIProvider({ provider })
    toast.success(`${provider === 'openai' ? 'OpenAI' : 'OpenRouter'} set as default`)
    await fetchProviders()
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to set default')
  }
}

async function testProvider(provider) {
  const config = {
    provider,
    api_key: provider === 'openai' ? openaiApiKey.value : openrouterApiKey.value,
    model: provider === 'openrouter' ? openrouterModel.value : ''
  }

  testing.value = true
  try {
    await api.testAIProvider(config)
    toast.success('Connection successful!')
  } catch (err) {
    toast.error(err.response?.data?.message || 'Connection failed')
  } finally {
    testing.value = false
  }
}

function applySettingsToForm(data, isGlobal = true) {
  if (isGlobal) {
    systemPrompt.value = data['ai.system_prompt'] || ''
    maxContextChunks.value = data['ai.max_context_chunks'] || 5
    similarityThreshold.value = data['ai.similarity_threshold'] || 0.7
    externalSearchEnabled.value = data['ai.external_search_enabled'] || false
    externalSearchURL.value = data['ai.external_search_url'] || ''
    externalSearchMaxResults.value = data['ai.external_search_max_results'] || 3
    externalSearchEndpoints.value = data['ai.external_search_endpoints'] || ''
    externalSearchHeaders.value = data['ai.external_search_headers'] || ''
    selectedKnowledgeSourceIds.value = []
  } else {
    systemPrompt.value = data.system_prompt || ''
    maxContextChunks.value = data.max_context_chunks || 5
    similarityThreshold.value = data.similarity_threshold || 0.7
    externalSearchEnabled.value = data.external_search_enabled || false
    externalSearchURL.value = data.external_search_url || ''
    externalSearchMaxResults.value = data.external_search_max_results || 3
    externalSearchEndpoints.value = data.external_search_endpoints || ''
    externalSearchHeaders.value = data.external_search_headers || ''
    const ksIds = data.knowledge_source_ids || []
    selectedKnowledgeSourceIds.value = Array.isArray(ksIds) ? ksIds.map(String) : []
  }
}

async function fetchAISettings() {
  try {
    const res = await api.getAISettings()
    const data = res.data.data || {}
    applySettingsToForm(data, true)
    hasInboxOverride.value = false
  } catch (err) {
    console.error('Error fetching AI settings:', err)
  }
}

async function fetchInboxAISettings(inboxId) {
  try {
    const res = await api.getInboxAISettings(inboxId)
    const data = res.data.data || {}
    // If id > 0, inbox has its own override
    if (data.id > 0) {
      hasInboxOverride.value = true
      applySettingsToForm(data, false)
    } else {
      // No override — load global settings as starting point
      hasInboxOverride.value = false
      await fetchAISettings()
    }
  } catch (err) {
    console.error('Error fetching inbox AI settings:', err)
    hasInboxOverride.value = false
    await fetchAISettings()
  }
}

async function saveAISettings() {
  savingRAG.value = true
  try {
    if (isInboxScope.value) {
      await api.updateInboxAISettings(selectedInboxId.value, {
        system_prompt: systemPrompt.value,
        max_context_chunks: parseInt(maxContextChunks.value) || 5,
        similarity_threshold: parseFloat(similarityThreshold.value) || 0.7,
        external_search_enabled: externalSearchEnabled.value,
        external_search_url: externalSearchURL.value,
        external_search_max_results: parseInt(externalSearchMaxResults.value) || 3,
        external_search_endpoints: externalSearchEndpoints.value,
        external_search_headers: externalSearchHeaders.value,
        knowledge_source_ids: selectedKnowledgeSourceIds.value.map(Number)
      })
      hasInboxOverride.value = true
      const inboxName = inboxes.value.find(i => i.id == selectedInboxId.value)?.name || ''
      toast.success(`AI settings saved for ${inboxName}`)
    } else {
      await api.updateAISettings({
        'ai.system_prompt': systemPrompt.value,
        'ai.max_context_chunks': parseInt(maxContextChunks.value) || 5,
        'ai.similarity_threshold': parseFloat(similarityThreshold.value) || 0.7,
        'ai.external_search_enabled': externalSearchEnabled.value,
        'ai.external_search_url': externalSearchURL.value,
        'ai.external_search_max_results': parseInt(externalSearchMaxResults.value) || 3,
        'ai.external_search_endpoints': externalSearchEndpoints.value,
        'ai.external_search_headers': externalSearchHeaders.value
      })
      toast.success('Global AI settings saved')
    }
  } catch (err) {
    toast.error(err.response?.data?.message || 'Failed to save AI settings')
  } finally {
    savingRAG.value = false
  }
}

async function resetToGlobal() {
  if (!confirm('Remove inbox-specific settings and fall back to global defaults?')) return
  try {
    await api.deleteInboxAISettings(selectedInboxId.value)
    hasInboxOverride.value = false
    await fetchAISettings()
    toast.success('Inbox settings removed, now using global defaults')
  } catch (err) {
    toast.error('Failed to reset settings')
  }
}

// Watch for inbox scope changes
watch(selectedInboxId, async (newVal) => {
  if (newVal === 'global') {
    await fetchAISettings()
  } else {
    await fetchInboxAISettings(newVal)
  }
})

onMounted(async () => {
  loading.value = true
  await Promise.all([fetchProviders(), fetchModels(), fetchAISettings(), fetchInboxes(), fetchKnowledgeSources()])
  loading.value = false
})
</script>

<template>
  <AdminPageWithHelp>
    <template #content>
      <div v-if="loading" class="flex items-center justify-center py-12">
        <Spinner />
      </div>

      <div v-else class="space-y-6">
        <!-- OpenAI Card -->
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Bot class="h-5 w-5" />
                <CardTitle>OpenAI</CardTitle>
              </div>
              <div class="flex items-center gap-2">
                <Badge v-if="hasOpenAIKey" class="bg-green-100 text-green-800">
                  <CheckCircle class="h-3 w-3 mr-1" />
                  Configured
                </Badge>
                <Badge v-else variant="secondary">
                  <AlertCircle class="h-3 w-3 mr-1" />
                  Not configured
                </Badge>
                <Badge v-if="currentDefaultProvider === 'openai'">
                  Default
                </Badge>
              </div>
            </div>
            <CardDescription>
              Use OpenAI's GPT-4o-mini model for AI assistance.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="space-y-2">
              <Label for="openai-key">API Key</Label>
              <Input
                id="openai-key"
                v-model="openaiApiKey"
                type="password"
                :placeholder="hasOpenAIKey ? '********' : 'sk-...'"
              />
              <p class="text-xs text-muted-foreground">
                Get your API key from <a href="https://platform.openai.com/api-keys" target="_blank" class="underline">OpenAI Dashboard</a>
              </p>
            </div>
            <div class="flex gap-2">
              <Button @click="saveOpenAI" :disabled="saving || !openaiApiKey">
                Save
              </Button>
              <Button variant="outline" @click="testProvider('openai')" :disabled="testing">
                <RefreshCw v-if="testing" class="h-4 w-4 mr-2 animate-spin" />
                Test Connection
              </Button>
              <Button
                v-if="currentDefaultProvider !== 'openai' && hasOpenAIKey"
                variant="secondary"
                @click="setDefaultProvider('openai')"
              >
                Set as Default
              </Button>
            </div>
          </CardContent>
        </Card>

        <!-- OpenRouter Card -->
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Bot class="h-5 w-5" />
                <CardTitle>OpenRouter</CardTitle>
              </div>
              <div class="flex items-center gap-2">
                <Badge v-if="hasOpenRouterKey" class="bg-green-100 text-green-800">
                  <CheckCircle class="h-3 w-3 mr-1" />
                  Configured
                </Badge>
                <Badge v-else variant="secondary">
                  <AlertCircle class="h-3 w-3 mr-1" />
                  Not configured
                </Badge>
                <Badge v-if="currentDefaultProvider === 'openrouter'">
                  Default
                </Badge>
              </div>
            </div>
            <CardDescription>
              Access multiple AI models through OpenRouter - Claude, GPT-4, Gemini, Llama, and more.
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="space-y-2">
              <Label for="openrouter-key">API Key</Label>
              <Input
                id="openrouter-key"
                v-model="openrouterApiKey"
                type="password"
                :placeholder="hasOpenRouterKey ? '********' : 'sk-or-...'"
              />
              <p class="text-xs text-muted-foreground">
                Get your API key from <a href="https://openrouter.ai/keys" target="_blank" class="underline">OpenRouter Dashboard</a>
              </p>
            </div>

            <div class="space-y-2">
              <Label for="openrouter-model">Model</Label>
              <Select v-model="openrouterModel">
                <SelectTrigger>
                  <SelectValue :placeholder="openrouterModel" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="model in availableModels"
                    :key="model"
                    :value="model"
                  >
                    {{ model }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <p class="text-xs text-muted-foreground">
                Choose the AI model to use. Different models have different capabilities and pricing.
              </p>
            </div>

            <div class="flex gap-2">
              <Button @click="saveOpenRouter" :disabled="saving">
                Save
              </Button>
              <Button variant="outline" @click="testProvider('openrouter')" :disabled="testing || !hasOpenRouterKey">
                <RefreshCw v-if="testing" class="h-4 w-4 mr-2 animate-spin" />
                Test Connection
              </Button>
              <Button
                v-if="currentDefaultProvider !== 'openrouter' && hasOpenRouterKey"
                variant="secondary"
                @click="setDefaultProvider('openrouter')"
              >
                Set as Default
              </Button>
            </div>
          </CardContent>
        </Card>

        <!-- RAG AI Assistant Settings -->
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Bot class="h-5 w-5" />
                <CardTitle>AI Assistant Settings</CardTitle>
              </div>
              <!-- Inbox scope selector -->
              <div class="flex items-center gap-2">
                <Inbox class="h-4 w-4 text-muted-foreground" />
                <Select v-model="selectedInboxId">
                  <SelectTrigger class="w-[220px]">
                    <SelectValue placeholder="Global (all inboxes)" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="global">Global (all inboxes)</SelectItem>
                    <SelectItem
                      v-for="inbox in inboxes"
                      :key="inbox.id"
                      :value="String(inbox.id)"
                    >
                      {{ inbox.name }}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <CardDescription>
              <template v-if="isInboxScope">
                <span class="text-amber-600 font-medium" v-if="hasInboxOverride">
                  Custom settings for this inbox.
                </span>
                <span v-else>
                  Using global defaults. Save to create inbox-specific settings.
                </span>
              </template>
              <template v-else>
                Configure the default system prompt and RAG settings. These apply to all inboxes unless overridden.
              </template>
            </CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="space-y-2">
              <Label for="system-prompt">System Prompt</Label>
              <Textarea
                id="system-prompt"
                v-model="systemPrompt"
                rows="8"
                placeholder="You are a helpful customer support assistant for {{site_name}}..."
                class="font-mono text-sm"
              />
              <p class="text-xs text-muted-foreground">
                The system prompt sent to the AI when generating responses. Use <code v-pre>{{site_name}}</code>, <code v-pre>{{context}}</code>, <code v-pre>{{macros}}</code>, <code v-pre>{{enquiry}}</code>, and <code v-pre>{{external_search_results}}</code> as placeholders.
              </p>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div class="space-y-2">
                <Label for="max-chunks">Max Context Chunks</Label>
                <Input
                  id="max-chunks"
                  v-model="maxContextChunks"
                  type="number"
                  min="1"
                  max="20"
                />
                <p class="text-xs text-muted-foreground">
                  Maximum number of knowledge base chunks to include as context (default: 5).
                </p>
              </div>

              <div class="space-y-2">
                <Label for="similarity">Similarity Threshold</Label>
                <Input
                  id="similarity"
                  v-model="similarityThreshold"
                  type="number"
                  min="0"
                  max="1"
                  step="0.05"
                />
                <p class="text-xs text-muted-foreground">
                  Minimum similarity score for knowledge base matches (0-1, default: 0.7).
                </p>
              </div>
            </div>

            <!-- Knowledge Sources (inbox scope only) -->
            <div v-if="isInboxScope" class="border-t pt-4 mt-4 space-y-4">
              <div>
                <Label>Knowledge Sources</Label>
                <p class="text-xs text-muted-foreground mb-2">
                  Select which knowledge sources this inbox should use. If none selected, all sources are searched.
                </p>
                <div class="space-y-2">
                  <label
                    v-for="source in knowledgeSources"
                    :key="source.id"
                    class="flex items-center gap-2 p-2 rounded hover:bg-muted cursor-pointer"
                  >
                    <input
                      type="checkbox"
                      :value="String(source.id)"
                      v-model="selectedKnowledgeSourceIds"
                      class="rounded border-gray-300"
                    />
                    <span class="text-sm">{{ source.name }}</span>
                    <Badge variant="secondary" class="text-xs">{{ source.source_type }}</Badge>
                  </label>
                  <p v-if="knowledgeSources.length === 0" class="text-sm text-muted-foreground italic">
                    No knowledge sources configured. Add them in Knowledge Sources settings.
                  </p>
                </div>
              </div>
            </div>

            <!-- External Search Integration -->
            <div class="border-t pt-4 mt-4 space-y-4">
              <div class="flex items-center justify-between">
                <div>
                  <Label>External Search Integration</Label>
                  <p class="text-xs text-muted-foreground">
                    Enable live search from an external API (e.g. Meilisearch) when generating AI responses. The AI will classify the customer query and search relevant endpoints.
                  </p>
                </div>
                <Switch v-model:checked="externalSearchEnabled" />
              </div>

              <div v-if="externalSearchEnabled" class="space-y-4">
                <div class="space-y-2">
                  <Label for="search-url">Search Base URL</Label>
                  <Input
                    id="search-url"
                    v-model="externalSearchURL"
                    type="text"
                    placeholder="https://example.com/search-api"
                  />
                  <p class="text-xs text-muted-foreground">
                    Base URL for the search API. Endpoint paths below are appended to this.
                  </p>
                </div>

                <div class="space-y-2">
                  <Label for="search-endpoints">Search Endpoints (JSON)</Label>
                  <Textarea
                    id="search-endpoints"
                    v-model="externalSearchEndpoints"
                    rows="4"
                    placeholder='{"product": "/indexes/products/search", "category": "/indexes/categories/search", "faq": "/indexes/faqs/search"}'
                    class="font-mono text-sm"
                  />
                  <p class="text-xs text-muted-foreground">
                    JSON object mapping intent types to URL path suffixes. Supported types: <code>product</code>, <code>category</code>, <code>faq</code>.
                  </p>
                </div>

                <div class="space-y-2">
                  <Label for="search-headers">Custom HTTP Headers (JSON)</Label>
                  <Textarea
                    id="search-headers"
                    v-model="externalSearchHeaders"
                    rows="3"
                    placeholder='{"User-Agent": "Mozilla/5.0...", "Referer": "https://example.com/"}'
                    class="font-mono text-sm"
                  />
                  <p class="text-xs text-muted-foreground">
                    Optional JSON object of custom HTTP headers to send with search requests.
                  </p>
                </div>

                <div class="space-y-2">
                  <Label for="search-max">Max Results Per Endpoint</Label>
                  <Input
                    id="search-max"
                    v-model="externalSearchMaxResults"
                    type="number"
                    min="1"
                    max="10"
                    class="max-w-[200px]"
                  />
                  <p class="text-xs text-muted-foreground">
                    Maximum number of results to fetch from each search endpoint (default: 3).
                  </p>
                </div>
              </div>
            </div>

            <div class="flex gap-2">
              <Button @click="saveAISettings" :disabled="savingRAG">
                {{ savingRAG ? 'Saving...' : (isInboxScope ? 'Save Inbox Settings' : 'Save Global Settings') }}
              </Button>
              <Button
                v-if="isInboxScope && hasInboxOverride"
                variant="outline"
                @click="resetToGlobal"
              >
                Reset to Global
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>
    <template #help>
      <h4 class="font-medium mb-2">AI Settings</h4>
      <p class="text-sm text-muted-foreground mb-4">
        Configure AI providers for response assistance. You can use OpenAI directly or OpenRouter for access to multiple models.
      </p>
      <h4 class="font-medium mb-2">Per-Inbox Settings</h4>
      <p class="text-sm text-muted-foreground mb-4">
        Use the inbox dropdown to configure different AI settings per inbox. Each inbox can have its own system prompt, knowledge sources, and external search configuration.
      </p>
      <h4 class="font-medium mb-2">How AI Assist Works</h4>
      <p class="text-sm text-muted-foreground">
        When composing replies, select text and click the AI button in the toolbar to rewrite it.
        Choose from options like "Make Friendly", "Make Professional", "Add Empathy", etc.
      </p>
    </template>
  </AdminPageWithHelp>
</template>
