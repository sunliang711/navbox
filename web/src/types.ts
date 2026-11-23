export type ApiResponse<T> = {
  code: number;
  data: T;
  message: string;
};

export type Tag = {
  id: string;
  name: string;
  icon: string;
  color: string;
  sort_order: number;
  is_default: boolean;
  is_enabled: boolean;
  site_count: number;
  created_at: string;
  updated_at: string;
};

export type Site = {
  id: string;
  title: string;
  description: string;
  default_url: string;
  lan_url: string;
  open_method: string;
  icon_type: string;
  icon_value: string;
  background_color: string;
  only_name: boolean;
  is_enabled: boolean;
  is_favorite: boolean;
  sort_order: number;
  tags: Tag[];
  created_at: string;
  updated_at: string;
};

export type PublicConfig = {
  default_tag_id: string;
};

export type TagMatchMode = 'all' | 'any';

export type SiteQuery = {
  search?: string;
  tagIds?: string[];
  tagMatch?: TagMatchMode;
  view?: string;
};

export type RecentSite = Site & {
  visited_at: string;
};

export type SessionResp = {
  authenticated: boolean;
};

export type RestoreStatus = {
  enabled: boolean;
  mode: string;
};

export type RestorePasswordResp = {
  restored: boolean;
};

export type SiteSaveReq = {
  title: string;
  description: string;
  default_url: string;
  lan_url: string;
  open_method: string;
  icon_type: string;
  icon_value: string;
  background_color: string;
  only_name: boolean;
  is_enabled: boolean;
  is_favorite: boolean;
  sort_order: number;
  tag_ids: string[];
};

export type TagSaveReq = {
  name: string;
  icon: string;
  color: string;
  sort_order: number;
  is_enabled: boolean;
};

export type BatchTagAction = 'add' | 'remove';

export type OrderItem = {
  id: string;
  sort_order: number;
};

export type IconResp = {
  id: string;
  file_name: string;
  file_path: string;
  url: string;
  sha256: string;
  mime_type: string;
  size_bytes: number;
  created_at: string;
};

export type IconSiteRef = {
  id: string;
  title: string;
};

export type IconLibraryItem = IconResp & {
  used_by: IconSiteRef[];
};

export type IconListResp = {
  items: IconLibraryItem[];
  total: number;
};

export type IconListQuery = {
  q?: string;
  unused?: boolean;
  limit?: number;
  offset?: number;
};

export type ImportReport = {
  imported: ImportCount;
  skipped: ImportCount;
  conflicts: ImportConflict[];
};

export type ImportCount = {
  sites: number;
  tags: number;
  icons: number;
  relations: number;
};

export type ImportConflict = {
  type: string;
  id?: string;
  name?: string;
  reason: string;
};

export type ExportFile = {
  blob: Blob;
  fileName: string;
};

export type CronJob = {
  id: string;
  job_key: string;
  name: string;
  description: string;
  cron_expr: string;
  default_schedule: string;
  enabled: boolean;
  timeout_sec: number;
  config_json: string;
  last_run_at?: string;
  last_status: string;
  last_duration_ms: number;
  last_summary?: string;
  next_run_at?: string;
  next_runs: string[];
  updated_at: string;
};

export type CronJobRun = {
  id: string;
  job_id: string;
  trigger_type: 'schedule' | 'manual' | string;
  triggered_by: string;
  status: 'running' | 'succeeded' | 'failed' | 'skipped' | 'timeout' | string;
  affected_rows: number;
  output_log: string;
  started_at: string;
  finished_at?: string;
};

export type UpdateCronJobReq = {
  cron_expr: string;
  enabled: boolean;
  timeout_sec?: number;
  config?: Record<string, unknown>;
};

export type SafeKeepUpstream = {
  upstream_id: string;
  upstream_name: string;
  upstream_type: string;
  enabled: boolean;
  is_default: boolean;
  priority: number;
};

export type TestSafeKeepReq = {
  api_base_url?: string;
  api_key?: string;
  skip_tls_verify?: boolean;
};

export type TestSafeKeepResp = {
  ok: boolean;
  upstreams: SafeKeepUpstream[];
};

export type AuditLog = {
  id: number;
  actor_type: string;
  actor_id: string;
  actor_name: string;
  action: string;
  resource_type: string;
  resource_id: string;
  status: string;
  ip_address: string;
  user_agent: string;
  request_id: string;
  detail_json: string;
  created_at: string;
};

export type AuditListResp = {
  items: AuditLog[];
  total: number;
};

export type AuditQuery = {
  actor_type?: string;
  action?: string;
  status?: string;
  q?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
};
