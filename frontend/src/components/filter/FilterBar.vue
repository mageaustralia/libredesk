<script setup>
import { ref, computed } from 'vue'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { Plus, X } from 'lucide-vue-next'
import FilterFieldPicker from './FilterFieldPicker.vue'
import FilterPill from './FilterPill.vue'

const DATE_FIELDS = [
  'created_at',
  'last_message_at',
  'last_interaction_at',
  'waiting_since',
  'next_sla_deadline_at',
  'closed_at',
  'resolved_at'
]

const props = defineProps({
  fields: {
    type: Array,
    required: true
  },
  modelValue: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue'])

const pickerOpen = ref(false)
const newestFieldKey = ref(null)

const activeFieldKeys = computed(() => {
  return props.modelValue.map((f) => f.field)
})

const availableFields = computed(() => {
  const active = new Set(activeFieldKeys.value)
  return props.fields.filter((f) => !active.has(f.key))
})

function isDateField(fieldKey) {
  return DATE_FIELDS.includes(fieldKey)
}

function getFieldDef(fieldKey) {
  return props.fields.find((f) => f.key === fieldKey) || { key: fieldKey, label: fieldKey }
}

function handleFieldSelect(field) {
  pickerOpen.value = false
  if (activeFieldKeys.value.includes(field.key)) return

  newestFieldKey.value = field.key

  const newFilter = {
    field: field.key,
    operator: isDateField(field.key) ? 'relative_date' : (field.key === 'tags' ? 'contains' : 'in'),
    value: isDateField(field.key) ? 'last_7_days' : '[]',
    model: field.model || ''
  }
  emit('update:modelValue', [...props.modelValue, newFilter])
}

function handleFilterUpdate(index, filter) {
  const updated = [...props.modelValue]
  updated[index] = filter
  newestFieldKey.value = null
  emit('update:modelValue', updated)
}

function handleFilterRemove(index) {
  newestFieldKey.value = null
  const updated = props.modelValue.filter((_, i) => i !== index)
  emit('update:modelValue', updated)
}

function handleClearAll() {
  newestFieldKey.value = null
  emit('update:modelValue', [])
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-1.5" v-if="fields.length > 0">
    <!-- Active filter pills -->
    <FilterPill
      v-for="(filter, index) in modelValue"
      :key="filter.field"
      :field="getFieldDef(filter.field)"
      :model-value="filter"
      :is-date-field="isDateField(filter.field)"
      :auto-open="filter.field === newestFieldKey && !isDateField(filter.field)"
      @update:model-value="handleFilterUpdate(index, $event)"
      @remove="handleFilterRemove(index)"
    />

    <!-- Add filter button -->
    <Popover v-model:open="pickerOpen" v-if="availableFields.length > 0">
      <PopoverTrigger as-child>
        <Button variant="ghost" size="xs" class="gap-1 text-muted-foreground">
          <Plus class="h-3.5 w-3.5" />
          Filter
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" :side-offset="4" class="p-0 w-auto">
        <FilterFieldPicker
          :fields="availableFields"
          :active-field-keys="activeFieldKeys"
          @select="handleFieldSelect"
          @close="pickerOpen = false"
        />
      </PopoverContent>
    </Popover>

    <!-- Clear all -->
    <Button
      v-if="modelValue.length > 0"
      variant="ghost"
      size="xs"
      class="gap-1 text-muted-foreground"
      @click="handleClearAll"
    >
      <X class="h-3.5 w-3.5" />
      Clear all
    </Button>
  </div>
</template>
