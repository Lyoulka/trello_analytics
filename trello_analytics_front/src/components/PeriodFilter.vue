<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { fetchWorkload } from '@/api/analytics'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useFiltersStore, type PeriodPreset } from '@/stores/filters'

export type MemberOption = { id: string; label: string }

const props = withDefaults(
  defineProps<{
    /** Show participant (`member_id`) control — required on Reports. */
    showMember?: boolean
  }>(),
  { showMember: false },
)

const filters = useFiltersStore()
const { from, to, preset, memberId } = storeToRefs(filters)

const presets: Array<{ id: PeriodPreset; label: string }> = [
  { id: '2d', label: '2 дня' },
  { id: '7d', label: '7 дней' },
  { id: '14d', label: '14 дней' },
  { id: '30d', label: '30 дней' },
]

const memberOptions = ref<MemberOption[]>([])

const selectedMemberLabel = computed(() => {
  const id = memberId.value
  if (!id) return ''
  return memberOptions.value.find((m) => m.id === id)?.label ?? id.slice(0, 8)
})

async function loadMemberOptions() {
  if (!props.showMember) {
    memberOptions.value = []
    return
  }
  try {
    const res = await fetchWorkload({ from: from.value, to: to.value })
    const byId = new Map<string, string>()
    for (const w of res.workload ?? []) {
      if (!w.member_id) continue
      const name = w.full_name?.trim()
      byId.set(w.member_id, name || w.member_id.slice(0, 8))
    }
    memberOptions.value = [...byId.entries()]
      .map(([id, label]) => ({ id, label }))
      .sort((a, b) => a.label.localeCompare(b.label, 'ru'))
    // Drop sticky filter if member left workload (e.g. is_active=false).
    if (memberId.value && !byId.has(memberId.value)) {
      memberId.value = ''
    }
  } catch {
    memberOptions.value = []
  }
}

watch(
  [from, to, () => props.showMember],
  () => void loadMemberOptions(),
  { immediate: true },
)
</script>

<template>
  <div class="flex flex-wrap items-end gap-3">
    <div class="flex flex-wrap gap-2">
      <Button
        v-for="p in presets"
        :key="p.id"
        size="sm"
        :variant="preset === p.id ? 'default' : 'outline'"
        @click="filters.applyPreset(p.id)"
      >
        {{ p.label }}
      </Button>
    </div>
    <div class="flex flex-wrap items-end gap-2">
      <div class="space-y-1">
        <Label for="period-from">С</Label>
        <Input
          id="period-from"
          type="date"
          class="w-40"
          :model-value="from"
          @update:model-value="filters.setCustomRange(String($event), to)"
        />
      </div>
      <div class="space-y-1">
        <Label for="period-to">По (включ.)</Label>
        <Input
          id="period-to"
          type="date"
          class="w-40"
          :model-value="to"
          @update:model-value="filters.setCustomRange(from, String($event))"
        />
      </div>
      <div v-if="showMember" class="space-y-1">
        <Label for="member-id">Участник</Label>
        <div class="flex items-center gap-2">
          <select
            id="member-id"
            class="border-input bg-background h-9 w-56 rounded-md border px-2 text-sm"
            :value="memberId"
            @change="memberId = String(($event.target as HTMLSelectElement).value)"
          >
            <option value="">Все</option>
            <option v-for="m in memberOptions" :key="m.id" :value="m.id">
              {{ m.label }}
            </option>
          </select>
          <Button
            v-if="memberId"
            size="sm"
            variant="ghost"
            type="button"
            :title="selectedMemberLabel"
            @click="memberId = ''"
          >
            Сброс
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
