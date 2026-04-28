import type {
  ApiResponse,
  BatchTagAction,
  ExportFile,
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
  TagSaveReq
} from './types';

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    credentials: 'include',
    ...init,
    headers: {
      Accept: 'application/json',
      ...init.headers
    }
  });
  if (!response.ok) {
    throw new Error('request failed');
  }

  const body = (await response.json()) as ApiResponse<T>;
  if (body.code !== 200) {
    throw new Error(body.message || 'request failed');
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
  for (const tagId of query.tagIds ?? []) {
    params.append('tag_ids', tagId);
  }

  const suffix = params.toString();
  return request<Site[]>(`/api/v1/sites${suffix ? `?${suffix}` : ''}`);
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

function parseDownloadFileName(header: string | null): string {
  if (!header) {
    return 'navbox-export.zip';
  }
  const match = header.match(/filename="([^"]+)"/);
  return match?.[1] ?? 'navbox-export.zip';
}
