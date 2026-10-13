<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchSettings, fetchWorkflow, patchSettings, putWorkflow } from '@/api/settings'
import { fetchSyncHistory, fetchSyncStatus, startSync } from '@/api/sync'
import { messageFromUnknown, ApiError } from '@/api/errors'
import { formatDateTime } from '@/composables/format'
import { parseNonNegInt } from '@/composables/parseIntField'
import type {
  AppSettingsResponse,
  SyncHistoryRun,
  SyncStatusResponse,
  WorkflowSettings,
} from '@/types/api'
import { useSessionStore } from '@/stores/session'
import DataQualityBanner from '@/components/DataQualityBanner.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const session = useSessionStore()
const canEdit = computed(() => session.hasMinRole('team_lead'))
const canSync = computed(() => session.hasMinRole('admin'))
const canHistory = computed(() => session.hasMinRole('team_lead'))

const loading = ref(true)
const error = ref<string | null>(null)
const saveMsg = ref<string | null>(null)

const settings = ref<AppSettingsResponse | null>(null)
const workflow = ref<WorkflowSettings | null>(null)
const sync = ref<SyncStatusResponse | null>(null)
const history = ref<SyncHistoryRun[]>([])

const appForm = ref({
  timezone_display: '',
  default_period_days: '',
  report_title: '',
  ui_locale: '',
})

const wfForm = ref({
  lead_time_start_rule: 'unset',
  lead_time_start_stage: '',
  cycle_start_stage: '',
  done_stages: '',
  work_stages: '',
  blocked_stages: '',
  bug_reopen_from_stages: '',
  stale_threshold_seconds: '604800',
  data_stale_sync_age_seconds: '86400',
  timezone: 'UTC',
})

function splitCsv(s: string): string[] {
  return s
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean)
}

async function load() {
  loading.value = true
  error.value = null
  try {
    const [s, w, st] = await Promise.all([
      fetchSettings(),
      fetchWorkflow(),
      fetchSyncStatus().catch(() => null),
    ])
    settings.value = s
    workflow.value = w
    sync.value = st
    appForm.value = {
      timezone_display: s.settings.timezone_display ?? '',
      default_period_days: s.settings.default_period_days ?? '',
      report_title: s.settings.report_title ?? '',
      ui_locale: s.settings.ui_locale ?? '',
    }
    wfForm.value = {
      lead_time_start_rule: w.lead_time_start_rule || 'unset',
      lead_time_start_stage: w.lead_time_start_stage ?? '',
      cycle_start_stage: w.cycle_start_stage ?? '',
      done_stages: (w.done_stages ?? []).join(', '),
      work_stages: (w.work_stages ?? []).join(', '),
      blocked_stages: (w.blocked_stages ?? []).join(', '),
      bug_reopen_from_stages: (w.bug_reopen_from_stages ?? []).join(', '),
      stale_threshold_seconds: String(w.stale_threshold_seconds ?? 604800),
      data_stale_sync_age_seconds: String(w.data_stale_sync_age_seconds ?? 86400),
      timezone: w.timezone || 'UTC',
    }
    if (canHistory.value) {
      try {
        const h = await fetchSyncHistory(10)
        history.value = h.runs ?? []
      } catch {
        history.value = []
      }
    }
  } catch (err) {
    error.value = messageFromUnknown(err)
  } finally {
    loading.value = false
  }
}

async function saveAppSettings() {
  if (!canEdit.value) return
  saveMsg.value = null
  try {
    const payload: Record<string, string> = {}
    for (const [k, v] of Object.entries(appForm.value)) {
      if (v !== '') payload[k] = v
    }
    settings.value = await patchSettings(payload)
    saveMsg.value = 'Настройки приложения сохранены'
  } catch (err) {
    saveMsg.value = messageFromUnknown(err)
  }
}

async function saveWorkflow() {
  if (!canEdit.value || !workflow.value) return
  saveMsg.value = null
  try {
    const staleThreshold = parseNonNegInt(
      wfForm.value.stale_threshold_seconds,
      'stale_threshold_seconds',
    )
    const dataStaleAge = parseNonNegInt(
      wfForm.value.data_stale_sync_age_seconds,
      'data_stale_sync_age_seconds',
    )
    const body: WorkflowSettings = {
      ...workflow.value,
      lead_time_start_rule: wfForm.value.lead_time_start_rule,
      lead_time_start_stage: wfForm.value.lead_time_start_stage || undefined,
      cycle_start_stage: wfForm.value.cycle_start_stage || undefined,
      done_stages: splitCsv(wfForm.value.done_stages),
      work_stages: splitCsv(wfForm.value.work_stages),
      blocked_stages: splitCsv(wfForm.value.blocked_stages),
      bug_reopen_from_stages: splitCsv(wfForm.value.bug_reopen_from_stages),
      stale_threshold_seconds: staleThreshold,
      data_stale_sync_age_seconds: dataStaleAge,
      timezone: wfForm.value.timezone || 'UTC',
    }
    workflow.value = await putWorkflow(body)
    saveMsg.value = 'Workflow сохранён'
  } catch (err) {
    saveMsg.value = messageFromUnknown(err)
  }
}

async function onStartSync() {
  if (!canSync.value) return
  saveMsg.value = null
  try {
    const res = await startSync()
    saveMsg.value = `Синхронизация запущена: ${res.run_id}`
    sync.value = await fetchSyncStatus()
  } catch (err) {
    if (err instanceof ApiError && err.status === 409) {
      saveMsg.value = 'Синхронизация уже выполняется'
    } else {
      saveMsg.value = messageFromUnknown(err)
    }
  }
}

onMounted(() => void load())
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Настройки</h1>
      <p class="text-sm text-muted-foreground">
        Workflow, пороги, часовой пояс и синхронизация с Trello. Секреты Trello только в env.
      </p>
    </div>

    <DataQualityBanner :sync="sync" />

    <Alert v-if="error" variant="destructive">
      <AlertTitle>Ошибка</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>
    <Alert v-if="saveMsg">
      <AlertTitle>Статус</AlertTitle>
      <AlertDescription>{{ saveMsg }}</AlertDescription>
    </Alert>

    <Skeleton v-if="loading" class="h-40" />

    <template v-else>
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Статус синхронизации</CardTitle>
          <CardDescription>Последний успешный импорт и текущий прогон.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-2 text-sm">
          <template v-if="sync">
            <p>
              Последний успех:
              <strong>{{ formatDateTime(sync.last_success_at) }}</strong>
            </p>
            <p>
              Первичный импорт не завершён:
              <Badge :variant="sync.primary_import_incomplete ? 'warning' : 'success'">
                {{ sync.primary_import_incomplete ? 'да' : 'нет' }}
              </Badge>
            </p>
            <p v-if="sync.current_run">
              Текущий прогон: {{ sync.current_run.status }} ({{ sync.current_run.run_id }})
            </p>
            <p>
              Счётчики: карточки={{ sync.counts.cards }}, действия={{ sync.counts.actions }},
              участники={{ sync.counts.members }}
            </p>
            <div class="flex flex-wrap gap-2 pt-2">
              <Button v-if="canSync" size="sm" @click="onStartSync">Запустить синхронизацию</Button>
              <Button size="sm" variant="outline" @click="load">Обновить</Button>
            </div>
          </template>
          <p v-else class="text-muted-foreground">Статус синхронизации недоступен.</p>
        </CardContent>
      </Card>

      <Card v-if="canHistory && history.length">
        <CardHeader>
          <CardTitle class="text-base">История синхронизации</CardTitle>
        </CardHeader>
        <CardContent class="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Статус</TableHead>
                <TableHead>Начало</TableHead>
                <TableHead>Конец</TableHead>
                <TableHead>Обработано</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="r in history" :key="r.id">
                <TableCell>{{ r.status }}</TableCell>
                <TableCell class="text-xs">{{ formatDateTime(r.started_at) }}</TableCell>
                <TableCell class="text-xs">{{ formatDateTime(r.finished_at) }}</TableCell>
                <TableCell>{{ r.processed_count }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-base">Настройки приложения</CardTitle>
          <CardDescription>
            Ключ/токен Trello:
            {{ settings?.trello_api_key_configured ? 'ключ ✓' : 'ключ ✗' }}
            /
            {{ settings?.trello_token_configured ? 'токен ✓' : 'токен ✗' }}
          </CardDescription>
        </CardHeader>
        <CardContent class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1">
            <Label for="tz_disp">Часовой пояс отображения</Label>
            <Input id="tz_disp" v-model="appForm.timezone_display" :disabled="!canEdit" />
          </div>
          <div class="space-y-1">
            <Label for="period_days">Период по умолчанию (дней)</Label>
            <Input id="period_days" v-model="appForm.default_period_days" :disabled="!canEdit" />
          </div>
          <div class="space-y-1">
            <Label for="report_title">Заголовок отчёта</Label>
            <Input id="report_title" v-model="appForm.report_title" :disabled="!canEdit" />
          </div>
          <div class="space-y-1">
            <Label for="ui_locale">Язык интерфейса</Label>
            <Input id="ui_locale" v-model="appForm.ui_locale" :disabled="!canEdit" />
          </div>
          <div class="sm:col-span-2">
            <Button v-if="canEdit" @click="saveAppSettings">Сохранить настройки</Button>
            <p v-else class="text-sm text-muted-foreground">Редактирование — для тимлида и выше.</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="text-base">Workflow</CardTitle>
          <CardDescription>Логические стадии через запятую (как в маппинге списков).</CardDescription>
        </CardHeader>
        <CardContent class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1">
            <Label for="lead_rule">Старт lead time</Label>
            <select
              id="lead_rule"
              v-model="wfForm.lead_time_start_rule"
              class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm"
              :disabled="!canEdit"
            >
              <option value="unset">не задано</option>
              <option value="card_created">создание карточки</option>
              <option value="first_enter_stage">первый вход в стадию</option>
            </select>
          </div>
          <div class="space-y-1">
            <Label for="lead_stage">Стадия старта lead time</Label>
            <Input id="lead_stage" v-model="wfForm.lead_time_start_stage" :disabled="!canEdit" />
          </div>
          <div class="space-y-1">
            <Label for="cycle_stage">Стадия старта cycle time</Label>
            <Input id="cycle_stage" v-model="wfForm.cycle_start_stage" :disabled="!canEdit" />
          </div>
          <div class="space-y-1">
            <Label for="tz">Часовой пояс</Label>
            <Input id="tz" v-model="wfForm.timezone" :disabled="!canEdit" />
          </div>
          <div class="space-y-1 sm:col-span-2">
            <Label for="done">Стадии Done</Label>
            <Input id="done" v-model="wfForm.done_stages" :disabled="!canEdit" />
          </div>
          <div class="space-y-1 sm:col-span-2">
            <Label for="work">Рабочие стадии</Label>
            <Input id="work" v-model="wfForm.work_stages" :disabled="!canEdit" />
          </div>
          <div class="space-y-1 sm:col-span-2">
            <Label for="blocked">Стадии блокировки</Label>
            <Input id="blocked" v-model="wfForm.blocked_stages" :disabled="!canEdit" />
          </div>
          <div class="space-y-1 sm:col-span-2">
            <Label for="bug_reopen_from">Стадии reopen бага</Label>
            <Input
              id="bug_reopen_from"
              v-model="wfForm.bug_reopen_from_stages"
              :disabled="!canEdit"
              placeholder="qa_review"
            />
            <p class="text-xs text-muted-foreground">
              Уход из этих стадий не в Done = reopen (счётчик переходов). Список «Тестирование (на
              проверку)» → логическая стадия, напр. qa_review.
            </p>
          </div>
          <div class="space-y-1">
            <Label for="stale">Порог «зависших», сек</Label>
            <Input id="stale" v-model="wfForm.stale_threshold_seconds" :disabled="!canEdit" />
          </div>
          <div class="space-y-1">
            <Label for="data_stale">Порог устаревания sync, сек</Label>
            <Input id="data_stale" v-model="wfForm.data_stale_sync_age_seconds" :disabled="!canEdit" />
          </div>
          <div class="sm:col-span-2">
            <Button v-if="canEdit" @click="saveWorkflow">Сохранить workflow</Button>
            <p v-else class="text-sm text-muted-foreground">Участник видит настройки без записи.</p>
          </div>
        </CardContent>
      </Card>
    </template>
  </div>
</template>
