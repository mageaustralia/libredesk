<script setup>
import { ref, watch } from 'vue'
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

const textValue = ref('')

function initFromModelValue() {
  if (props.modelValue) {
    textValue.value = props.modelValue.value || ''
  } else {
    textValue.value = ''
  }
}

initFromModelValue()
watch(() => props.modelValue, initFromModelValue, { deep: true })

function emitUpdate() {
  emit('update:modelValue', {
    field: props.field.key,
    operator: 'ilike',
    value: textValue.value,
    model: props.field.model || ''
  })
}

function handleKeydown(e) {
  if (e.key === 'Enter') {
    emitUpdate()
  }
}

function handleClear() {
  textValue.value = ''
  emit('remove')
}
</script>

<template>
  <div class="w-64">
    <div class="px-2 pb-2">
      <label class="text-xs text-muted-foreground mb-1 block">{{ field.label }} contains</label>
      <input
        v-model="textValue"
        type="text"
        placeholder="Type and press Enter..."
        class="w-full h-8 px-2 text-sm border rounded-md bg-transparent outline-none focus:ring-1 focus:ring-ring"
        @keydown="handleKeydown"
        autofocus
      />
    </div>
    <div class="flex justify-between border-t mt-2 pt-2 px-2 pb-1">
      <Button variant="ghost" size="xs" @click="handleClear">
        Clear
      </Button>
      <Button variant="default" size="xs" @click="emitUpdate">
        Apply
      </Button>
    </div>
  </div>
</template>
