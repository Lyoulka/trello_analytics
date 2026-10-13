import type { AppRole } from './roles'

/** Error envelope from docs/api.md */
export interface ApiErrorBody {
  error: {
    code: string
    message: string
  }
}

export interface User {
  id: string
  email: string
  display_name: string
  role: AppRole | string
  is_active: boolean
}

export interface Warning {
  code: string
  message: string
  metric?: string
  card_id?: string | null
}

export interface CountMetric {
  available: boolean
  count: number
  card_ids?: string[]
}

export interface DurationStats {
  available: boolean
  count: number
  mean_seconds?: number
  median_seconds?: number
}

export interface BugReopenMetric {
  available: boolean
  completed_bugs: number
  reopened_bugs: number
  reopen_percent?: number
  completed_bug_card_ids?: string[]
  reopened_card_ids?: string[]
}

export interface WIPMetric {
  available: boolean
  count: number
  card_ids?: string[]
  bug_card_ids?: string[]
  blocked_count: number
}

export interface StaleMetric {
  available: boolean
  count: number
  card_ids?: string[]
}

export interface WorkloadEntry {
  member_id: string
  /** Display name from Trello member (full_name / username); may be empty. */
  full_name?: string
  active_cards: number
  completed_cards: number
}

export interface NeedsReviewItem {
  card_id: string
  reason: string
  metric?: string
  action_id?: string
}

export interface OverviewResponse {
  completed: CountMetric
  throughput: CountMetric
  lead_time: DurationStats
  cycle_time: DurationStats
  bug_reopen: BugReopenMetric
  fix_time: DurationStats
  wip: WIPMetric
  stale: StaleMetric
  workload: WorkloadEntry[]
  warnings: Warning[]
  needs_review: NeedsReviewItem[]
}

export interface CardListItem {
  id: string
  trello_card_id: string
  name: string
  current_list_id: string | null
  current_list_name: string | null
  is_archived: boolean
  created_at_trello: string | null
  last_trello_update_at: string | null
  last_updated_by_member_id?: string | null
  last_updated_by_name?: string | null
  member_ids: string[]
  label_ids: string[]
}

export interface CardDetail extends CardListItem {
  description: string | null
}

export interface CardsListResponse {
  items: CardListItem[]
  total: number
  limit: number
  offset: number
}

export interface CardHistoryItem {
  id: string
  trello_action_id: string
  action_type: string
  action_date: string
  member_id: string | null
  member_name?: string | null
  source_list_id: string | null
  source_list_name?: string | null
  destination_list_id: string | null
  destination_list_name?: string | null
}

export interface CardHistoryResponse {
  events: CardHistoryItem[]
}

export interface ReportCard {
  id: string
  trello_card_id: string
  name: string
  member_ids: string[]
  /** Display names for current assignees (same order as resolvable member_ids). */
  member_names?: string[]
  label_ids: string[]
  is_bug: boolean
  list_name?: string | null
}

export interface ReportPreview {
  period: { from: string; to: string }
  /** Counts are JSON null when Engine marks the metric unavailable (incomplete workflow). */
  summary: {
    completed_count: number | null
    bugs_fixed_count: number | null
    reopened_count: number | null
    wip_count: number | null
  }
  completed_tasks: ReportCard[]
  bugs: {
    fixed: ReportCard[]
    reopened: ReportCard[]
    still_in_work: ReportCard[]
  }
  current_work: {
    wip: ReportCard[]
    stale: ReportCard[]
  }
  problems: {
    stale_card_ids: string[]
    reopened_card_ids: string[]
    needs_review: NeedsReviewItem[]
    data_quality_hints: string[]
  }
  warnings: Warning[]
  /** Per-metric delta is null when either current or previous side is unavailable. */
  deltas?: {
    completed_delta: number | null
    reopened_delta: number | null
  }
}

export interface AppSettingsResponse {
  settings: Record<string, string>
  board_id: string
  trello_board_id: string
  trello_api_key_configured: boolean
  trello_token_configured: boolean
}

export interface WorkflowSettings {
  board_id: string
  lead_time_start_rule: string
  lead_time_start_stage?: string
  cycle_start_stage?: string
  done_stages: string[]
  blocked_stages: string[]
  work_stages: string[]
  bug_reopen_from_stages: string[]
  stale_threshold_seconds: number
  data_stale_sync_age_seconds: number
  timezone: string
  list_stages: Record<string, string>
  bug_label_ids: string[]
  updated_at?: string
  updated_by?: string | null
}

/** current_run on GET /sync/status */
export interface SyncRunStatus {
  run_id: string
  status: string
  started_at: string
  finished_at: string | null
  processed_count: number
  created_count: number
  updated_count: number
  error_count: number
  error_summary: string | null
}

/** Row from GET /sync/history (store uses `id`, not `run_id`) */
export interface SyncHistoryRun {
  id: string
  status: string
  started_at: string
  finished_at: string | null
  processed_count: number
  created_count: number
  updated_count: number
  error_count: number
  error_summary: string | null
}

export interface SyncStatusResponse {
  board_id: string
  last_success_at: string | null
  current_run: SyncRunStatus | null
  earliest_action_at: string | null
  latest_action_at: string | null
  counts: {
    cards: number
    lists: number
    members: number
    labels: number
    actions: number
  }
  primary_import_incomplete: boolean
  warnings: string[]
}

export interface SyncHistoryResponse {
  runs: SyncHistoryRun[]
}

export interface UsersListResponse {
  users: User[]
}

/** Imported Trello board participant (not an app user). */
export interface BoardMember {
  id: string
  trello_member_id: string
  full_name: string
  username: string
  is_active: boolean
}

export interface MembersListResponse {
  members: BoardMember[]
}

/** Imported Trello list/column. */
export interface BoardList {
  id: string
  trello_list_id: string
  name: string
  is_archived: boolean
  is_active: boolean
  logical_stage: string | null
}

export interface ListsListResponse {
  lists: BoardList[]
}

export interface PeriodQuery {
  from: string
  to: string
  member_id?: string
  label_id?: string
}
