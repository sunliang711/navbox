import { describeCron } from './cronUtils';
import type { LanguageMode, TranslationKey } from './preferences';
import type { AuditLog } from './types';

export type TranslateFn = (key: TranslationKey, values?: Record<string, string | number>) => string;

export type AuditActionOption = {
  value: string;
  labelKey: TranslationKey;
};

export type AuditActionGroup = {
  labelKey: TranslationKey;
  options: AuditActionOption[];
};

export const AUDIT_ACTION_GROUPS: AuditActionGroup[] = [
  {
    labelKey: 'auditActionGroupSite',
    options: [
      { value: 'site.create', labelKey: 'auditActionSiteCreate' },
      { value: 'site.update', labelKey: 'auditActionSiteUpdate' },
      { value: 'site.delete', labelKey: 'auditActionSiteDelete' },
      { value: 'site.batch_delete', labelKey: 'auditActionSiteBatchDelete' },
      { value: 'site.batch_tags', labelKey: 'auditActionSiteBatchTags' },
      { value: 'site.reorder', labelKey: 'auditActionSiteReorder' }
    ]
  },
  {
    labelKey: 'auditActionGroupTag',
    options: [
      { value: 'tag.create', labelKey: 'auditActionTagCreate' },
      { value: 'tag.update', labelKey: 'auditActionTagUpdate' },
      { value: 'tag.delete', labelKey: 'auditActionTagDelete' },
      { value: 'tag.set_default', labelKey: 'auditActionTagSetDefault' },
      { value: 'tag.reorder', labelKey: 'auditActionTagReorder' }
    ]
  },
  {
    labelKey: 'auditActionGroupIcon',
    options: [
      { value: 'icon.upload', labelKey: 'auditActionIconUpload' },
      { value: 'icon.fetch', labelKey: 'auditActionIconFetch' }
    ]
  },
  {
    labelKey: 'auditActionGroupData',
    options: [
      { value: 'data.export', labelKey: 'auditActionDataExport' },
      { value: 'data.import', labelKey: 'auditActionDataImport' }
    ]
  },
  {
    labelKey: 'auditActionGroupCron',
    options: [
      { value: 'cron.execute', labelKey: 'auditActionCronExecute' },
      { value: 'cron.update', labelKey: 'auditActionCronUpdate' },
      { value: 'cron.test_safekeep', labelKey: 'auditActionCronTestSafeKeep' },
      { value: 'cron.reveal_secrets', labelKey: 'auditActionCronRevealSecrets' }
    ]
  },
  {
    labelKey: 'auditActionGroupAuth',
    options: [
      { value: 'auth.login', labelKey: 'auditActionAuthLogin' },
      { value: 'auth.logout', labelKey: 'auditActionAuthLogout' },
      { value: 'auth.session_revoke', labelKey: 'auditActionSessionRevoke' },
      { value: 'admin.create', labelKey: 'auditActionAccountCreate' },
      { value: 'admin.update', labelKey: 'auditActionAccountUpdate' },
      { value: 'auth.change_password', labelKey: 'auditActionAuthChangePassword' },
      { value: 'auth.restore_password', labelKey: 'auditActionAuthRestorePassword' }
    ]
  }
];

const ACTION_KEY_MAP: Record<string, TranslationKey> = Object.fromEntries(
  AUDIT_ACTION_GROUPS.flatMap((g) => g.options.map((o) => [o.value, o.labelKey]))
);

const RESOURCE_KEY_MAP: Record<string, TranslationKey> = {
  session: 'auditResourceSession',
  admin_setting: 'auditResourceAdminSetting',
  admin_account: 'auditResourceAccount',
  site: 'auditResourceSite',
  tag: 'auditResourceTag',
  icon: 'auditResourceIcon',
  archive: 'auditResourceArchive',
  cron_job: 'auditResourceCronJob'
};

const CRON_JOB_KEY_MAP: Record<string, TranslationKey> = {
  safekeep_backup: 'cronJobSafeKeepName',
  cleanup_audit_logs: 'cronJobCleanupAuditName',
  cleanup_cron_runs: 'cronJobCleanupRunsName'
};

export function parseAuditDetail(raw: string): Record<string, unknown> {
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {};
  } catch {
    return {};
  }
}

function asStr(v: unknown): string {
  return typeof v === 'string' ? v.trim() : '';
}

function asNum(v: unknown): number | null {
  return typeof v === 'number' && Number.isFinite(v) ? v : null;
}

function asBool(v: unknown): boolean | null {
  return typeof v === 'boolean' ? v : null;
}

function asRecord(v: unknown): Record<string, unknown> | null {
  return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : null;
}

function shortId(id: string): string {
  if (!id) return '';
  return id.length > 12 ? `${id.slice(0, 8)}…` : id;
}

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

function formatMs(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  const m = Math.floor(ms / 60_000);
  const s = Math.round((ms % 60_000) / 1000);
  return `${m}m${s}s`;
}

function formatHostname(url: string): string {
  if (!url) return '';
  try {
    return new URL(url).hostname || url;
  } catch {
    return url;
  }
}

export function formatReason(reason: string, lang: LanguageMode): string {
  const isEn = lang === 'en';
  switch (reason) {
    case 'invalid_credentials':
      return isEn ? 'Incorrect password' : '密码错误';
    case 'weak_password':
      return isEn ? 'New password does not meet complexity requirements' : '新密码强度不足';
    case 'unauthenticated':
      return isEn ? 'Session expired or not signed in' : '未登录或会话已失效';
    case 'invalid_token':
      return isEn ? 'Restore token is invalid or expired' : '恢复令牌无效或已过期';
    case 'reset_failed':
      return isEn ? 'Failed to reset password' : '密码重置失败';
    case 'internal_error':
      return isEn ? 'Internal server error' : '系统内部异常';
    default:
      return reason;
  }
}

function formatCronStatus(status: string, t: TranslateFn): string {
  switch (status) {
    case 'succeeded':
      return t('cronStatusSucceeded');
    case 'failed':
      return t('cronStatusFailed');
    case 'running':
      return t('cronStatusRunning');
    case 'skipped':
      return t('cronStatusSkipped');
    case 'timeout':
      return t('cronStatusTimeout');
    default:
      return status;
  }
}

function formatCronJobName(jobKey: string, t: TranslateFn): string {
  const key = CRON_JOB_KEY_MAP[jobKey];
  return key ? t(key) : jobKey;
}

function formatCronSummaryText(summary: string, lang: LanguageMode): string {
  if (!summary || lang === 'en') return summary;
  const auditMatch = /^deleted (\d+) audit logs older than (\d+) days$/.exec(summary);
  if (auditMatch) {
    return `清理了 ${auditMatch[1]} 条超过 ${auditMatch[2]} 天的审计日志`;
  }
  const runsMatch = /^deleted (\d+) cron runs older than (\d+) days(?: and expired sessions)?$/.exec(summary);
  if (runsMatch) {
    return summary.includes('expired sessions')
      ? `清理了 ${runsMatch[1]} 条超过 ${runsMatch[2]} 天的任务记录与过期会话`
      : `清理了 ${runsMatch[1]} 条超过 ${runsMatch[2]} 天的任务记录`;
  }
  const uploadMatch = /^uploaded (.+?) \((\d+) bytes\) to (.+?) -> task (.+?) \((.+?)\)$/.exec(summary);
  if (uploadMatch) {
    const size = formatBytes(Number(uploadMatch[2]));
    return `已上传 ${uploadMatch[1]}（${size}）`;
  }
  return summary;
}

export function formatAuditActor(item: AuditLog, t: TranslateFn): { badge: string; name: string } {
  const badge =
    item.actor_type === 'admin'
      ? t('auditActorAdmin')
      : item.actor_type === 'system'
        ? t('auditActorSystem')
        : item.actor_type || '—';

  const rawName = (item.actor_name || '').trim();
  if (!rawName || rawName === item.actor_type) {
    return { badge, name: '' };
  }
  if (rawName === 'scheduler') {
    return { badge, name: t('auditActorScheduler') };
  }
  return { badge, name: rawName };
}

export function formatAuditAction(action: string, t: TranslateFn): { label: string; category: string } {
  const key = ACTION_KEY_MAP[action];
  const category = action.split('.')[0] || 'other';
  return {
    label: key ? t(key) : action,
    category
  };
}

export function formatAuditResource(
  item: AuditLog,
  t: TranslateFn,
  lang: LanguageMode
): { typeLabel: string; target: string; summary: string } {
  const isEn = lang === 'en';
  const typeKey = RESOURCE_KEY_MAP[item.resource_type];
  const typeLabel = typeKey ? t(typeKey) : item.resource_type || '—';
  const detail = parseAuditDetail(item.detail_json);

  switch (item.action) {
    case 'site.create':
    case 'site.update':
    case 'site.delete': {
      const title = asStr(detail.title);
      const url = asStr(detail.default_url);
      return {
        typeLabel,
        target: title || (item.resource_id ? `#${shortId(item.resource_id)}` : ''),
        summary: url
      };
    }
    case 'site.batch_delete': {
      const count = asNum(detail.count);
      return {
        typeLabel,
        target: count !== null ? (isEn ? `${count} sites` : `共 ${count} 个网站`) : '',
        summary: ''
      };
    }
    case 'site.batch_tags': {
      const mode = asStr(detail.action);
      const siteCount = asNum(detail.site_count) ?? 0;
      const tagCount = asNum(detail.tag_count) ?? 0;
      const target =
        mode === 'remove'
          ? isEn
            ? `Remove ${tagCount} tags from ${siteCount} sites`
            : `从 ${siteCount} 个网站移除 ${tagCount} 个标签`
          : isEn
            ? `Add ${tagCount} tags to ${siteCount} sites`
            : `为 ${siteCount} 个网站添加 ${tagCount} 个标签`;
      return { typeLabel, target, summary: '' };
    }
    case 'site.reorder': {
      const count = asNum(detail.count);
      return {
        typeLabel,
        target: count !== null ? (isEn ? `Reordered ${count} sites` : `调整 ${count} 个网站顺序`) : '',
        summary: ''
      };
    }
    case 'tag.create':
    case 'tag.update':
    case 'tag.delete':
    case 'tag.set_default': {
      const name = asStr(detail.name);
      return {
        typeLabel,
        target: name || (item.resource_id ? `#${shortId(item.resource_id)}` : ''),
        summary: ''
      };
    }
    case 'tag.reorder': {
      const count = asNum(detail.count);
      return {
        typeLabel,
        target: count !== null ? (isEn ? `Reordered ${count} tags` : `调整 ${count} 个标签顺序`) : '',
        summary: ''
      };
    }
    case 'icon.upload': {
      const fileName = asStr(detail.file_name);
      const sizeBytes = asNum(detail.size_bytes);
      return {
        typeLabel,
        target: fileName || (item.resource_id ? `#${shortId(item.resource_id)}` : ''),
        summary: sizeBytes !== null ? formatBytes(sizeBytes) : ''
      };
    }
    case 'icon.fetch': {
      const url = asStr(detail.url);
      const fileName = asStr(detail.file_name);
      const host = formatHostname(url);
      return {
        typeLabel,
        target: host || fileName || (item.resource_id ? `#${shortId(item.resource_id)}` : ''),
        summary: url && url !== host ? url : fileName
      };
    }
    case 'data.export': {
      const fileName = asStr(detail.file_name) || item.resource_id;
      const sizeBytes = asNum(detail.size_bytes);
      const siteCount = asNum(detail.site_filter_count) ?? 0;
      const tagCount = asNum(detail.tag_filter_count) ?? 0;
      const parts: string[] = [];
      if (sizeBytes !== null) parts.push(formatBytes(sizeBytes));
      if (siteCount > 0 || tagCount > 0) {
        parts.push(isEn ? 'Filtered export' : '按条件筛选导出');
      } else if (item.status === 'success') {
        parts.push(isEn ? 'Full backup' : '全量导出');
      }
      return {
        typeLabel,
        target: fileName,
        summary: parts.join(' · ')
      };
    }
    case 'data.import': {
      const fileName = asStr(detail.file_name) || item.resource_id;
      const imported = asRecord(detail.imported);
      const sizeBytes = asNum(detail.size_bytes);
      let summary = '';
      if (imported) {
        const sites = asNum(imported.sites) ?? 0;
        const tags = asNum(imported.tags) ?? 0;
        const icons = asNum(imported.icons) ?? 0;
        summary = isEn
          ? `Imported ${sites} sites, ${tags} tags, ${icons} icons`
          : `导入 ${sites} 网站 · ${tags} 标签 · ${icons} 图标`;
      } else if (sizeBytes !== null) {
        summary = formatBytes(sizeBytes);
      }
      return {
        typeLabel,
        target: fileName,
        summary
      };
    }
    case 'cron.update': {
      const jobKey = asStr(detail.job_key);
      const cronExpr = asStr(detail.cron_expr);
      const enabled = asBool(detail.enabled);
      const parts: string[] = [];
      if (enabled !== null) {
        parts.push(enabled ? (isEn ? 'Enabled' : '已启用') : isEn ? 'Paused' : '已暂停');
      }
      if (cronExpr) {
        parts.push(describeCron(cronExpr, lang) || cronExpr);
      }
      return {
        typeLabel,
        target: jobKey ? formatCronJobName(jobKey, t) : item.resource_id ? `#${shortId(item.resource_id)}` : '',
        summary: parts.join(' · ')
      };
    }
    case 'cron.execute': {
      const jobKey = asStr(detail.job_key);
      const triggerType = asStr(detail.trigger_type);
      const durationMs = asNum(detail.duration_ms);
      const rawSummary = asStr(detail.summary);
      const parts: string[] = [];
      if (triggerType) {
        parts.push(
          triggerType === 'manual'
            ? isEn
              ? 'Manual'
              : '手动执行'
            : isEn
              ? 'Scheduled'
              : '定时触发'
        );
      }
      if (durationMs !== null) {
        parts.push(formatMs(durationMs));
      }
      const niceSummary = formatCronSummaryText(rawSummary, lang);
      if (niceSummary) {
        parts.push(niceSummary);
      }
      return {
        typeLabel,
        target: jobKey ? formatCronJobName(jobKey, t) : item.resource_id ? `#${shortId(item.resource_id)}` : '',
        summary: parts.join(' · ')
      };
    }
    case 'cron.reveal_secrets': {
      const jobKey = asStr(detail.job_key) || 'safekeep_backup';
      return {
        typeLabel,
        target: formatCronJobName(jobKey, t),
        summary: 'API Key'
      };
    }
    case 'cron.test_safekeep': {
      const jobKey = asStr(detail.job_key) || 'safekeep_backup';
      const baseUrl = asStr(detail.api_base_url);
      const upstreamCount = asNum(detail.upstream_count);
      const errMsg = asStr(detail.error);
      const parts: string[] = [];
      if (baseUrl) parts.push(baseUrl);
      if (errMsg) {
        parts.push(errMsg);
      } else if (upstreamCount !== null) {
        parts.push(isEn ? `${upstreamCount} upstreams` : `${upstreamCount} 个可用上游`);
      }
      return {
        typeLabel,
        target: formatCronJobName(jobKey, t),
        summary: parts.join(' · ')
      };
    }
    case 'admin.create':
    case 'admin.update':
      return {
        typeLabel,
        target: asStr(detail.username) || (item.resource_id ? `#${shortId(item.resource_id)}` : ''),
        summary: [asStr(detail.role), typeof detail.disabled === 'boolean' ? (detail.disabled ? t('disabledStatus') : t('enabledStatus')) : '', detail.password_reset ? t('resetPassword') : ''].filter(Boolean).join(' · ')
      };
    case 'auth.session_revoke':
    case 'auth.login':
    case 'auth.logout':
    case 'auth.change_password':
    case 'auth.restore_password': {
      const reason = asStr(detail.reason);
      return {
        typeLabel,
        target: '',
        summary: reason ? formatReason(reason, lang) : ''
      };
    }
    default:
      return {
        typeLabel,
        target: item.resource_id ? `#${shortId(item.resource_id)}` : '',
        summary: ''
      };
  }
}

export type DetailFieldRow = {
  label: string;
  value: string;
  mono?: boolean;
};

export function formatAuditDetailParams(
  item: AuditLog,
  t: TranslateFn,
  lang: LanguageMode
): DetailFieldRow[] {
  const isEn = lang === 'en';
  const detail = parseAuditDetail(item.detail_json);
  const rows: DetailFieldRow[] = [];
  const handled = new Set<string>();

  const pushIf = (key: string, label: string, value: string, mono = false) => {
    if (!value) return;
    handled.add(key);
    rows.push({ label, value, mono });
  };

  if (detail.title !== undefined) {
    pushIf('title', isEn ? 'Site Name' : '网站名称', asStr(detail.title));
  }
  if (detail.default_url !== undefined) {
    pushIf('default_url', isEn ? 'Default URL' : '默认地址', asStr(detail.default_url), true);
  }
  if (detail.name !== undefined) {
    pushIf('name', isEn ? 'Tag Name' : '标签名称', asStr(detail.name));
  }
  if (detail.job_key !== undefined) {
    const jk = asStr(detail.job_key);
    pushIf('job_key', isEn ? 'Cron Job' : '定时任务', formatCronJobName(jk, t));
  }
  if (detail.trigger_type !== undefined) {
    const tt = asStr(detail.trigger_type);
    pushIf(
      'trigger_type',
      isEn ? 'Trigger' : '触发方式',
      tt === 'manual' ? (isEn ? 'Manual' : '手动执行') : isEn ? 'Scheduled' : '定时调度'
    );
  }
  if (detail.enabled !== undefined) {
    const en = asBool(detail.enabled);
    if (en !== null) {
      pushIf('enabled', isEn ? 'Schedule Status' : '调度状态', en ? (isEn ? 'Enabled' : '已启用') : isEn ? 'Paused' : '已暂停');
    }
  }
  if (detail.cron_expr !== undefined) {
    const expr = asStr(detail.cron_expr);
    const desc = describeCron(expr, lang);
    pushIf('cron_expr', isEn ? 'Schedule' : '执行周期', desc ? `${desc} (${expr})` : expr);
  }
  if (detail.timeout_sec !== undefined) {
    const sec = asNum(detail.timeout_sec);
    if (sec !== null) {
      pushIf('timeout_sec', isEn ? 'Timeout' : '超时时间', isEn ? `${sec}s` : `${sec} 秒`);
    }
  }
  if (detail.status !== undefined) {
    const st = asStr(detail.status);
    pushIf('status', isEn ? 'Job Result' : '任务状态', formatCronStatus(st, t));
  }
  if (detail.duration_ms !== undefined) {
    const ms = asNum(detail.duration_ms);
    if (ms !== null) {
      pushIf('duration_ms', isEn ? 'Duration' : '执行耗时', formatMs(ms));
    }
  }
  if (detail.affected_rows !== undefined) {
    const n = asNum(detail.affected_rows);
    if (n !== null) {
      pushIf('affected_rows', isEn ? 'Affected Rows' : '影响记录数', String(n));
    }
  }
  if (detail.summary !== undefined) {
    const rawSum = asStr(detail.summary);
    pushIf('summary', isEn ? 'Execution Summary' : '执行摘要', formatCronSummaryText(rawSum, lang));
  }
  if (detail.api_base_url !== undefined) {
    pushIf('api_base_url', isEn ? 'SafeKeep URL' : 'SafeKeep 地址', asStr(detail.api_base_url), true);
  }
  if (detail.upstream_count !== undefined) {
    const n = asNum(detail.upstream_count);
    if (n !== null) {
      pushIf('upstream_count', isEn ? 'Available Upstreams' : '可用上游数量', String(n));
    }
  }
  if (detail.error !== undefined) {
    pushIf('error', isEn ? 'Error' : '错误信息', asStr(detail.error));
  }
  if (detail.reason !== undefined) {
    pushIf('reason', isEn ? 'Reason' : '原因', formatReason(asStr(detail.reason), lang));
  }
  if (detail.action !== undefined && item.action === 'site.batch_tags') {
    const act = asStr(detail.action);
    pushIf(
      'action',
      isEn ? 'Operation' : '调整方式',
      act === 'remove' ? (isEn ? 'Remove tags' : '移除标签') : isEn ? 'Add tags' : '添加标签'
    );
  }
  if (detail.site_count !== undefined) {
    const n = asNum(detail.site_count);
    if (n !== null) pushIf('site_count', isEn ? 'Sites Affected' : '涉及网站数', String(n));
  }
  if (detail.tag_count !== undefined) {
    const n = asNum(detail.tag_count);
    if (n !== null) pushIf('tag_count', isEn ? 'Tags Applied' : '涉及标签数', String(n));
  }
  if (detail.count !== undefined) {
    const n = asNum(detail.count);
    if (n !== null) pushIf('count', isEn ? 'Item Count' : '处理数量', String(n));
  }
  if (detail.file_name !== undefined) {
    pushIf('file_name', isEn ? 'File Name' : '文件名称', asStr(detail.file_name));
  }
  if (detail.mime_type !== undefined) {
    pushIf('mime_type', isEn ? 'File Type' : '文件类型', asStr(detail.mime_type));
  }
  if (detail.size_bytes !== undefined) {
    const n = asNum(detail.size_bytes);
    if (n !== null) pushIf('size_bytes', isEn ? 'File Size' : '文件大小', formatBytes(n));
  }
  if (detail.url !== undefined) {
    pushIf('url', isEn ? 'Source URL' : '来源链接', asStr(detail.url), true);
  }
  if (detail.site_filter_count !== undefined) {
    const n = asNum(detail.site_filter_count);
    if (n !== null) {
      pushIf(
        'site_filter_count',
        isEn ? 'Site Scope' : '导出网站范围',
        n === 0 ? (isEn ? 'All sites' : '全部网站') : isEn ? `${n} selected` : `指定 ${n} 个网站`
      );
    }
  }
  if (detail.tag_filter_count !== undefined) {
    const n = asNum(detail.tag_filter_count);
    if (n !== null) {
      pushIf(
        'tag_filter_count',
        isEn ? 'Tag Scope' : '导出标签范围',
        n === 0 ? (isEn ? 'All tags' : '全部标签') : isEn ? `${n} selected` : `指定 ${n} 个标签`
      );
    }
  }
  if (detail.imported !== undefined) {
    const rec = asRecord(detail.imported);
    if (rec) {
      const s = asNum(rec.sites) ?? 0;
      const tg = asNum(rec.tags) ?? 0;
      const ic = asNum(rec.icons) ?? 0;
      const rel = asNum(rec.relations) ?? 0;
      pushIf(
        'imported',
        isEn ? 'Imported' : '成功导入',
        isEn
          ? `${s} sites, ${tg} tags, ${ic} icons, ${rel} links`
          : `${s} 个网站、${tg} 个标签、${ic} 个图标、${rel} 条关联`
      );
    }
  }
  if (detail.skipped !== undefined) {
    const rec = asRecord(detail.skipped);
    if (rec) {
      const s = asNum(rec.sites) ?? 0;
      const tg = asNum(rec.tags) ?? 0;
      const ic = asNum(rec.icons) ?? 0;
      pushIf(
        'skipped',
        isEn ? 'Skipped' : '跳过重复',
        isEn ? `${s} sites, ${tg} tags, ${ic} icons` : `${s} 个网站、${tg} 个标签、${ic} 个图标`
      );
    }
  }

  // Skip redundant IDs already shown in Resource ID
  handled.add('job_id');
  if (Array.isArray(detail.site_ids)) {
    handled.add('site_ids');
    if (detail.site_ids.length > 0) {
      rows.push({
        label: isEn ? 'Site IDs' : '网站 ID 列表',
        value: detail.site_ids.map(String).join(', '),
        mono: true
      });
    }
  }

  for (const [key, val] of Object.entries(detail)) {
    if (handled.has(key) || val === null || val === undefined || val === '') continue;
    rows.push({
      label: key,
      value: typeof val === 'object' ? JSON.stringify(val) : String(val),
      mono: typeof val === 'object'
    });
  }

  return rows;
}
