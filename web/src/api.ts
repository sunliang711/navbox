import type {
  ApiResponse,
  AuditListResp,
  AuditQuery,
  BatchTagAction,
  CronJob,
  CronJobRun,
  ExportFile,
  IconListQuery,
  IconListResp,
  IconResp,
  ImportReport,
  OrderItem,
  PublicConfig,
  RestorePasswordResp,
  RestoreStatus,
  SessionResp,
  Site,
  SiteQuery,
  SiteSaveReq,
  Tag,
  TagSaveReq,
  TestSafeKeepReq,
  TestSafeKeepResp,
  UpdateCronJobReq
} from './types';

export class UnauthorizedError extends Error {
  constructor() {
    super('unauthorized');
    this.name = 'UnauthorizedError';
  }
}

export function isUnauthorizedError(error: unknown): error is UnauthorizedError {
  return error instanceof UnauthorizedError;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
    cache: 'no-store',
    ...init,
    headers: {
      Accept: 'application/json',
      ...init.headers
    }
  });
  if (response.status === 401) {
    throw new UnauthorizedError();
  }
  const text = await response.text();
  let body: ApiResponse<T> | null = null;
  if (text) {
    try {
      body = JSON.parse(text) as ApiResponse<T>;
    } catch {
      body = null;
    }
  }
  if (!response.ok) {
    throw new Error(body?.message || 'request failed');
  }
  if (!body || body.code !== 200) {
    throw new Error(body?.message || 'request failed');
  }
  return body.data;
}

function jsonRequest<T>(path: string, method: string, body?: unknown): Promise<T> {
  return request<T>(path, {
    method,
    headers: {
      'Content-Type': 'application/json'
    },
    body: body === undefined ? undefined : JSON.stringify(body)
  });
}

export function listTags(): Promise<Tag[]> {
  return request<Tag[]>('/api/v1/tags');
}

export function getPublicConfig(): Promise<PublicConfig> {
  return request<PublicConfig>('/api/v1/config/public');
}

export function listSites(query: SiteQuery): Promise<Site[]> {
  const params = new URLSearchParams();
  if (query.search) {
    params.set('search', query.search);
  }
  if (query.view) {
    params.set('view', query.view);
  }
  if (query.tagMatch) {
    params.set('tag_match', query.tagMatch);
  }
  for (const tagId of query.tagIds ?? []) {
    params.append('tag_ids', tagId);
  }

  const suffix = params.toString();
  return request<Site[]>(`/api/v1/sites${suffix ? `?${suffix}` : ''}`);
}

export function listAdminSites(): Promise<Site[]> {
  return request<Site[]>('/api/v1/admin/sites');
}

export function getAdminSession(): Promise<SessionResp> {
  return request<SessionResp>('/api/v1/admin/session');
}

export function getRestoreStatus(): Promise<RestoreStatus> {
  return request<RestoreStatus>('/api/v1/restore/status');
}

export function loginAdmin(password: string): Promise<SessionResp> {
  return jsonRequest<SessionResp>('/api/v1/admin/login', 'POST', { password });
}

export function logoutAdmin(): Promise<SessionResp> {
  return jsonRequest<SessionResp>('/api/v1/admin/logout', 'POST');
}

export function changeAdminPassword(currentPassword: string, newPassword: string): Promise<SessionResp> {
  return jsonRequest<SessionResp>('/api/v1/admin/password', 'POST', {
    current_password: currentPassword,
    new_password: newPassword
  });
}

export function restoreAdminPassword(restoreToken: string, newPassword: string): Promise<RestorePasswordResp> {
  return jsonRequest<RestorePasswordResp>('/api/v1/restore/admin-password', 'POST', {
    restore_token: restoreToken,
    new_password: newPassword
  });
}

export function createSite(req: SiteSaveReq): Promise<Site> {
  return jsonRequest<Site>('/api/v1/admin/sites', 'POST', req);
}

export function updateSite(id: string, req: SiteSaveReq): Promise<Site> {
  return jsonRequest<Site>(`/api/v1/admin/sites/${id}`, 'PUT', req);
}

export function deleteSite(id: string): Promise<{ deleted: boolean }> {
  return jsonRequest<{ deleted: boolean }>(`/api/v1/admin/sites/${id}`, 'DELETE');
}

export function batchDeleteSites(siteIds: string[]): Promise<{ deleted: boolean }> {
  return jsonRequest<{ deleted: boolean }>('/api/v1/admin/sites/batch-delete', 'POST', { site_ids: siteIds });
}

export function batchUpdateSiteTags(
  siteIds: string[],
  tagIds: string[],
  action: BatchTagAction
): Promise<{ updated: boolean }> {
  return jsonRequest<{ updated: boolean }>('/api/v1/admin/sites/batch-tags', 'POST', {
    site_ids: siteIds,
    tag_ids: tagIds,
    action
  });
}

export function updateSiteOrder(items: OrderItem[]): Promise<{ updated: boolean }> {
  return jsonRequest<{ updated: boolean }>('/api/v1/admin/sites/order', 'PUT', { items });
}

export function createTag(req: TagSaveReq): Promise<Tag> {
  return jsonRequest<Tag>('/api/v1/admin/tags', 'POST', req);
}

export function updateTag(id: string, req: TagSaveReq): Promise<Tag> {
  return jsonRequest<Tag>(`/api/v1/admin/tags/${id}`, 'PUT', req);
}

export function deleteTag(id: string): Promise<{ deleted: boolean }> {
  return jsonRequest<{ deleted: boolean }>(`/api/v1/admin/tags/${id}`, 'DELETE');
}

export function setDefaultTag(id: string): Promise<{ updated: boolean }> {
  return jsonRequest<{ updated: boolean }>(`/api/v1/admin/tags/${id}/default`, 'PUT');
}

export function updateTagOrder(items: OrderItem[]): Promise<{ updated: boolean }> {
  return jsonRequest<{ updated: boolean }>('/api/v1/admin/tags/order', 'PUT', { items });
}

export async function uploadIcon(file: File): Promise<IconResp> {
  const data = new FormData();
  data.append('file', file);
  return request<IconResp>('/api/v1/admin/icons/upload', {
    method: 'POST',
    body: data
  });
}

export function fetchWebsiteIcon(url: string): Promise<IconResp> {
  return jsonRequest<IconResp>('/api/v1/admin/icons/fetch', 'POST', { url });
}

export function listAdminIcons(query: IconListQuery = {}): Promise<IconListResp> {
  const params = new URLSearchParams();
  if (query.q) {
    params.set('q', query.q);
  }
  if (query.unused) {
    params.set('unused', 'true');
  }
  if (query.limit !== undefined) {
    params.set('limit', String(query.limit));
  }
  if (query.offset !== undefined) {
    params.set('offset', String(query.offset));
  }
  const suffix = params.toString();
  return request<IconListResp>(`/api/v1/admin/icons${suffix ? `?${suffix}` : ''}`);
}

export async function exportConfig(req: { site_ids?: string[]; tag_ids?: string[] }): Promise<ExportFile> {
  const response = await fetch('/api/v1/admin/export', {
    method: 'POST',
    credentials: 'include',
    headers: {
      Accept: 'application/zip',
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(req)
  });
  if (response.status === 401) {
    throw new UnauthorizedError();
  }
  if (!response.ok) {
    throw new Error('request failed');
  }
  const blob = await response.blob();
  return {
    blob,
    fileName: parseDownloadFileName(response.headers.get('Content-Disposition'))
  };
}

export async function importConfig(file: File): Promise<ImportReport> {
  const data = new FormData();
  data.append('file', file);
  return request<ImportReport>('/api/v1/admin/import', {
    method: 'POST',
    body: data
  });
}

export function listCronJobs(): Promise<{ items: CronJob[] }> {
  return request<{ items: CronJob[] }>('/api/v1/admin/cron-jobs');
}

export function previewCronSchedule(schedule: string): Promise<{ next_runs: string[] }> {
  return jsonRequest<{ next_runs: string[] }>('/api/v1/admin/cron-jobs/preview', 'POST', { schedule });
}

export function updateCronJob(id: string, req: UpdateCronJobReq): Promise<CronJob> {
  return jsonRequest<CronJob>(`/api/v1/admin/cron-jobs/${id}`, 'PUT', req);
}

export function runCronJob(id: string): Promise<CronJobRun> {
  return jsonRequest<CronJobRun>(`/api/v1/admin/cron-jobs/${id}/run`, 'POST');
}

export function listCronRuns(id: string, limit = 20): Promise<{ items: CronJobRun[] }> {
  return request<{ items: CronJobRun[] }>(`/api/v1/admin/cron-jobs/${id}/runs?limit=${limit}`);
}

export function revealCronSecrets(id: string): Promise<{ api_key: string }> {
  return request<{ api_key: string }>(`/api/v1/admin/cron-jobs/${id}/secrets`);
}

export function testSafeKeepConnection(id: string, req: TestSafeKeepReq): Promise<TestSafeKeepResp> {
  return jsonRequest<TestSafeKeepResp>(`/api/v1/admin/cron-jobs/${id}/test-safekeep`, 'POST', req);
}

export function buildAuditSearchParams(query: AuditQuery, includePage = true): URLSearchParams {
  const params = new URLSearchParams();
  if (includePage) {
    if (query.limit !== undefined) {
      params.set('limit', String(query.limit));
    }
    if (query.offset !== undefined) {
      params.set('offset', String(query.offset));
    }
  }
  if (query.actor_type) {
    params.set('actor_type', query.actor_type);
  }
  if (query.action) {
    params.set('action', query.action);
  }
  if (query.status) {
    params.set('status', query.status);
  }
  if (query.q) {
    params.set('q', query.q);
  }
  if (query.from) {
    params.set('from', query.from);
  }
  if (query.to) {
    params.set('to', query.to);
  }
  return params;
}

export function listAuditLogs(query: AuditQuery): Promise<AuditListResp> {
  const suffix = buildAuditSearchParams(query, true).toString();
  return request<AuditListResp>(`/api/v1/admin/audit-logs${suffix ? `?${suffix}` : ''}`);
}

export function getAuditLogsExportUrl(format: 'csv' | 'json', query: AuditQuery): string {
  const params = buildAuditSearchParams(query, false);
  params.set('format', format);
  return `/api/v1/admin/audit-logs/export?${params.toString()}`;
}

export async function exportAuditLogs(format: 'csv' | 'json', query: AuditQuery): Promise<ExportFile> {
  const response = await fetch(getAuditLogsExportUrl(format, query), {
    credentials: 'include',
    cache: 'no-store'
  });
  if (response.status === 401) {
    throw new UnauthorizedError();
  }
  if (!response.ok) {
    throw new Error('request failed');
  }
  const blob = await response.blob();
  return {
    blob,
    fileName: parseDownloadFileName(response.headers.get('Content-Disposition'), `navbox-audit-logs.${format}`)
  };
}

function parseDownloadFileName(header: string | null, fallback = 'navbox-export.zip'): string {
  if (!header) {
    return fallback;
  }
  const match = header.match(/filename="([^"]+)"/);
  return match?.[1] ?? fallback;
}
