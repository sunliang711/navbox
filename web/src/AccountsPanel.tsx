import { KeyRound, Loader2, LogOut, Monitor, Pencil, Plus, RefreshCw, Save, Shield, Users, X } from 'lucide-react';
import { type FormEvent, useEffect, useRef, useState } from 'react';
import {
  changeAdminPassword,
  createAdminAccount,
  getAdminSession,
  isUnauthorizedError,
  listActiveSessions,
  listAdminAccounts,
  revokeSession,
  updateAdminAccount
} from './api';
import { formatDateTime } from './cronUtils';
import { usePreferences } from './preferences';
import type { AccountUpdate, ActiveSession, AdminAccount } from './types';
import './accounts.css';

type AccountAction = {
  kind: 'create' | 'rename' | 'role' | 'disabled' | 'password' | 'session';
  title: string;
  account?: AdminAccount;
  session?: ActiveSession;
};

export function AccountsPanel({ user, onUserChange, onNotice, onUnauthorized, onSignedOut, reauthVersion }: {
  user: AdminAccount;
  onUserChange: (user: AdminAccount) => void;
  onNotice: (message: string) => void;
  onUnauthorized: (error: unknown) => boolean;
  onSignedOut: () => void;
  reauthVersion: number;
}) {
  const { t, language } = usePreferences();
  const [sessions, setSessions] = useState<ActiveSession[]>([]);
  const [accounts, setAccounts] = useState<AdminAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);
  const loadVersion = useRef(0);
  const [passwordForm, setPasswordForm] = useState({ current: '', next: '', confirm: '' });
  const [action, setAction] = useState<AccountAction | null>(null);
  const [draft, setDraft] = useState({ username: '', password: '', current: '', role: 'viewer' as AdminAccount['role'] });
  const [actionError, setActionError] = useState('');
  const dialog = useRef<HTMLDialogElement>(null);
  const isAdmin = user.role === 'admin';

  async function load() {
    const version = ++loadVersion.current;
    setLoading(true);
    setError('');
    try {
      const [sessionList, accountList] = await Promise.all([
        listActiveSessions(),
        isAdmin ? listAdminAccounts() : Promise.resolve({ accounts: [] })
      ]);
      if (version === loadVersion.current) {
        setSessions(sessionList.sessions);
        setAccounts(accountList.accounts);
      }
    } catch (ex) {
      if (version === loadVersion.current && !onUnauthorized(ex)) {
        setError(ex instanceof Error ? ex.message : t('accountsLoadFailed'));
      }
    } finally {
      if (version === loadVersion.current) setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    return () => { loadVersion.current++; };
  }, [user.id, user.role, reauthVersion]);

  useEffect(() => {
    if (action) dialog.current?.showModal();
    else dialog.current?.close();
  }, [action]);

  function beginAction(next: AccountAction) {
    setDraft({ username: next.account?.username || '', password: '', current: '', role: 'viewer' });
    setActionError('');
    setAction(next);
  }

  function closeAction() {
    if (busyRef.current) return;
    setAction(null);
    setDraft({ username: '', password: '', current: '', role: 'viewer' });
    setActionError('');
  }

  async function actionFailure(ex: unknown): Promise<string> {
    if (isUnauthorizedError(ex)) {
      // 修改密码接口在当前密码错误和会话过期时都会返回 401，先检查会话避免误弹过期登录框。
      try {
        await getAdminSession();
        return t('accountPasswordIncorrect');
      } catch (sessionError) {
        if (isUnauthorizedError(sessionError)) {
          // 先关闭原生对话框，再显示页面的重新登录弹窗。
          dialog.current?.close();
          setAction(null);
          setDraft({ username: '', password: '', current: '', role: 'viewer' });
          onUnauthorized(sessionError);
          return '';
        }
      }
    }
    return ex instanceof Error ? ex.message : t('accountActionFailed');
  }

  async function submitPassword(event: FormEvent) {
    event.preventDefault();
    if (busyRef.current) return;
    if (passwordForm.next !== passwordForm.confirm) {
      setError(t('passwordMismatch'));
      return;
    }
    busyRef.current = true;
    setBusy(true);
    setError('');
    try {
      await changeAdminPassword(passwordForm.current, passwordForm.next);
      setPasswordForm({ current: '', next: '', confirm: '' });
      onNotice(t('accountPasswordChanged'));
      await load();
    } catch (ex) {
      setError(await actionFailure(ex));
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  }

  async function revoke(session: ActiveSession) {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setError('');
    try {
      const result = await revokeSession(session.id);
      if (result.current) {
        dialog.current?.close();
        onSignedOut();
        return;
      }
      onNotice(t('sessionRevoked'));
      await load();
    } catch (ex) {
      if (!onUnauthorized(ex)) setError(ex instanceof Error ? ex.message : t('accountActionFailed'));
    } finally {
      busyRef.current = false;
      setBusy(false);
      setAction(null);
    }
  }

  async function submitAction(event: FormEvent) {
    event.preventDefault();
    if (!action || busyRef.current) return;
    if (action.kind === 'session' && action.session) {
      await revoke(action.session);
      return;
    }
    busyRef.current = true;
    setBusy(true);
    setActionError('');
    try {
      if (action.kind === 'create') {
        await createAdminAccount({ username: draft.username, password: draft.password, role: draft.role, current_password: draft.current });
      } else if (action.account) {
        const target = action.account;
        const change: AccountUpdate = {};
        if (action.kind === 'rename') change.username = draft.username;
        if (action.kind === 'role') change.role = target.role === 'admin' ? 'viewer' : 'admin';
        if (action.kind === 'disabled') change.disabled = !target.disabled;
        if (action.kind === 'password') change.new_password = draft.password;
        const updated = await updateAdminAccount(target.id, { ...change, current_password: draft.current });
        if (target.id === user.id) {
          if (action.kind === 'role') {
            dialog.current?.close();
            onSignedOut();
            return;
          }
          onUserChange(updated);
        }
      }
      setAction(null);
      setDraft({ username: '', password: '', current: '', role: 'viewer' });
      onNotice(t('accountSaved'));
      await load();
    } catch (ex) {
      setActionError(await actionFailure(ex));
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  }

  return (
    <section className="accounts-panel">
      <div className="account-panel-heading">
        <div>
          <strong>{user.username}</strong>
          <span className="status-pill">{isAdmin ? t('accountRoleAdmin') : t('accountRoleViewer')}</span>
        </div>
        <button type="button" onClick={() => void load()} disabled={loading || busy}>
          <RefreshCw size={16} className={loading ? 'spin' : undefined} aria-hidden="true" />{t('refresh')}
        </button>
      </div>
      {error && <div className="inline-alert inline-alert-danger" role="alert">{error}</div>}
      <div className="account-overview">
        <div className="admin-tool-panel">
          <h2><KeyRound size={18} aria-hidden="true" />{t('changePassword')}</h2>
          <form className="admin-form" onSubmit={submitPassword}>
            <input type="text" autoComplete="username" value={user.username} hidden readOnly />
            <label>
              <span>{t('currentPassword')}</span>
              <input type="password" autoComplete="current-password" value={passwordForm.current} onChange={(event) => setPasswordForm({ ...passwordForm, current: event.target.value })} required />
            </label>
            <label>
              <span>{t('newPassword')}</span>
              <input type="password" autoComplete="new-password" value={passwordForm.next} onChange={(event) => setPasswordForm({ ...passwordForm, next: event.target.value })} minLength={8} maxLength={72} required />
            </label>
            <label>
              <span>{t('confirmNewPassword')}</span>
              <input type="password" autoComplete="new-password" value={passwordForm.confirm} onChange={(event) => setPasswordForm({ ...passwordForm, confirm: event.target.value })} minLength={8} maxLength={72} required />
            </label>
            <p className="field-hint">{t('accountPasswordHint')}</p>
            <button className="primary-button" type="submit" disabled={busy}>
              {busy ? <Loader2 size={16} className="spin" aria-hidden="true" /> : <Save size={16} aria-hidden="true" />}{t('changePassword')}
            </button>
          </form>
        </div>
        <div className="admin-tool-panel">
          <h2><Monitor size={18} aria-hidden="true" />{t('activeSessions')} <span className="status-pill">{sessions.length}</span></h2>
          {loading ? <p role="status">{t('loading')}</p> : sessions.length === 0 ? <p>{t('noActiveSessions')}</p> : (
            <div className="account-records">
              {sessions.map((session) => (
                <article className="account-record" key={session.id}>
                  <div className="account-record-info">
                    <div className="account-record-title"><strong>{session.client_ip || t('unknownClient')}</strong>{session.current && <span className="status-pill active">{t('currentSession')}</span>}</div>
                    <p className="account-user-agent">{session.user_agent || t('unknownBrowser')}</p>
                    <p>{t('sessionLastSeen')}{formatDateTime(session.last_seen_at, language)}</p>
                    <p>{t('sessionExpires')}{formatDateTime(session.expires_at, language)}</p>
                  </div>
                  <button type="button" disabled={busy || loading} onClick={() => session.current ? beginAction({ kind: 'session', title: t('revokeCurrentSession'), session }) : void revoke(session)}>
                    <LogOut size={15} aria-hidden="true" />{t('revokeSession')}
                  </button>
                </article>
              ))}
            </div>
          )}
        </div>
      </div>
      {isAdmin && (
        <div className="admin-tool-panel">
          <div className="account-panel-heading">
            <h2><Users size={18} aria-hidden="true" />{t('managedAccounts')}</h2>
            <button className="primary-button" type="button" disabled={busy || loading} onClick={() => beginAction({ kind: 'create', title: t('createAccount') })}><Plus size={16} aria-hidden="true" />{t('createAccount')}</button>
          </div>
          <p className="field-hint">{t('accountManagementHint')}</p>
          <div className="account-records">
            {accounts.map((account) => (
              <article className="account-record" key={account.id}>
                <div className="account-record-info">
                  <div className="account-record-title"><strong>{account.username}</strong><span className={account.disabled ? 'status-pill' : 'status-pill active'}>{account.disabled ? t('disabledStatus') : t('enabledStatus')}</span>{account.id === user.id && <span className="status-pill">{t('currentAccount')}</span>}</div>
                  <p>{account.role === 'admin' ? t('accountRoleAdmin') : t('accountRoleViewer')}</p>
                  <p>{t('accountLastLogin')}{account.last_login_at ? formatDateTime(account.last_login_at, language) : '—'}</p>
                </div>
                <div className="account-record-actions">
                  <button type="button" disabled={busy || loading} onClick={() => beginAction({ kind: 'rename', title: t('renameAccount'), account })}><Pencil size={14} aria-hidden="true" />{t('renameAccount')}</button>
                  <button type="button" disabled={busy || loading} onClick={() => beginAction({ kind: 'role', title: account.role === 'admin' ? t('setAccountViewer') : t('setAccountAdmin'), account })}><Shield size={14} aria-hidden="true" />{account.role === 'admin' ? t('setAccountViewer') : t('setAccountAdmin')}</button>
                  <button type="button" disabled={busy || loading} onClick={() => beginAction({ kind: 'password', title: t('resetPassword'), account })}><KeyRound size={14} aria-hidden="true" />{t('resetPassword')}</button>
                  <button type="button" disabled={busy || loading || (!account.disabled && account.id === user.id)} onClick={() => beginAction({ kind: 'disabled', title: account.disabled ? t('enableAccount') : t('disableAccount'), account })}>{account.disabled ? t('enableAccount') : t('disableAccount')}</button>
                </div>
              </article>
            ))}
          </div>
        </div>
      )}
      <dialog ref={dialog} className="account-dialog" aria-labelledby="account-dialog-title" onCancel={(event) => { event.preventDefault(); closeAction(); }}>
        {action && (
          <form className="admin-form" onSubmit={submitAction}>
            <div className="account-panel-heading"><h2 id="account-dialog-title">{action.title}</h2><button type="button" disabled={busy} onClick={closeAction} aria-label={t('close')}><X size={18} aria-hidden="true" /></button></div>
            {action.account && <strong>{action.account.username}</strong>}
            <p className="field-hint">{action.kind === 'session' ? t('revokeCurrentSessionHint') : action.kind === 'role' && action.account?.id === user.id ? t('changeOwnRoleHint') : t('accountVerifyHint')}</p>
            {(action.kind === 'create' || action.kind === 'rename') && <label><span>{t('username')}</span><input value={draft.username} onChange={(event) => setDraft({ ...draft, username: event.target.value })} pattern="[a-zA-Z0-9][a-zA-Z0-9_.\-]{2,63}" minLength={3} maxLength={64} required /><small className="field-hint">{t('usernameHint')}</small></label>}
            {action.kind === 'create' && <label><span>{t('accountRole')}</span><select value={draft.role} onChange={(event) => setDraft({ ...draft, role: event.target.value as AdminAccount['role'] })}><option value="viewer">{t('accountRoleViewer')}</option><option value="admin">{t('accountRoleAdmin')}</option></select></label>}
            {(action.kind === 'create' || action.kind === 'password') && <label><span>{t('newPassword')}</span><input type="password" autoComplete="new-password" value={draft.password} onChange={(event) => setDraft({ ...draft, password: event.target.value })} minLength={8} maxLength={72} required /></label>}
            {action.kind !== 'session' && <label><span>{t('actingAccountPassword')}</span><input type="password" autoComplete="current-password" value={draft.current} onChange={(event) => setDraft({ ...draft, current: event.target.value })} required /></label>}
            {actionError && <p className="form-error" role="alert">{actionError}</p>}
            <div className="account-dialog-actions"><button type="button" disabled={busy} onClick={closeAction}>{t('cancel')}</button><button className="primary-button" type="submit" disabled={busy}>{busy ? <Loader2 size={16} className="spin" aria-hidden="true" /> : <Save size={16} aria-hidden="true" />}{busy ? t('saving') : t('confirm')}</button></div>
          </form>
        )}
      </dialog>
    </section>
  );
}
