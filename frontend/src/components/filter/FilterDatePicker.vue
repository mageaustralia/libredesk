<script setup>
import { ref, computed, watch } from 'vue'
import { Button } from '@/components/ui/button'

const props = defineProps({
  field: {
    type: Object,
    required: true
  },
  modelValue: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'remove'])

const presets = [
  { label: 'Today', value: 'today' },
  { label: 'Yesterday', value: 'yesterday' },
  { label: 'Last 7 days', value: 'last_7_days' },
  { label: 'Last 30 days', value: 'last_30_days' },
  { label: 'This month', value: 'this_month' }
]

const showCustomRange = ref(false)
const customStart = ref('')
const customEnd = ref('')
const activePreset = ref(null)

function initFromModelValue() {
  if (props.modelValue) {
    if (props.modelValue.operator === 'relative_date') {
      activePreset.value = props.modelValue.value
      showCustomRange.value = false
    } else if (props.modelValue.operator === 'between') {
      showCustomRange.value = true
      activePreset.value = null
      const parts = (props.modelValue.value || '').split(',')
      customStart.value = parts[0] || ''
      customEnd.value = parts[1] || ''
    }
  } else {
    activePreset.value = null
    showCustomRange.value = false
    customStart.value = ''
    customEnd.value = ''
  }
}

initFromModelValue()
watch(() => props.modelValue, initFromModelValue, { deep: true })

function selectPreset(preset) {
  activePreset.value = preset.value
  showCustomRange.value = false
  emit('update:modelValue', {
    field: props.field.key,
    operator: 'relative_date',
    value: preset.value,
    model: props.field.model || ''
  })
}

function toggleCustomRange() {
  showCustomRange.value = true
  activePreset.value = null
}

function applyCustomRange() {
  if (!customStart.value || !customEnd.value) return
  emit('update:modelValue', {
    field: props.field.key,
    operator: 'between',
    value: `${customStart.value},${customEnd.value}`,
    model: props.field.model || ''
  })
}

function handleClear() {
  activePreset.value = null
  showCustomRange.value = false
  customStart.value = ''
  customEnd.value = ''
  emit('remove')
}
</script>

<template>
  <div class="w-64">
    <!-- Preset buttons -->
    <div class="flex flex-col gap-0.5 px-1">
      <button
        v-for="preset in presets"
        :key="preset.value"
        class="w-full text-left px-2 py-1.5 rounded-sm text-sm transition-colors"
        :class="activePreset === preset.value
          ? 'bg-accent text-accent-foreground font-medium'
          : 'hover:bg-accent'"
        @click="selectPreset(preset)"
      >
        {{ preset.label }}
      </button>

      <button
        class="w-full text-left px-2 py-1.5 rounded-sm text-sm transition-colors"
        :class="showCustomRange
          ? 'bg-accent text-accent-foreground font-medium'
          : 'hover:bg-accent'"
        @click="toggleCustomRange"
      >
        Custom range
      </button>
    </div>

    <!-- Custom range inputs -->
    <div v-if="showCustomRange" class="px-2 pt-2 space-y-2">
      <div>
        <label class="text-xs text-muted-foreground">From</label>
        <input
          v-model="customStart"
          type="date"
          class="w-full h-8 px-2 text-sm border rounded-md bg-transparent outline-none focus:ring-1 focus:ring-ring"
        />
      </div>
      <div>
        <label class="text-xs text-muted-foreground">To</label>
        <input
          v-model="customEnd"
          type="date"
          class="w-full h-8 px-2 text-sm border rounded-md bg-transparent outline-none focus:ring-1 focus:ring-ring"
        />
      </div>
      <Button size="sm" class="w-full" @click="applyCustomRange">
        Apply
      </Button>
    </div>

    <!-- Footer -->
    <div class="flex justify-end border-t mt-2 pt-2 px-2 pb-1">
      <Button variant="ghost" size="xs" @click="handleClear">
        Clear
      </Button>
    </div>
  </div>
</template>
