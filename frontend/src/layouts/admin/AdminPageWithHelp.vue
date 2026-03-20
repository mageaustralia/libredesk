<template>
  <div class="flex gap-4 h-full">
    <div class="flex-1 min-w-0 relative">
      <slot name="content" />
    </div>
    <div class="rounded w-[200px] shrink-0 p-2 space-y-2 self-start text-sm text-muted-foreground">
      <slot name="help" />
    </div>
  </div>
</template>
