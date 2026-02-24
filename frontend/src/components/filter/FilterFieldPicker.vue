<script setup>
import { ref, computed } from 'vue'
import {
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem
} from '@/components/ui/command'

const props = defineProps({
  fields: {
    type: Array,
    required: true
  },
  activeFieldKeys: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['select', 'close'])

const searchTerm = ref('')

const availableFields = computed(() => {
  return props.fields.filter((f) => !props.activeFieldKeys.includes(f.key))
})

function handleSelect(field) {
  emit('select', field)
}
</script>

<template>
  <Command
    v-model:search-term="searchTerm"
    class="w-56"
    :filter-function="(list, term) => list.filter((item) => {
      const field = availableFields.find((f) => f.key === item)
      return field && field.label.toLowerCase().includes(term.toLowerCase())
    })"
  >
    <CommandInput placeholder="Search fields..." />
    <CommandList>
      <CommandEmpty>No fields found.</CommandEmpty>
      <CommandGroup>
        <CommandItem
          v-for="field in availableFields"
          :key="field.key"
          :value="field.key"
          @select="handleSelect(field)"
        >
          {{ field.label }}
        </CommandItem>
      </CommandGroup>
    </CommandList>
  </Command>
</template>
