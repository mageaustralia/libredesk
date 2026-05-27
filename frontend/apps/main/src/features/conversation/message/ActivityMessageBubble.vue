<template>
  <div class="text-center">
    <div class="text-muted-foreground text-sm">
      {{ message.content }}
      <Tooltip>
        <TooltipTrigger>
          <span class="text-xs ml-1">{{ formatActivityTimestamp(message.updated_at) }}</span>
        </TooltipTrigger>
        <TooltipContent>
          <p>{{ formatFullTimestamp(message.updated_at) }}</p>
        </TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<script setup>
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { formatActivityTimestamp, formatFullTimestamp } from '@shared-ui/utils/datetime.js'

defineProps({
  message: Object
})
</script>
