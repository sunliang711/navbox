import {
  Check,
  Copy,
  Eye,
  EyeOff,
  History,
  Loader2,
  Pencil,
  Play,
  PlugZap,
  Plus,
  RefreshCw,
  Save,
  X
} from 'lucide-react';
import { type FormEvent, useEffect, useRef, useState } from 'react';
import {
  listCronJobs,
  listCronRuns,
  previewCronSchedule,
  revealCronSecrets,
  runCronJob,
  testSafeKeepConnection,
  updateCronJob
} from './api';
import { copyToClipboard, describeCron, formatDateTime, formatDuration } from './cronUtils';
import { type TranslationKey, usePreferences } from './preferences';
import type { CronJob, CronJobRun, SafeKeepUpstream } from './types';

type PresetOption = {
  labelKey: TranslationKey;
  value: string;
};

const SCHEDULE_PRESETS: PresetOption[] = [
  { labelKey: 'cronPresetEveryMinute', value: '* * * * *' },
  { labelKey: 'cronPresetEvery5Minutes', value: '*/5 * * * *' },
  { labelKey: 'cronPresetEvery10Minutes', value: '*/10 * * * *' },
  { labelKey: 'cronPresetHourly', value: '0 * * * *' },
  { labelKey: 'cronPresetDaily00', value: '0 0 * * *' },
  { labelKey: 'cronPresetWeeklyMon00', value: '0 0 * * 1' },
  { labelKey: 'cronPresetMonthly01', value: '0 0 1 * *' },
  { labelKey: 'cronPresetCustom', value: 'custom' }
];

const JOB_I18N: Record<string, { name: TranslationKey; desc: TranslationKey }> = {
  safekeep_backup: { name: 'cronJobSafeKeepName', desc: 'cronJobSafeKeepDesc' },
  cleanup_audit_logs: { name: 'cronJobCleanupAuditName', desc: 'cronJobCleanupAuditDesc' },
  cleanup_cron_runs: { name: 'cronJobCleanupRunsName', desc: 'cronJobCleanupRunsDesc' }
};

type SafeKeepForm = {
  api_base_url: string;
  api_key: string;
  has_api_key: boolean;
  clear_api_key: boolean;
  targets: string[];
  path: string;
  mode: 'sync' | 'async';
  rename: string;
  skip_dedup: boolean;
  skip_tls_verify: boolean;
};

type RetentionForm = {
  retention_days: number;
};

type StructuredParams =
  | { kind: 'safekeep_backup'; data: SafeKeepForm }
  | { kind: 'cleanup_audit_logs'; data: RetentionForm }
  | { kind: 'cleanup_cron_runs'; data: RetentionForm }
  | { kind: 'raw'; data: string };

function safeParseJSON(raw?: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(raw || '{}');
    return parsed && typeof parsed === 'object' ? (parsed as Record<string, unknown>) : {};
  } catch {
    return {};
  }
}

function parseSafeKeepForm(raw?: string): SafeKeepForm {
  const obj = safeParseJSON(raw);
  const rawTargets = typeof obj.targets === 'string' ? obj.targets : '';
  const targets = rawTargets
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);
  return {
    api_base_url: typeof obj.api_base_url === 'string' ? obj.api_base_url : '',
    api_key: '',
    has_api_key: Boolean(obj.has_api_key),
    clear_api_key: false,
    targets,
    path: typeof obj.path === 'string' ? obj.path : 'navbox-backups',
    mode: obj.mode === 'async' ? 'async' : 'sync',
    rename: typeof obj.rename === 'string' ? obj.rename : '',
    skip_dedup: Boolean(obj.skip_dedup),
    skip_tls_verify: Boolean(obj.skip_tls_verify)
  };
}

function parseStructuredParams(job: CronJob): StructuredParams {
  const obj = safeParseJSON(job.config_json);
  switch (job.job_key) {
    case 'safekeep_backup':
      return { kind: 'safekeep_backup', data: parseSafeKeepForm(job.config_json) };
    case 'cleanup_audit_logs':
      return {
        kind: 'cleanup_audit_logs',
        data: { retention_days: Number(obj.retention_days) > 0 ? Number(obj.retention_days) : 90 }
      };
    case 'cleanup_cron_runs':
      return {
        kind: 'cleanup_cron_runs',
        data: { retention_days: Number(obj.retention_days) > 0 ? Number(obj.retention_days) : 180 }
      };
    default:
      return { kind: 'raw', data: job.config_json || '{}' };
  }
}

export function CronPanel({
  onNotice,
  onUnauthorized,
  reauthVersion = 0
}: {
  onNotice: (msg: string) => void;
  onUnauthorized: (err: unknown) => boolean;
  reauthVersion?: number;
}) {
  const { t, language } = usePreferences();
  const [loading, setLoading] = useState(true);
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [runningId, setRunningId] = useState('');
  const [togglingId, setTogglingId] = useState('');

  // Edit panel state
  const [editing, setEditing] = useState<CronJob | null>(null);
  const [preset, setPreset] = useState('custom');
  const [cronExpr, setCronExpr] = useState('');
  const [enabled, setEnabled] = useState(true);
  const [timeoutSec, setTimeoutSec] = useState(300);
  const [params, setParams] = useState<StructuredParams | null>(null);
  const [previewRaw, setPreviewRaw] = useState<string[]>([]);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState('');
  const [formError, setFormError] = useState('');
  const [saving, setSaving] = useState(false);
  const [targetInput, setTargetInput] = useState('');
  const [testingSafeKeep, setTestingSafeKeep] = useState(false);
  const [testResultMsg, setTestResultMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [remoteUpstreams, setRemoteUpstreams] = useState<SafeKeepUpstream[]>([]);
  const previewSeq = useRef(0);

  // Runs panel state
  const [runsJob, setRunsJob] = useState<CronJob | null>(null);
  const [runs, setRuns] = useState<CronJobRun[]>([]);
  const [runsLoading, setRunsLoading] = useState(false);
  const [runsError, setRunsError] = useState('');

  const jobName = (job: CronJob) => {
    const mapped = JOB_I18N[job.job_key];
    return mapped ? t(mapped.name) : job.name;
  };

  const jobDesc = (job: CronJob) => {
    const mapped = JOB_I18N[job.job_key];
    return mapped ? t(mapped.desc) : job.description;
  };

  const loadJobs = async () => {
    try {
      const res = await listCronJobs();
      setJobs(res.items || []);
    } catch (error) {
      if (!onUnauthorized(error)) {
        const message = error instanceof Error ? error.message : String(error);
        onNotice(t('cronLoadFailed', { error: message }));
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadJobs();
  }, []);

  useEffect(() => {
    if (reauthVersion <= 0) {
      return;
    }
    if (jobs.length === 0 && !loading) {
      void loadJobs();
    }
    if (runsJob && runs.length === 0 && !runsLoading) {
      void loadRuns(runsJob);
    }
  }, [reauthVersion]);

  const openEdit = (job: CronJob, overrideEnabled?: boolean) => {
    const matched = SCHEDULE_PRESETS.find((item) => item.value === job.cron_expr)?.value ?? 'custom';
    const parsedParams = parseStructuredParams(job);
    const nextEnabled = overrideEnabled ?? job.enabled;
    const nextTimeout = job.timeout_sec > 0 ? job.timeout_sec : 300;

    setEditing(job);
    setPreset(matched);
    setCronExpr(job.cron_expr);
    setEnabled(nextEnabled);
    setTimeoutSec(nextTimeout);
    setParams(parsedParams);
    setTargetInput('');
    setFormError('');
    setPreviewError('');
    setPreviewRaw(job.next_runs || []);
    setTestResultMsg(null);
    setRemoteUpstreams([]);
  };

  const closeEdit = () => {
    if (saving) {
      return;
    }
    setEditing(null);
    setParams(null);
    setFormError('');
    setTestResultMsg(null);
    setRemoteUpstreams([]);
  };

  // Live preview of next 5 runs whenever cronExpr changes
  useEffect(() => {
    if (!editing) {
      return;
    }
    const expr = cronExpr.trim();
    if (!expr) {
      setPreviewRaw([]);
      setPreviewError(t('cronErrEmptyExpr'));
      return;
    }
    const seq = ++previewSeq.current;
    const timer = window.setTimeout(async () => {
      setPreviewLoading(true);
      try {
        const res = await previewCronSchedule(expr);
        if (previewSeq.current !== seq) {
          return;
        }
        setPreviewRaw(res.next_runs || []);
        setPreviewError('');
      } catch (error) {
        if (previewSeq.current !== seq) {
          return;
        }
        if (onUnauthorized(error)) {
          return;
        }
        setPreviewRaw([]);
        setPreviewError(error instanceof Error ? error.message : t('cronErrInvalidExpr'));
      } finally {
        if (previewSeq.current === seq) {
          setPreviewLoading(false);
        }
      }
    }, 200);
    return () => window.clearTimeout(timer);
  }, [editing?.id, cronExpr]);

  const handlePresetChange = (value: string) => {
    setPreset(value);
    if (value !== 'custom') {
      setCronExpr(value);
    }
  };

  const handleToggle = async (job: CronJob, nextEnabled: boolean) => {
    if (togglingId) {
      return;
    }
    if (nextEnabled && job.job_key === 'safekeep_backup') {
      const parsed = parseSafeKeepForm(job.config_json);
      if (!parsed.api_base_url.trim() || !parsed.has_api_key) {
        onNotice(t('cronNeedSafeKeepConfig'));
        openEdit(job, true);
        return;
      }
    }
    setTogglingId(job.id);
    try {
      await updateCronJob(job.id, {
        cron_expr: job.cron_expr,
        enabled: nextEnabled,
        timeout_sec: job.timeout_sec
      });
      await loadJobs();
    } catch (error) {
      if (!onUnauthorized(error)) {
        const msg = error instanceof Error ? error.message : String(error);
        onNotice(t('cronToggleFailed', { error: msg }));
      }
    } finally {
      setTogglingId('');
    }
  };

  const handleSave = async (event: FormEvent) => {
    event.preventDefault();
    if (!editing || !params || previewError) {
      return;
    }
    setFormError('');

    const cleanTimeout = Number(timeoutSec);
    if (!Number.isInteger(cleanTimeout) || cleanTimeout < 1 || cleanTimeout > 86400) {
      setFormError(t('cronErrTimeout'));
      return;
    }

    let cfg: Record<string, unknown>;
    switch (params.kind) {
      case 'cleanup_audit_logs':
      case 'cleanup_cron_runs': {
        const days = Number(params.data.retention_days);
        if (!Number.isInteger(days) || days < 1 || days > 36500) {
          setFormError(t('cronErrRetentionDays'));
          return;
        }
        cfg = { retention_days: days };
        break;
      }
      case 'safekeep_backup': {
        const sk = params.data;
        const hasKey = (!sk.clear_api_key && sk.has_api_key) || Boolean(sk.api_key.trim());
        if (enabled && (!sk.api_base_url.trim() || !hasKey)) {
          setFormError(t('cronNeedSafeKeepConfig'));
          return;
        }
        const pendingTag = targetInput.trim();
        const finalTargets = pendingTag && !sk.targets.includes(pendingTag) ? [...sk.targets, pendingTag] : sk.targets;
        cfg = {
          api_base_url: sk.api_base_url.trim(),
          targets: finalTargets.join(','),
          path: sk.path.trim(),
          mode: sk.mode,
          rename: sk.rename.trim(),
          skip_dedup: sk.skip_dedup,
          skip_tls_verify: sk.skip_tls_verify
        };
        if (sk.clear_api_key) {
          cfg.api_key = '';
        } else if (sk.api_key.trim()) {
          cfg.api_key = sk.api_key.trim();
        }
        break;
      }
      case 'raw': {
        try {
          cfg = JSON.parse(params.data || '{}') as Record<string, unknown>;
        } catch (error) {
          setFormError(error instanceof Error ? error.message : 'invalid json');
          return;
        }
        break;
      }
    }

    setSaving(true);
    try {
      await updateCronJob(editing.id, {
        cron_expr: cronExpr.trim(),
        enabled,
        timeout_sec: cleanTimeout,
        config: cfg
      });
      closeEdit();
      onNotice(t('cronSaveSuccess'));
      await loadJobs();
    } catch (error) {
      if (!onUnauthorized(error)) {
        const msg = error instanceof Error ? error.message : String(error);
        setFormError(t('cronSaveFailed', { error: msg }));
      }
    } finally {
      setSaving(false);
    }
  };

  const handleRunNow = async (job: CronJob) => {
    if (runningId) {
      return;
    }
    const displayName = jobName(job);
    setRunningId(job.id);
    onNotice(t('cronRunningNotice', { name: displayName }));
    try {
      const run = await runCronJob(job.id);
      const detail = run.output_log ? (language === 'en' ? ` (${run.output_log})` : `（${run.output_log}）`) : '';
      if (run.status === 'succeeded') {
        onNotice(t('cronRunSucceeded', { name: displayName, detail }));
      } else if (run.status === 'skipped') {
        onNotice(t('cronRunSkipped', { name: displayName, detail }));
      } else {
        onNotice(t('cronRunFailed', { name: displayName, detail }));
      }
      await loadJobs();
      if (runsJob?.id === job.id) {
        await loadRuns(job);
      }
    } catch (error) {
      if (!onUnauthorized(error)) {
        const msg = error instanceof Error ? error.message : String(error);
        onNotice(t('cronRunError', { error: msg }));
      }
    } finally {
      setRunningId('');
    }
  };

  const loadRuns = async (job: CronJob) => {
    setRunsError('');
    setRunsLoading(true);
    try {
      const res = await listCronRuns(job.id, 30);
      setRuns(res.items || []);
    } catch (error) {
      if (!onUnauthorized(error)) {
        setRunsError(error instanceof Error ? error.message : String(error));
      }
    } finally {
      setRunsLoading(false);
    }
  };

  const openRuns = (job: CronJob) => {
    setRunsJob(job);
    setRuns([]);
    void loadRuns(job);
  };

  const skForm = params?.kind === 'safekeep_backup' ? params.data : null;
  const setSkForm = (next: SafeKeepForm) => setParams({ kind: 'safekeep_backup', data: next });

  const addTargetTag = (raw: string) => {
    if (!skForm) {
      return;
    }
    const items = raw
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
    if (items.length === 0) {
      return;
    }
    const next = [...skForm.targets];
    for (const item of items) {
      if (!next.includes(item)) {
        next.push(item);
      }
    }
    setSkForm({ ...skForm, targets: next });
    setTargetInput('');
  };

  const removeTargetTag = (name: string) => {
    if (!skForm) {
      return;
    }
    setSkForm({ ...skForm, targets: skForm.targets.filter((item) => item !== name) });
  };

  const handleTestSafeKeep = async () => {
    if (!editing || !skForm || testingSafeKeep) {
      return;
    }
    setTestingSafeKeep(true);
    setTestResultMsg(null);
    try {
      const req: { api_base_url: string; api_key?: string; skip_tls_verify: boolean } = {
        api_base_url: skForm.api_base_url.trim(),
        skip_tls_verify: skForm.skip_tls_verify
      };
      if (skForm.clear_api_key) {
        req.api_key = '';
      } else if (skForm.api_key.trim()) {
        req.api_key = skForm.api_key.trim();
      }
      const res = await testSafeKeepConnection(editing.id, req);
      const list = res.upstreams || [];
      setRemoteUpstreams(list);
      setTestResultMsg({
        ok: true,
        text:
          list.length > 0
            ? t('cronSafeKeepTestSuccess', { count: list.length })
            : t('cronSafeKeepTestSuccessEmpty')
      });
    } catch (error) {
      if (!onUnauthorized(error)) {
        const msg = error instanceof Error ? error.message : String(error);
        setTestResultMsg({ ok: false, text: t('cronSafeKeepTestFailed', { error: msg }) });
      }
    } finally {
      setTestingSafeKeep(false);
    }
  };

  return (
    <section className="admin-section">
      <div className="module-intro">
        <p>{t('cronSubtitle')}</p>
        <button type="button" onClick={() => void loadJobs()} disabled={loading}>
          <RefreshCw className={loading ? 'spin' : undefined} size={15} aria-hidden="true" />
          {t('cronRefreshRuns')}
        </button>
      </div>

      {loading ? (
        <div className="admin-state">
          <span>{t('cronLoading')}</span>
        </div>
      ) : jobs.length === 0 ? (
        <div className="admin-state">
          <span>{t('cronEmpty')}</span>
        </div>
      ) : (
        <>
          <div className="admin-table-wrap">
            <table className="admin-table cron-table">
              <thead>
                <tr>
                  <th>{t('cronColJob')}</th>
                  <th>{t('cronColSchedule')}</th>
                  <th>{t('cronColNextRun')}</th>
                  <th>{t('cronColLastRun')}</th>
                  <th>{t('cronColEnabled')}</th>
                  <th>{t('actions')}</th>
                </tr>
              </thead>
              <tbody>
                {jobs.map((job) => {
                  const scheduleText = describeCron(job.cron_expr, language);
                  const nextTime = job.enabled ? job.next_runs?.[0] || job.next_run_at : undefined;
                  const isRunning = runningId === job.id;
                  const displayName = jobName(job);
                  return (
                    <tr key={job.id}>
                      <td className="site-cell">
                        <strong>{displayName}</strong>
                        <span className="cron-desc-cell">{jobDesc(job)}</span>
                      </td>
                      <td>
                        <div className="cron-schedule-cell">
                          {scheduleText && <span>{scheduleText}</span>}
                          <code className="mono-badge">{job.cron_expr}</code>
                        </div>
                      </td>
                      <td className="nowrap-cell">
                        {nextTime ? formatDateTime(nextTime, language) : <span className="muted-dash">—</span>}
                      </td>
                      <td>
                        {!job.last_run_at ? (
                          <span className="muted-dash">—</span>
                        ) : (
                          <div className="cron-last-run">
                            <div className="cron-last-run-top">
                              <JobStatusPill status={job.last_status} />
                              <span>{formatDateTime(job.last_run_at, language)}</span>
                            </div>
                            {job.last_summary && (
                              <span className="cron-summary" title={job.last_summary}>
                                {job.last_summary}
                              </span>
                            )}
                          </div>
                        )}
                      </td>
                      <td>
                        <ToggleSwitch
                          checked={job.enabled}
                          disabled={togglingId === job.id}
                          label={displayName}
                          onChange={(next) => void handleToggle(job, next)}
                        />
                      </td>
                      <td className="actions-cell">
                        <div className="row-actions cron-row-actions">
                          <button
                            type="button"
                            disabled={Boolean(runningId)}
                            onClick={() => void handleRunNow(job)}
                            title={t('cronRunNow')}
                          >
                            {isRunning ? <Loader2 size={15} className="spin" aria-hidden="true" /> : <Play size={15} aria-hidden="true" />}
                            <span>{isRunning ? t('cronRunning') : t('cronRunNow')}</span>
                          </button>
                          <button type="button" onClick={() => openEdit(job)} title={t('cronEdit')}>
                            <Pencil size={15} aria-hidden="true" />
                            <span>{t('cronEdit')}</span>
                          </button>
                          <button type="button" onClick={() => openRuns(job)} title={t('cronRuns')}>
                            <History size={15} aria-hidden="true" />
                            <span>{t('cronRuns')}</span>
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          <div className="admin-mobile-list">
            {jobs.map((job) => {
              const scheduleText = describeCron(job.cron_expr, language);
              const nextTime = job.enabled ? job.next_runs?.[0] || job.next_run_at : undefined;
              const isRunning = runningId === job.id;
              const displayName = jobName(job);
              return (
                <article className="mobile-admin-card" key={job.id}>
                  <div className="mobile-card-head">
                    <div className="mobile-card-title">
                      <strong>{displayName}</strong>
                      <span>{scheduleText ? `${scheduleText} · ${job.cron_expr}` : job.cron_expr}</span>
                    </div>
                    <ToggleSwitch
                      checked={job.enabled}
                      disabled={togglingId === job.id}
                      label={displayName}
                      onChange={(next) => void handleToggle(job, next)}
                    />
                  </div>
                  <p className="mobile-card-description">{jobDesc(job)}</p>
                  <div className="mobile-card-meta">
                    {nextTime && <span className="status-pill">{t('cronColNextRun')}: {formatDateTime(nextTime, language)}</span>}
                    {job.last_run_at && <JobStatusPill status={job.last_status} />}
                  </div>
                  {job.last_summary && <p className="mobile-card-description">{job.last_summary}</p>}
                  <div className="cron-mobile-actions">
                    <button type="button" disabled={Boolean(runningId)} onClick={() => void handleRunNow(job)}>
                      {isRunning ? <Loader2 size={15} className="spin" aria-hidden="true" /> : <Play size={15} aria-hidden="true" />}
                      {isRunning ? t('cronRunning') : t('cronRunNow')}
                    </button>
                    <button type="button" onClick={() => openEdit(job)}>
                      <Pencil size={15} aria-hidden="true" />
                      {t('cronEdit')}
                    </button>
                    <button type="button" onClick={() => openRuns(job)}>
                      <History size={15} aria-hidden="true" />
                      {t('cronRuns')}
                    </button>
                  </div>
                </article>
              );
            })}
          </div>
        </>
      )}

      {editing && params && (
        <div className="panel-backdrop" onClick={closeEdit}>
          <aside className="side-panel side-panel-wide" onClick={(e) => e.stopPropagation()}>
            <header>
              <div>
                <h2>{jobName(editing)}</h2>
                <p className="panel-subtitle">{jobDesc(editing)}</p>
              </div>
              <button type="button" onClick={closeEdit} title={t('close')}>
                <X size={18} aria-hidden="true" />
              </button>
            </header>

            <form className="admin-form" onSubmit={handleSave}>
              {formError && <div className="inline-alert inline-alert-danger">{formError}</div>}

              <label>
                <span>{t('cronFieldSchedule')}</span>
                <select value={preset} onChange={(e) => handlePresetChange(e.target.value)}>
                  {SCHEDULE_PRESETS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.value === 'custom'
                        ? t(opt.labelKey)
                        : language === 'en'
                          ? `${t(opt.labelKey)} (${opt.value})`
                          : `${t(opt.labelKey)}（${opt.value}）`}
                    </option>
                  ))}
                </select>
              </label>

              {preset === 'custom' && (
                <label>
                  <span>{t('cronFieldExpr')}</span>
                  <input
                    className="mono-input"
                    value={cronExpr}
                    onChange={(e) => setCronExpr(e.target.value)}
                    required
                  />
                  <small className="field-hint">
                    {t('cronFieldExprHint')}
                    <code>{editing.default_schedule || editing.cron_expr}</code>
                  </small>
                </label>
              )}

              <div className="cron-preview-block">
                <div className="cron-preview-head">
                  <span>{t('cronNextRuns')}</span>
                  {previewLoading && <Loader2 size={14} className="spin" aria-hidden="true" />}
                </div>
                {previewError ? (
                  <div className="inline-alert inline-alert-danger">{previewError}</div>
                ) : (
                  <div className="cron-preview-list">
                    {previewRaw.length > 0 ? (
                      previewRaw.map((rawTime) => (
                        <div key={rawTime} className="cron-preview-item">
                          {formatDateTime(rawTime, language)}
                        </div>
                      ))
                    ) : (
                      <span className="muted-dash">—</span>
                    )}
                  </div>
                )}
              </div>

              <div className="form-grid">
                <label>
                  <span>{t('cronFieldTimeout')}</span>
                  <input
                    type="number"
                    min={1}
                    max={86400}
                    value={timeoutSec}
                    onChange={(e) => setTimeoutSec(Number(e.target.value))}
                    required
                  />
                </label>
                <div className="switch-field">
                  <span className="switch-field-label">{t('cronFieldEnabled')}</span>
                  <div className="switch-field-row">
                    <ToggleSwitch checked={enabled} label={t('cronFieldEnabled')} onChange={setEnabled} />
                    <span>{enabled ? t('enabledStatus') : t('cronStatusPaused')}</span>
                  </div>
                </div>
              </div>

              {(params.kind === 'cleanup_audit_logs' || params.kind === 'cleanup_cron_runs') && (
                <div className="panel-divider">
                  <label>
                    <span>{t('cronFieldRetentionDays')}</span>
                    <input
                      type="number"
                      min={1}
                      max={36500}
                      value={params.data.retention_days}
                      onChange={(e) =>
                        setParams({
                          kind: params.kind,
                          data: { retention_days: Number(e.target.value) }
                        })
                      }
                      required
                    />
                    <small className="field-hint">
                      {params.kind === 'cleanup_audit_logs' ? t('cronAuditRetentionHint') : t('cronRunsRetentionHint')}
                    </small>
                  </label>
                </div>
              )}

              {skForm && (
                <div className="panel-divider stack">
                  <label>
                    <span>{t('cronSafeKeepBaseUrl')}</span>
                    <input
                      className="mono-input"
                      value={skForm.api_base_url}
                      onChange={(e) => setSkForm({ ...skForm, api_base_url: e.target.value })}
                      placeholder="https://safekeep.example.com"
                    />
                    <small className="field-hint">{t('cronSafeKeepBaseUrlHint')}</small>
                  </label>

                  <div className="secret-block">
                    <div className="secret-block-head">
                      <span>{t('cronSafeKeepApiKey')}</span>
                      {skForm.has_api_key && (
                        <button
                          type="button"
                          className="link-danger-btn"
                          onClick={() =>
                            setSkForm({ ...skForm, api_key: '', has_api_key: false, clear_api_key: true })
                          }
                        >
                          {t('cronSafeKeepClearKey')}
                        </button>
                      )}
                    </div>
                    <SecretInput
                      hasSaved={skForm.has_api_key}
                      value={skForm.api_key}
                      onChange={(v) => setSkForm({ ...skForm, api_key: v, clear_api_key: false })}
                      onReveal={async () => (await revealCronSecrets(editing.id)).api_key || ''}
                      onNotice={onNotice}
                      onUnauthorized={onUnauthorized}
                      placeholder={
                        skForm.has_api_key ? t('cronSafeKeepApiKeySaved') : t('cronSafeKeepApiKeyPlaceholder')
                      }
                    />
                  </div>

                  <div className="safekeep-test-row">
                    <button
                      type="button"
                      className="secondary-btn"
                      disabled={testingSafeKeep || !skForm.api_base_url.trim()}
                      onClick={() => void handleTestSafeKeep()}
                    >
                      {testingSafeKeep ? (
                        <Loader2 size={15} className="spin" aria-hidden="true" />
                      ) : (
                        <PlugZap size={15} aria-hidden="true" />
                      )}
                      {testingSafeKeep ? t('cronSafeKeepTesting') : t('cronSafeKeepTestBtn')}
                    </button>
                    {testResultMsg && (
                      <div className={testResultMsg.ok ? 'inline-alert inline-alert-success' : 'inline-alert inline-alert-danger'}>
                        {testResultMsg.text}
                      </div>
                    )}
                  </div>

                  <label>
                    <span>{t('cronSafeKeepPath')}</span>
                    <input
                      value={skForm.path}
                      onChange={(e) => setSkForm({ ...skForm, path: e.target.value })}
                      placeholder="navbox-backups"
                    />
                    <small className="field-hint">{t('cronSafeKeepPathHint')}</small>
                  </label>

                  <div className="targets-field">
                    <span className="switch-field-label">{t('cronSafeKeepTargets')}</span>
                    {skForm.targets.length > 0 && (
                      <div className="target-chips">
                        {skForm.targets.map((tag) => (
                          <span className="target-chip" key={tag}>
                            <code>{tag}</code>
                            <button type="button" onClick={() => removeTargetTag(tag)} aria-label={tag}>
                              <X size={12} aria-hidden="true" />
                            </button>
                          </span>
                        ))}
                      </div>
                    )}
                    <div className="target-input-row">
                      <input
                        className="mono-input"
                        value={targetInput}
                        onChange={(e) => setTargetInput(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            addTargetTag(targetInput);
                          }
                        }}
                        placeholder={t('cronSafeKeepTargetsPlaceholder')}
                      />
                      <button
                        type="button"
                        className="secondary-btn"
                        disabled={!targetInput.trim()}
                        onClick={() => addTargetTag(targetInput)}
                      >
                        <Plus size={15} aria-hidden="true" />
                        {t('cronSafeKeepAddTarget')}
                      </button>
                    </div>
                    {remoteUpstreams.length > 0 && (
                      <div className="upstream-suggestions">
                        {remoteUpstreams.map((u) => {
                          const name = u.upstream_name || u.upstream_id;
                          const active = skForm.targets.includes(name) || skForm.targets.includes(u.upstream_id);
                          return (
                            <button
                              key={u.upstream_id}
                              type="button"
                              className={active ? 'upstream-chip active' : 'upstream-chip'}
                              onClick={() => {
                                if (active) {
                                  setSkForm({
                                    ...skForm,
                                    targets: skForm.targets.filter((item) => item !== name && item !== u.upstream_id)
                                  });
                                } else {
                                  setSkForm({ ...skForm, targets: [...skForm.targets, name] });
                                }
                              }}
                            >
                              {u.upstream_name} ({u.upstream_type?.toUpperCase()})
                            </button>
                          );
                        })}
                      </div>
                    )}
                    <small className="field-hint">{t('cronSafeKeepTargetsHint')}</small>
                  </div>

                  <div className="form-grid">
                    <label>
                      <span>{t('cronSafeKeepMode')}</span>
                      <select
                        value={skForm.mode}
                        onChange={(e) => setSkForm({ ...skForm, mode: e.target.value as 'sync' | 'async' })}
                      >
                        <option value="sync">{t('cronSafeKeepModeSync')}</option>
                        <option value="async">{t('cronSafeKeepModeAsync')}</option>
                      </select>
                    </label>
                    <label>
                      <span>{t('cronSafeKeepRename')}</span>
                      <input
                        value={skForm.rename}
                        onChange={(e) => setSkForm({ ...skForm, rename: e.target.value })}
                        placeholder={t('cronSafeKeepRenamePlaceholder')}
                      />
                    </label>
                  </div>

                  <div className="stack-tight">
                    <label className="checkbox-line">
                      <input
                        type="checkbox"
                        checked={skForm.skip_dedup}
                        onChange={(e) => setSkForm({ ...skForm, skip_dedup: e.target.checked })}
                      />
                      <span>{t('cronSafeKeepSkipDedup')}</span>
                    </label>
                    <label className="checkbox-line">
                      <input
                        type="checkbox"
                        checked={skForm.skip_tls_verify}
                        onChange={(e) => setSkForm({ ...skForm, skip_tls_verify: e.target.checked })}
                      />
                      <span>{t('cronSafeKeepSkipTls')}</span>
                    </label>
                  </div>
                </div>
              )}

              <div className="panel-footer-actions">
                <button type="button" className="secondary-btn" disabled={saving} onClick={closeEdit}>
                  {t('cancel')}
                </button>
                <button className="primary-button" type="submit" disabled={saving || Boolean(previewError)}>
                  <Save size={16} aria-hidden="true" />
                  {saving ? t('cronSaving') : t('cronSave')}
                </button>
              </div>
            </form>
          </aside>
        </div>
      )}

      {runsJob && (
        <div className="panel-backdrop" onClick={() => setRunsJob(null)}>
          <aside className="side-panel side-panel-wide" onClick={(e) => e.stopPropagation()}>
            <header>
              <h2>{t('cronRunsTitle', { name: jobName(runsJob) })}</h2>
              <div className="panel-head-actions">
                <button
                  type="button"
                  disabled={runsLoading}
                  onClick={() => void loadRuns(runsJob)}
                  title={t('cronRefreshRuns')}
                >
                  <RefreshCw size={15} className={runsLoading ? 'spin' : undefined} aria-hidden="true" />
                </button>
                <button type="button" onClick={() => setRunsJob(null)} title={t('close')}>
                  <X size={18} aria-hidden="true" />
                </button>
              </div>
            </header>

            {runsLoading && <div className="admin-state"><span>{t('loading')}</span></div>}
            {runsError && <div className="inline-alert inline-alert-danger">{runsError}</div>}
            {!runsLoading && !runsError && runs.length === 0 && (
              <div className="admin-state"><span>{t('cronNoRuns')}</span></div>
            )}
            {!runsLoading && runs.length > 0 && (
              <div className="cron-runs-list">
                {runs.map((run) => {
                  const dur = formatDuration(run.started_at, run.finished_at);
                  return (
                    <article className="cron-run-item" key={run.id}>
                      <div className="cron-run-meta">
                        <JobStatusPill status={run.status} />
                        <strong>{formatDateTime(run.started_at, language)}</strong>
                        <span>
                          {run.trigger_type === 'manual'
                            ? t('cronTriggerManual', { by: run.triggered_by })
                            : t('cronTriggerScheduled')}
                        </span>
                        {dur && <span>{t('cronRunDuration', { dur })}</span>}
                        {run.affected_rows > 0 && <span>{t('cronRunAffected', { count: run.affected_rows })}</span>}
                      </div>
                      {run.output_log && <pre className="cron-run-log">{run.output_log}</pre>}
                    </article>
                  );
                })}
              </div>
            )}
          </aside>
        </div>
      )}
    </section>
  );
}

function ToggleSwitch({
  checked,
  disabled,
  label,
  onChange
}: {
  checked: boolean;
  disabled?: boolean;
  label: string;
  onChange: (next: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={checked ? 'toggle-switch checked' : 'toggle-switch'}
    >
      <span className="toggle-switch-knob" />
    </button>
  );
}

function JobStatusPill({ status }: { status: string }) {
  const { t } = usePreferences();
  const map: Record<string, { cls: string; label: TranslationKey }> = {
    succeeded: { cls: 'status-pill status-success', label: 'cronStatusSucceeded' },
    success: { cls: 'status-pill status-success', label: 'cronStatusSucceeded' },
    failed: { cls: 'status-pill status-danger', label: 'cronStatusFailed' },
    failure: { cls: 'status-pill status-danger', label: 'cronStatusFailed' },
    timeout: { cls: 'status-pill status-danger', label: 'cronStatusTimeout' },
    skipped: { cls: 'status-pill status-warning', label: 'cronStatusSkipped' },
    running: { cls: 'status-pill status-info', label: 'cronStatusRunning' },
    never: { cls: 'status-pill', label: 'cronStatusNever' }
  };
  const info = map[status];
  return <span className={info ? info.cls : 'status-pill'}>{info ? t(info.label) : status}</span>;
}

function SecretInput({
  value,
  hasSaved,
  placeholder,
  onChange,
  onReveal,
  onNotice,
  onUnauthorized
}: {
  value: string;
  hasSaved: boolean;
  placeholder: string;
  onChange: (next: string) => void;
  onReveal: () => Promise<string>;
  onNotice: (msg: string) => void;
  onUnauthorized: (err: unknown) => boolean;
}) {
  const { t } = usePreferences();
  const [editing, setEditing] = useState(false);
  const [showPlain, setShowPlain] = useState(false);
  const [revealed, setRevealed] = useState<string | null>(null);
  const [revealing, setRevealing] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    setEditing(false);
    setShowPlain(false);
    setRevealed(null);
  }, [hasSaved]);

  const showingSavedMask = hasSaved && !editing;
  const displayValue = showingSavedMask ? (showPlain && revealed !== null ? revealed : '*****') : value;
  const canRevealOrCopy = hasSaved || value !== '';

  const fetchSecret = async (): Promise<string | null> => {
    if (revealed !== null) {
      return revealed;
    }
    setRevealing(true);
    try {
      const secret = await onReveal();
      setRevealed(secret);
      return secret;
    } catch (error) {
      if (!onUnauthorized(error)) {
        const msg = error instanceof Error ? error.message : String(error);
        onNotice(t('cronSafeKeepRevealFailed', { error: msg }));
      }
      return null;
    } finally {
      setRevealing(false);
    }
  };

  const handleToggleVisibility = async () => {
    if (revealing) {
      return;
    }
    if (showPlain) {
      setShowPlain(false);
      return;
    }
    if (showingSavedMask) {
      const secret = await fetchSecret();
      if (secret === null) {
        return;
      }
    }
    setShowPlain(true);
  };

  const handleCopy = async () => {
    if (revealing) {
      return;
    }
    let text = value;
    if (showingSavedMask) {
      const secret = await fetchSecret();
      if (!secret) {
        return;
      }
      text = secret;
    }
    if (!text) {
      return;
    }
    try {
      await copyToClipboard(text);
      setCopied(true);
      onNotice(t('cronSafeKeepCopied'));
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore clipboard failures
    }
  };

  return (
    <div className="secret-input-row">
      <div className="secret-input-wrap">
        <input
          className="mono-input"
          type={showingSavedMask ? 'text' : showPlain ? 'text' : 'password'}
          autoComplete="off"
          value={displayValue}
          placeholder={placeholder}
          onFocus={() => {
            if (hasSaved && !editing) {
              setEditing(true);
              onChange('');
            }
          }}
          onBlur={() => {
            if (editing && value === '' && hasSaved) {
              setEditing(false);
            }
          }}
          onChange={(e) => {
            if (!editing) {
              setEditing(true);
            }
            onChange(e.target.value);
          }}
        />
        <button
          type="button"
          className="secret-eye-btn"
          disabled={!canRevealOrCopy || revealing}
          onClick={() => void handleToggleVisibility()}
          title={showPlain ? t('cronSafeKeepHideKey') : t('cronSafeKeepShowKey')}
          aria-label={showPlain ? t('cronSafeKeepHideKey') : t('cronSafeKeepShowKey')}
        >
          {revealing ? (
            <Loader2 size={14} className="spin" aria-hidden="true" />
          ) : showPlain ? (
            <EyeOff size={14} aria-hidden="true" />
          ) : (
            <Eye size={14} aria-hidden="true" />
          )}
        </button>
      </div>
      <button
        type="button"
        className="secondary-btn"
        disabled={!canRevealOrCopy || revealing}
        onClick={() => void handleCopy()}
      >
        {copied ? <Check size={14} aria-hidden="true" /> : <Copy size={14} aria-hidden="true" />}
        {copied ? t('cronSafeKeepCopied') : t('cronSafeKeepCopyKey')}
      </button>
    </div>
  );
}
