<script setup lang="ts">
import { computed } from 'vue'
import { AlertTriangle } from 'lucide-vue-next'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import type { SyncStatusResponse, Warning } from '@/types/api'

const props = defineProps<{
  warnings?: Warning[] | null
  sync?: SyncStatusResponse | null
}>()

const messages = computed(() => {
  const out: string[] = []
  for (const w of props.warnings ?? []) {
    if (w.message) out.push(w.message)
  }
  if (props.sync?.primary_import_incomplete) {
    out.push('Первичный импорт доски ещё не завершён полностью.')
  }
  for (const w of props.sync?.warnings ?? []) {
    if (w) out.push(w)
  }
  // de-dupe
  return [...new Set(out)]
})
</script>

<template>
  <Alert v-if="messages.length" variant="warning" class="mb-4">
    <AlertTriangle class="size-4" />
    <AlertTitle>Качество данных</AlertTitle>
    <AlertDescription>
      <ul class="mt-1 list-disc space-y-1 pl-4">
        <li v-for="(m, i) in messages" :key="i">{{ m }}</li>
      </ul>
    </AlertDescription>
  </Alert>
</template>
