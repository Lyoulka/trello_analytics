<script setup lang="ts">
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import MetricValue from './MetricValue.vue'

defineProps<{
  title: string
  value: string
  description?: string
  hint?: string
}>()
</script>

<template>
  <Card>
    <CardHeader class="pb-2">
      <CardDescription>{{ title }}</CardDescription>
      <CardTitle class="sr-only">{{ title }}</CardTitle>
    </CardHeader>
    <CardContent>
      <MetricValue :value="value" :hint="hint" />
      <p v-if="description" class="mt-2 text-xs text-muted-foreground">{{ description }}</p>
    </CardContent>
  </Card>
</template>
