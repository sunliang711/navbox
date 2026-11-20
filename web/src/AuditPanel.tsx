import { Code2, Download, Eye, RotateCcw, Search, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { getAuditLogsExportUrl, listAuditLogs } from './api';
import {
  AUDIT_ACTION_GROUPS,
  formatAuditAction,
  formatAuditActor,
  formatAuditDetailParams,
  formatAuditResource,
  parseAuditDetail
} from './auditUtils';
import { formatDateTime } from './cronUtils';
import { usePreferences } from './preferences';
import type { AuditLog, AuditQuery } from './types';

type FilterState = {
  actorType: string;
  status: string;
  action: string;
  q: string;
  fromDate: string;
  toDate: string;
};

const emptyFilters: FilterState = {
  actorType: '',
  status: '',
  action: '',
  q: '',
  fromDate: '',
  toDate: ''
};

export function AuditPanel({ onUnauthorized }: { onUnauthorized: (err: unknown) => boolean }) {
  const { t, language } = usePreferences();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const limit = 50;

  const [filters, setFilters] = useState<FilterState>(emptyFilters);
  const [appliedQuery, setAppliedQuery] = useState<AuditQuery>({});
  const [pageErr, setPageErr] = useState('');
  const [detail, setDetail] = useState<AuditLog | null>(null);
  const [showRawJson, setShowRawJson] = useState(false);
  const reqSeq = useRef(0);

  const toAuditQuery = (f: FilterState, targetOffset: number): AuditQuery => {
    const query: AuditQuery = {
      limit,
      offset: targetOffset
    };
    if (f.actorType) query.actor_type = f.actorType;
    if (f.status) query.status = f.status;
    if (f.action.trim()) query.action = f.action.trim();
    if (f.q.trim()) query.q = f.q.trim();
    if (f.fromDate) query.from = new Date(f.fromDate).toISOString();
    if (f.toDate) query.to = new Date(f.toDate).toISOString();
    return query;
  };

  const load = async (targetOffset = offset, nextFilters = filters) => {
    const seq = ++reqSeq.current;
    setLoading(true);
    setPageErr('');
    try {
      if (nextFilters.fromDate && nextFilters.toDate && new Date(nextFilters.fromDate) > new Date(nextFilters.toDate)) {
        throw new Error(t('auditErrDateRange'));
      }
      const query = toAuditQuery(nextFilters, targetOffset);
      const res = await listAuditLogs(query);
      if (seq !== reqSeq.current) {
        return;
      }
      setAppliedQuery(query);
      setLogs(res.items || []);
      setTotal(res.total || 0);
    } catch (error) {
      if (seq !== reqSeq.current) {
        return;
      }
      if (!onUnauthorized(error)) {
        const msg = error instanceof Error ? error.message : String(error);
        setPageErr(t('auditLoadFailed', { error: msg }));
      }
    } finally {
      if (seq === reqSeq.current) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    setOffset(0);
    void load(0, filters);
  }, [filters.actorType, filters.status, filters.action]);

  const handleSearch = () => {
    setOffset(0);
    void load(0, filters);
  };

  const resetFilters = () => {
    setFilters(emptyFilters);
    setOffset(0);
    void load(0, emptyFilters);
  };

  const openDetail = (item: AuditLog) => {
    setShowRawJson(false);
    setDetail(item);
  };

  const hasActiveFilters = Boolean(
    filters.q || filters.action || filters.actorType || filters.status || filters.fromDate || filters.toDate
  );

  const csvUrl = getAuditLogsExportUrl('csv', appliedQuery);
  const jsonUrl = getAuditLogsExportUrl('json', appliedQuery);

  return (
    <section className="admin-section">
      <div className="module-intro">
        <p>{t('auditSubtitle', { total })}</p>
        <div className="audit-export-actions">
          <a className="secondary-btn" href={csvUrl}>
            <Download size={15} aria-hidden="true" />
            {t('auditExportCsv')}
          </a>
          <a className="secondary-btn" href={jsonUrl}>
            <Download size={15} aria-hidden="true" />
            {t('auditExportJson')}
          </a>
        </div>
      </div>

      <div className="audit-filter-bar">
        <label className="admin-search-field audit-search-input">
          <Search size={16} aria-hidden="true" />
          <input
            value={filters.q}
            onChange={(e) => setFilters({ ...filters, q: e.target.value })}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleSearch();
            }}
            placeholder={t('auditSearchPlaceholder')}
          />
        </label>
        <select
          className="audit-filter-select"
          value={filters.action}
          onChange={(e) => setFilters({ ...filters, action: e.target.value })}
          aria-label={t('auditColAction')}
        >
          <option value="">{t('auditActionAll')}</option>
          {AUDIT_ACTION_GROUPS.map((group) => (
            <optgroup key={group.labelKey} label={t(group.labelKey)}>
              {group.options.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {t(opt.labelKey)}
                </option>
              ))}
            </optgroup>
          ))}
        </select>
        <select
          className="audit-filter-select"
          value={filters.actorType}
          onChange={(e) => setFilters({ ...filters, actorType: e.target.value })}
          aria-label={t('auditColActor')}
        >
          <option value="">{t('auditActorAll')}</option>
          <option value="admin">{t('auditActorAdmin')}</option>
          <option value="system">{t('auditActorSystem')}</option>
        </select>
        <select
          className="audit-filter-select"
          value={filters.status}
          onChange={(e) => setFilters({ ...filters, status: e.target.value })}
          aria-label={t('auditColResult')}
        >
          <option value="">{t('auditStatusAll')}</option>
          <option value="success">{t('auditStatusSuccess')}</option>
          <option value="failure">{t('auditStatusFailure')}</option>
        </select>
        <input
          className="audit-filter-date"
          type="datetime-local"
          value={filters.fromDate}
          onChange={(e) => setFilters({ ...filters, fromDate: e.target.value })}
          aria-label={t('auditFromDate')}
          title={t('auditFromDate')}
        />
        <input
          className="audit-filter-date"
          type="datetime-local"
          value={filters.toDate}
          onChange={(e) => setFilters({ ...filters, toDate: e.target.value })}
          aria-label={t('auditToDate')}
          title={t('auditToDate')}
        />
        <button className="primary-button" type="button" disabled={loading} onClick={handleSearch}>
          {loading ? t('auditSearching') : t('auditSearchBtn')}
        </button>
        {hasActiveFilters && (
          <button className="secondary-btn" type="button" onClick={resetFilters}>
            <RotateCcw size={14} aria-hidden="true" />
            {t('auditResetFilters')}
          </button>
        )}
      </div>

      {pageErr && <div className="inline-alert inline-alert-danger">{pageErr}</div>}

      {loading ? (
        <div className="admin-state">
          <span>{t('loading')}</span>
        </div>
      ) : logs.length === 0 ? (
        <div className="admin-state">
          <span>{t('auditEmpty')}</span>
        </div>
      ) : (
        <>
          <div className="admin-table-wrap">
            <table className="admin-table audit-table">
              <thead>
                <tr>
                  <th>{t('auditColTime')}</th>
                  <th>{t('auditColActor')}</th>
                  <th>{t('auditColAction')}</th>
                  <th>{t('auditColResource')}</th>
                  <th>{t('auditColIp')}</th>
                  <th>{t('auditColResult')}</th>
                  <th>{t('auditColDetail')}</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((item) => {
                  const actor = formatAuditActor(item, t);
                  const action = formatAuditAction(item.action, t);
                  const resource = formatAuditResource(item, t, language);
                  return (
                    <tr key={item.id}>
                      <td className="nowrap-cell">{formatDateTime(item.created_at, language)}</td>
                      <td className="nowrap-cell">
                        <span className="table-tag">{actor.badge}</span>
                        {actor.name && <strong className="audit-actor-name"> {actor.name}</strong>}
                      </td>
                      <td className="nowrap-cell">
                        <span className="audit-action-label" title={item.action}>
                          {action.label}
                        </span>
                      </td>
                      <td className="audit-resource-cell">
                        <div className="audit-resource-main">
                          <span className="audit-resource-type">{resource.typeLabel}</span>
                          {resource.target && <span className="audit-resource-target"> · {resource.target}</span>}
                        </div>
                        {resource.summary && (
                          <div className="audit-resource-summary" title={resource.summary}>
                            {resource.summary}
                          </div>
                        )}
                      </td>
                      <td className="nowrap-cell">
                        <code className="mono-muted">{item.ip_address || '-'}</code>
                      </td>
                      <td>
                        <span
                          className={
                            item.status === 'success'
                              ? 'status-pill status-success'
                              : 'status-pill status-danger'
                          }
                        >
                          {item.status === 'success' ? t('auditStatusSuccess') : t('auditStatusFailure')}
                        </span>
                      </td>
                      <td className="actions-cell">
                        <div className="row-actions">
                          <button type="button" onClick={() => openDetail(item)} title={t('auditViewDetail')}>
                            <Eye size={15} aria-hidden="true" />
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
            {logs.map((item) => {
              const actor = formatAuditActor(item, t);
              const action = formatAuditAction(item.action, t);
              const resource = formatAuditResource(item, t, language);
              return (
                <article className="mobile-admin-card" key={item.id}>
                  <div className="mobile-card-head">
                    <div className="mobile-card-title">
                      <strong title={item.action}>{action.label}</strong>
                      <span>{formatDateTime(item.created_at, language)}</span>
                    </div>
                    <span
                      className={
                        item.status === 'success'
                          ? 'status-pill status-success'
                          : 'status-pill status-danger'
                      }
                    >
                      {item.status === 'success' ? t('auditStatusSuccess') : t('auditStatusFailure')}
                    </span>
                  </div>
                  <div className="mobile-card-meta">
                    <span className="table-tag">{actor.name ? `${actor.badge} · ${actor.name}` : actor.badge}</span>
                    <span className="status-pill">
                      {resource.typeLabel}
                      {resource.target ? ` · ${resource.target}` : ''}
                    </span>
                    {item.ip_address && <span className="status-pill">{item.ip_address}</span>}
                  </div>
                  {resource.summary && <p className="audit-mobile-summary">{resource.summary}</p>}
                  <div className="cron-mobile-actions">
                    <button type="button" onClick={() => openDetail(item)}>
                      <Eye size={15} aria-hidden="true" />
                      {t('auditViewDetail')}
                    </button>
                  </div>
                </article>
              );
            })}
          </div>
        </>
      )}

      <div className="audit-pagination">
        <span>{t('auditPageSummary', { page: Math.floor(offset / limit) + 1, total })}</span>
        <div className="audit-pagination-btns">
          <button
            type="button"
            className="secondary-btn"
            disabled={loading || offset === 0}
            onClick={() => {
              const next = Math.max(0, offset - limit);
              setOffset(next);
              void load(next, filters);
            }}
          >
            {t('auditPrevPage')}
          </button>
          <button
            type="button"
            className="secondary-btn"
            disabled={loading || offset + limit >= total}
            onClick={() => {
              const next = offset + limit;
              setOffset(next);
              void load(next, filters);
            }}
          >
            {t('auditNextPage')}
          </button>
        </div>
      </div>

      {detail && (() => {
        const actor = formatAuditActor(detail, t);
        const action = formatAuditAction(detail.action, t);
        const resource = formatAuditResource(detail, t, language);
        const paramRows = formatAuditDetailParams(detail, t, language);
        const showResourceId = Boolean(
          detail.resource_id && !(detail.resource_type === 'admin_setting' && detail.resource_id === '1')
        );
        return (
          <div className="confirm-backdrop" onClick={() => setDetail(null)} role="presentation">
            <div
              className="audit-detail-modal"
              role="dialog"
              aria-modal="true"
              aria-label={t('auditDetailTitle', { id: detail.id })}
              onClick={(e) => e.stopPropagation()}
            >
              <header>
                <h2>{t('auditDetailTitle', { id: detail.id })}</h2>
                <button type="button" onClick={() => setDetail(null)} title={t('close')}>
                  <X size={18} aria-hidden="true" />
                </button>
              </header>

              <div className="audit-detail-body">
                <section className="audit-detail-section">
                  <h3>{t('auditDetailSectionSummary')}</h3>
                  <dl className="audit-detail-kv">
                    <div>
                      <dt>{t('auditDetailFieldAction')}</dt>
                      <dd>
                        <strong>{action.label}</strong>
                        <code className="audit-detail-code-tag">{detail.action}</code>
                      </dd>
                    </div>
                    <div>
                      <dt>{t('auditDetailFieldResult')}</dt>
                      <dd>
                        <span
                          className={
                            detail.status === 'success'
                              ? 'status-pill status-success'
                              : 'status-pill status-danger'
                          }
                        >
                          {detail.status === 'success' ? t('auditStatusSuccess') : t('auditStatusFailure')}
                        </span>
                      </dd>
                    </div>
                    <div>
                      <dt>{t('auditDetailFieldTime')}</dt>
                      <dd>{formatDateTime(detail.created_at, language)}</dd>
                    </div>
                    <div>
                      <dt>{t('auditDetailFieldActor')}</dt>
                      <dd>{actor.name ? `${actor.badge} · ${actor.name}` : actor.badge}</dd>
                    </div>
                    <div>
                      <dt>{t('auditDetailFieldResource')}</dt>
                      <dd>
                        {resource.typeLabel}
                        {resource.target ? ` · ${resource.target}` : ''}
                      </dd>
                    </div>
                    {showResourceId && (
                      <div>
                        <dt>{t('auditDetailFieldResourceId')}</dt>
                        <dd>
                          <code className="mono-muted">{detail.resource_id}</code>
                        </dd>
                      </div>
                    )}
                  </dl>
                </section>

                <section className="audit-detail-section">
                  <h3>{t('auditDetailSectionParams')}</h3>
                  {paramRows.length === 0 ? (
                    <p className="audit-detail-empty">{t('auditDetailNoExtra')}</p>
                  ) : (
                    <dl className="audit-detail-kv">
                      {paramRows.map((row, idx) => (
                        <div key={`${row.label}-${idx}`}>
                          <dt>{row.label}</dt>
                          <dd className={row.mono ? 'audit-detail-mono' : undefined}>{row.value}</dd>
                        </div>
                      ))}
                    </dl>
                  )}
                </section>

                {(detail.ip_address || detail.request_id || detail.user_agent) && (
                  <section className="audit-detail-section">
                    <h3>{t('auditDetailSectionTech')}</h3>
                    <dl className="audit-detail-kv">
                      {detail.ip_address && (
                        <div>
                          <dt>{t('auditDetailFieldIp')}</dt>
                          <dd className="audit-detail-mono">{detail.ip_address}</dd>
                        </div>
                      )}
                      {detail.request_id && (
                        <div>
                          <dt>{t('auditDetailFieldRequestId')}</dt>
                          <dd className="audit-detail-mono">{detail.request_id}</dd>
                        </div>
                      )}
                      {detail.user_agent && (
                        <div>
                          <dt>{t('auditDetailFieldUserAgent')}</dt>
                          <dd className="audit-detail-mono">{detail.user_agent}</dd>
                        </div>
                      )}
                    </dl>
                  </section>
                )}

                {showRawJson && (
                  <pre className="audit-detail-pre">
                    {JSON.stringify(
                      {
                        ...detail,
                        detail: parseAuditDetail(detail.detail_json)
                      },
                      null,
                      2
                    )}
                  </pre>
                )}
              </div>

              <div className="audit-detail-footer">
                <button
                  type="button"
                  className="secondary-btn"
                  onClick={() => setShowRawJson((prev) => !prev)}
                >
                  <Code2 size={14} aria-hidden="true" />
                  {showRawJson ? t('auditDetailHideRaw') : t('auditDetailShowRaw')}
                </button>
                <button type="button" className="secondary-btn" onClick={() => setDetail(null)}>
                  {t('close')}
                </button>
              </div>
            </div>
          </div>
        );
      })()}
    </section>
  );
}
