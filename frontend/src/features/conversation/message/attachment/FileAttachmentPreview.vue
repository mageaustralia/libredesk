<template>
  <div class="flex items-center group text-left">
    <div
      class="relative w-36 h-28 flex flex-col items-center justify-between rounded-lg border bg-gray-50 dark:bg-gray-800 p-3 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors cursor-pointer"
      @click="handleClick"
    >
      <div class="flex-1 flex items-center justify-center">
        <component :is="fileIcon" class="w-10 h-10" :class="iconColor" />
      </div>
      <div class="w-full text-center">
        <p class="text-xs font-medium text-gray-700 dark:text-gray-200 truncate" :title="attachment.name">
          {{ getAttachmentName(attachment.name) }}
        </p>
        <p class="text-[10px] text-gray-400">{{ formatBytes(attachment.size) }}</p>
      </div>
      <div class="absolute top-1.5 right-1.5 opacity-0 group-hover:opacity-100 transition-opacity flex gap-1">
        <button v-if="canPreview" @click.stop="openPreview" class="p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-600" title="Preview">
          <Eye class="w-4 h-4 text-gray-500" />
        </button>
        <a :href="attachment.url" download @click.stop class="p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-600" title="Download">
          <Download class="w-4 h-4 text-gray-500" />
        </a>
      </div>
    </div>

    <!-- PDF preview overlay -->
    <Teleport to="body">
      <div
        v-if="showPdfPreview"
        class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80"
        @click.self="showPdfPreview = false"
      >
        <button
          class="absolute top-4 right-4 text-white hover:text-gray-300 z-10"
          @click="showPdfPreview = false"
        >
          <X :size="28" />
        </button>
        <a
          :href="attachment.url"
          download
          class="absolute top-4 right-14 text-white hover:text-gray-300 z-10"
          title="Download"
        >
          <Download :size="24" />
        </a>
        <iframe
          :src="attachment.url"
          class="w-[90vw] h-[90vh] rounded shadow-2xl bg-white"
        />
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { formatBytes } from '@/utils/file.js'
import { Download, Eye, X, FileText, FileSpreadsheet, File, FileImage, FileArchive, FileCode } from 'lucide-vue-next'

const props = defineProps({
  attachment: {
    type: Object,
    required: true
  }
})

const showPdfPreview = ref(false)

const getAttachmentName = (name) => {
  return name.substring(0, 30)
}

const ext = computed(() => {
  const name = props.attachment.name || ''
  const parts = name.split('.')
  return parts.length > 1 ? parts.pop().toLowerCase() : ''
})

const canPreview = computed(() => {
  return ['pdf'].includes(ext.value)
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

const openPreview = () => {
  showPdfPreview.value = true
}

const handleClick = () => {
  if (canPreview.value) {
    openPreview()
  } else {
    window.open(props.attachment.url, '_blank')
  }
}
</script>
