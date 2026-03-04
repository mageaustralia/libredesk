<template>
  <div class="flex items-center group text-left cursor-pointer" @click="downloadAttachment">
    <div class="relative w-36 h-28 flex flex-col items-center justify-between rounded-lg border bg-gray-50 dark:bg-gray-800 p-3 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
      <div class="flex-1 flex items-center justify-center">
        <component :is="fileIcon" class="w-10 h-10" :class="iconColor" />
      </div>
      <div class="w-full text-center">
        <p class="text-xs font-medium text-gray-700 dark:text-gray-200 truncate" :title="attachment.name">
          {{ getAttachmentName(attachment.name) }}
        </p>
        <p class="text-[10px] text-gray-400">{{ formatBytes(attachment.size) }}</p>
      </div>
      <div class="absolute top-1.5 right-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
        <Download class="w-4 h-4 text-gray-500" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { formatBytes } from '@/utils/file.js'
import { Download, FileText, FileSpreadsheet, File, FileImage, FileArchive, FileCode } from 'lucide-vue-next'

const props = defineProps({
  attachment: {
    type: Object,
    required: true
  }
})

const getAttachmentName = (name) => {
  return name.substring(0, 30)
}

const ext = computed(() => {
  const name = props.attachment.name || ''
  const parts = name.split('.')
  return parts.length > 1 ? parts.pop().toLowerCase() : ''
})

const fileIcon = computed(() => {
  const e = ext.value
  if (['pdf'].includes(e)) return FileText
  if (['xls', 'xlsx', 'csv'].includes(e)) return FileSpreadsheet
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(e)) return FileImage
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(e)) return FileArchive
  if (['html', 'xml', 'json', 'js', 'css'].includes(e)) return FileCode
  if (['doc', 'docx', 'txt', 'rtf'].includes(e)) return FileText
  return File
})

const iconColor = computed(() => {
  const e = ext.value
  if (['pdf'].includes(e)) return 'text-red-500'
  if (['xls', 'xlsx', 'csv'].includes(e)) return 'text-green-600'
  if (['doc', 'docx', 'txt', 'rtf'].includes(e)) return 'text-blue-500'
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(e)) return 'text-amber-600'
  return 'text-gray-500'
})

const downloadAttachment = () => {
  window.open(props.attachment.url, '_blank')
}
</script>