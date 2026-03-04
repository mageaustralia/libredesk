<template>
  <div>
    <div class="flex items-center px-2 h-12">
      <SidebarTrigger class="cursor-pointer" />
      <Input
        data-search-input
        v-model="model"
        :placeholder="t('globals.terms.search')"
        class="w-full border-none shadow-none focus:ring-0 focus:ring-offset-0"
      />
    </div>
    <Separator />
  </div>
</template>

<script setup>
import { Separator } from '@/components/ui/separator'
import { Input } from '@/components/ui/input'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { useI18n } from 'vue-i18n'
const model = defineModel({
  type: String,
  default: '',
  required: false
})
const { t } = useI18n()

import { ref, onMounted, nextTick } from 'vue'
const searchInput = ref(null)
onMounted(() => {
  nextTick(() => {
    const input = document.querySelector('[data-search-input] input, [data-search-input]')
    if (input) input.focus()
  })
})
</script>

<style scoped>
.focus\:ring-0:focus {
  --tw-ring-offset-shadow: none;
  --tw-ring-shadow: none;
  box-shadow: none;
}
</style>
