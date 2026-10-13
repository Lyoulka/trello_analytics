<script setup lang="ts">
defineProps<{
  value: string
  hint?: string
}>()
</script>

<template>
  <div>
    <div class="text-3xl font-semibold tracking-tight tabular-nums">{{ value }}</div>
    <p v-if="hint" class="mt-1 text-xs text-muted-foreground">{{ hint }}</p>
  </div>
</template>
