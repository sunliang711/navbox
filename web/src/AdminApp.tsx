import {
  Check,
  Download,
  FileUp,
  GripVertical,
  KeyRound,
  LogIn,
  LogOut,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Settings,
  Shield,
  Tags,
  Trash2,
  Upload,
  X
} from 'lucide-react';
import { type ChangeEvent, type DragEvent, type FormEvent, type ReactNode, useEffect, useMemo, useState } from 'react';
import {
  batchDeleteSites,
  batchUpdateSiteTags,
  changeAdminPassword,
  createSite,
  createTag,
  deleteSite,
  deleteTag,
  exportConfig,
  fetchWebsiteIcon,
  getAdminSession,
  getRestoreStatus,
  importConfig,
  isUnauthorizedError,
  listSites,
  listTags,
  loginAdmin,
  logoutAdmin,
  restoreAdminPassword,
  setDefaultTag,
  updateSite,
  updateSiteOrder,
  updateTag,
  updateTagOrder,
  uploadIcon
} from './api';
import { PreferenceControls, usePreferences } from './preferences';
import type { ImportReport, OrderItem, Site, SiteSaveReq, Tag, TagSaveReq } from './types';

type AdminTab = 'sites' | 'tags' | 'io' | 'password';

const emptySiteForm: SiteSaveReq = {
  title: '',
  description: '',
  default_url: '',
  lan_url: '',
  open_method: 'new_window',
  icon_type: 'text',
  icon_value: '',
  background_color: '',
  only_name: false,
  is_favorite: false,
  sort_order: 0,
  tag_ids: []
};

const emptyTagForm: TagSaveReq = {
  name: '',
  icon: '',
  color: '#2f7d6d',
  sort_order: 0,
  is_enabled: true
};

export function AdminApp() {
  const { t } = usePreferences();
  const [checking, setChecking] = useState(true);
  const [authenticated, setAuthenticated] = useState(false);
  const [restoreMode, setRestoreMode] = useState(false);
  const [password, setPassword] = useState('');
  const [loginError, setLoginError] = useState('');
  const [tab, setTab] = useState<AdminTab>('sites');
  const [sites, setSites] = useState<Site[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState('');
  const [selectedSiteIds, setSelectedSiteIds] = useState<string[]>([]);
  const [batchTagIds, setBatchTagIds] = useState<string[]>([]);
  const [exportTagIds, setExportTagIds] = useState<string[]>([]);
  const [siteDraft, setSiteDraft] = useState<SiteSaveReq>(emptySiteForm);
  const [editingSiteId, setEditingSiteId] = useState('');
  const [sitePanelOpen, setSitePanelOpen] = useState(false);
  const [iconFetching, setIconFetching] = useState(false);
  const [tagDraft, setTagDraft] = useState<TagSaveReq>(emptyTagForm);
  const [editingTagId, setEditingTagId] = useState('');
  const [tagPanelOpen, setTagPanelOpen] = useState(false);
  const [siteOrder, setSiteOrder] = useState<Record<string, number>>({});
  const [tagOrder, setTagOrder] = useState<Record<string, number>>({});
  const [draggingSiteId, setDraggingSiteId] = useState('');
  const [draggingTagId, setDraggingTagId] = useState('');
  const [importReport, setImportReport] = useState<ImportReport | null>(null);
  const [passwordForm, setPasswordForm] = useState({ current: '', next: '' });
  const [restoreForm, setRestoreForm] = useState({ token: '', next: '', confirm: '' });
  const [restoreNotice, setRestoreNotice] = useState('');

  useEffect(() => {
    let cancelled = false;
    async function checkSession() {
      try {
        const restoreStatus = await getRestoreStatus();
        if (!cancelled && restoreStatus.enabled && restoreStatus.mode === 'admin-password') {
          setRestoreMode(true);
          setAuthenticated(false);
          setChecking(false);
          return;
        }
      } catch {
        if (!cancelled) {
          setRestoreMode(false);
        }
      }

      try {
        await getAdminSession();
        if (!cancelled) {
          setAuthenticated(true);
        }
      } catch {
        if (!cancelled) {
          setAuthenticated(false);
        }
      } finally {
        if (!cancelled) {
          setChecking(false);
        }
      }
    }
    checkSession();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!authenticated) {
      return;
    }
    refreshData();
  }, [authenticated]);

  useEffect(() => {
    if (!notice) {
      return;
    }
    const timer = window.setTimeout(() => setNotice(''), 2200);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const selectedSiteSet = useMemo(() => new Set(selectedSiteIds), [selectedSiteIds]);
  const batchTagSet = useMemo(() => new Set(batchTagIds), [batchTagIds]);
  const exportTagSet = useMemo(() => new Set(exportTagIds), [exportTagIds]);

  function handleAdminError(error: unknown): boolean {
    if (!isUnauthorizedError(error)) {
      return false;
    }
    setAuthenticated(false);
    setNotice('');
    setLoginError(t('sessionExpired'));
    setSitePanelOpen(false);
    setTagPanelOpen(false);
    setSelectedSiteIds([]);
    setBatchTagIds([]);
    setExportTagIds([]);
    return true;
  }

  async function refreshData() {
    setLoading(true);
    try {
      const [siteList, tagList] = await Promise.all([listSites({}), listTags()]);
      setSites(siteList);
      setTags(tagList);
      setSiteOrder(Object.fromEntries(siteList.map((site) => [site.id, site.sort_order])));
      setTagOrder(Object.fromEntries(tagList.map((tag) => [tag.id, tag.sort_order])));
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    } finally {
      setLoading(false);
    }
  }

  async function submitLogin(event: FormEvent) {
    event.preventDefault();
    setLoginError('');
    try {
      await loginAdmin(password);
      setAuthenticated(true);
      setPassword('');
    } catch {
      setLoginError(t('loginFailed'));
    }
  }

  async function handleLogout() {
    try {
      await logoutAdmin();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
    setAuthenticated(false);
  }

  function openNewSite() {
    setEditingSiteId('');
    setSiteDraft(emptySiteForm);
    setSitePanelOpen(true);
  }

  function openEditSite(site: Site) {
    setEditingSiteId(site.id);
    setSiteDraft({
      title: site.title,
      description: site.description,
      default_url: site.default_url,
      lan_url: site.lan_url,
      open_method: site.open_method,
      icon_type: site.icon_type,
      icon_value: site.icon_value,
      background_color: site.background_color,
      only_name: site.only_name,
      is_favorite: site.is_favorite,
      sort_order: site.sort_order,
      tag_ids: site.tags.map((tag) => tag.id)
    });
    setSitePanelOpen(true);
  }

  async function submitSite(event: FormEvent) {
    event.preventDefault();
    try {
      if (editingSiteId) {
        await updateSite(editingSiteId, siteDraft);
        setNotice(t('siteUpdated'));
      } else {
        await createSite(siteDraft);
        setNotice(t('siteCreated'));
      }
      setSitePanelOpen(false);
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function removeSite(id: string) {
    if (!window.confirm(t('confirmDeleteSite'))) {
      return;
    }
    try {
      await deleteSite(id);
      setSelectedSiteIds((current) => current.filter((item) => item !== id));
      setNotice(t('siteDeleted'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function handleIconUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    try {
      const icon = await uploadIcon(file);
      event.target.value = '';
      setSiteDraft((current) => ({
        ...current,
        icon_type: 'image',
        icon_value: icon.url
      }));
      setNotice(t('iconUploaded'));
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function handleFetchIcon() {
    const targetURL = siteDraft.default_url.trim();
    if (!targetURL) {
      setNotice(t('fillDefaultURL'));
      return;
    }

    setIconFetching(true);
    try {
      const icon = await fetchWebsiteIcon(targetURL);
      setSiteDraft((current) => ({
        ...current,
        icon_type: 'image',
        icon_value: icon.url
      }));
      setNotice(t('iconFetched'));
    } catch (error) {
      if (!handleAdminError(error)) {
        setNotice(t('iconFetchFailed'));
      }
    } finally {
      setIconFetching(false);
    }
  }

  function openNewTag() {
    setEditingTagId('');
    setTagDraft(emptyTagForm);
    setTagPanelOpen(true);
  }

  function openEditTag(tag: Tag) {
    setEditingTagId(tag.id);
    setTagDraft({
      name: tag.name,
      icon: tag.icon,
      color: tag.color || '#2f7d6d',
      sort_order: tag.sort_order,
      is_enabled: tag.is_enabled
    });
    setTagPanelOpen(true);
  }

  async function submitTag(event: FormEvent) {
    event.preventDefault();
    try {
      if (editingTagId) {
        await updateTag(editingTagId, tagDraft);
        setNotice(t('tagUpdated'));
      } else {
        await createTag(tagDraft);
        setNotice(t('tagCreated'));
      }
      setTagPanelOpen(false);
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function removeTag(id: string) {
    if (!window.confirm(t('confirmDeleteTag'))) {
      return;
    }
    try {
      await deleteTag(id);
      setNotice(t('tagDeleted'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function makeDefaultTag(id: string) {
    try {
      await setDefaultTag(id);
      setNotice(t('defaultTagUpdated'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function saveSiteOrder() {
    try {
      await updateSiteOrder(toOrderItems(siteOrder));
      setNotice(t('siteOrderSaved'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function saveTagOrder() {
    try {
      await updateTagOrder(toOrderItems(tagOrder));
      setNotice(t('tagOrderSaved'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  function startSiteDrag(event: DragEvent<HTMLElement>, siteId: string) {
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', siteId);
    setDraggingSiteId(siteId);
  }

  function startTagDrag(event: DragEvent<HTMLElement>, tagId: string) {
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', tagId);
    setDraggingTagId(tagId);
  }

  function allowSortDrop(event: DragEvent<HTMLTableRowElement>) {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }

  async function dropSite(event: DragEvent<HTMLTableRowElement>, targetId: string) {
    event.preventDefault();
    const sourceId = draggingSiteId || event.dataTransfer.getData('text/plain');
    setDraggingSiteId('');
    if (!sourceId || sourceId === targetId) {
      return;
    }

    const nextSites = moveItemById(sites, sourceId, targetId);
    const nextOrder = buildSequentialOrder(nextSites);
    setSites(nextSites);
    setSiteOrder(nextOrder);
    try {
      await updateSiteOrder(toOrderItems(nextOrder));
      setNotice(t('siteOrderUpdated'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        setNotice(t('siteOrderUpdateFailed'));
        await refreshData();
      }
    }
  }

  async function dropTag(event: DragEvent<HTMLTableRowElement>, targetId: string) {
    event.preventDefault();
    const sourceId = draggingTagId || event.dataTransfer.getData('text/plain');
    setDraggingTagId('');
    if (!sourceId || sourceId === targetId) {
      return;
    }

    const nextTags = moveItemById(tags, sourceId, targetId);
    const nextOrder = buildSequentialOrder(nextTags);
    setTags(nextTags);
    setTagOrder(nextOrder);
    try {
      await updateTagOrder(toOrderItems(nextOrder));
      setNotice(t('tagOrderUpdated'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        setNotice(t('tagOrderUpdateFailed'));
        await refreshData();
      }
    }
  }

  async function handleBatchDelete() {
    if (selectedSiteIds.length === 0 || !window.confirm(t('confirmBatchDelete', { count: selectedSiteIds.length }))) {
      return;
    }
    try {
      await batchDeleteSites(selectedSiteIds);
      setSelectedSiteIds([]);
      setNotice(t('batchDeleteDone'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function handleBatchTags(action: 'add' | 'remove') {
    if (selectedSiteIds.length === 0 || batchTagIds.length === 0) {
      setNotice(t('chooseSitesAndTags'));
      return;
    }
    try {
      await batchUpdateSiteTags(selectedSiteIds, batchTagIds, action);
      setNotice(action === 'add' ? t('batchAddTagDone') : t('batchRemoveTagDone'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function downloadExport(mode: 'all' | 'sites' | 'tags') {
    const req =
      mode === 'sites'
        ? { site_ids: selectedSiteIds }
        : mode === 'tags'
          ? { tag_ids: exportTagIds }
          : {};
    if (mode === 'sites' && selectedSiteIds.length === 0) {
      setNotice(t('chooseSites'));
      return;
    }
    if (mode === 'tags' && exportTagIds.length === 0) {
      setNotice(t('chooseTags'));
      return;
    }

    try {
      const file = await exportConfig(req);
      const url = URL.createObjectURL(file.blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = file.fileName;
      link.click();
      URL.revokeObjectURL(url);
      setNotice(t('exportStarted'));
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function handleImport(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    try {
      const report = await importConfig(file);
      setImportReport(report);
      setNotice(t('importDone'));
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function submitPassword(event: FormEvent) {
    event.preventDefault();
    try {
      await changeAdminPassword(passwordForm.current, passwordForm.next);
      setPasswordForm({ current: '', next: '' });
      setNotice(t('passwordChanged'));
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function submitRestorePassword(event: FormEvent) {
    event.preventDefault();
    setLoginError('');
    setRestoreNotice('');
    if (restoreForm.next.length < 8) {
      setLoginError(t('passwordMin'));
      return;
    }
    if (restoreForm.next !== restoreForm.confirm) {
      setLoginError(t('passwordMismatch'));
      return;
    }

    try {
      await restoreAdminPassword(restoreForm.token, restoreForm.next);
      setRestoreForm({ token: '', next: '', confirm: '' });
      setRestoreNotice(t('restoreSuccess'));
    } catch {
      setLoginError(t('restoreTokenInvalid'));
    }
  }

  function toggleSiteSelection(id: string) {
    setSelectedSiteIds((current) => (current.includes(id) ? current.filter((item) => item !== id) : [...current, id]));
  }

  function toggleString(current: string[], id: string, setter: (value: string[]) => void) {
    setter(current.includes(id) ? current.filter((item) => item !== id) : [...current, id]);
  }

  if (checking) {
    return <AdminState text={t('checkingSession')} />;
  }

  if (restoreMode) {
    return (
      <main className="admin-login">
        <form className="login-panel restore-panel" onSubmit={submitRestorePassword}>
          <div className="auth-preferences">
            <PreferenceControls />
          </div>
          <div className="login-mark">
            <KeyRound size={26} aria-hidden="true" />
          </div>
          <h1>{t('restorePasswordTitle')}</h1>
          <p className="form-note">{t('restoreModeNote')}</p>
          <label>
            <span>{t('restoreToken')}</span>
            <input
              type="password"
              value={restoreForm.token}
              onChange={(event) => setRestoreForm({ ...restoreForm, token: event.target.value })}
              autoFocus
              required
            />
          </label>
          <label>
            <span>{t('newPassword')}</span>
            <input
              type="password"
              value={restoreForm.next}
              onChange={(event) => setRestoreForm({ ...restoreForm, next: event.target.value })}
              required
            />
          </label>
          <label>
            <span>{t('confirmNewPassword')}</span>
            <input
              type="password"
              value={restoreForm.confirm}
              onChange={(event) => setRestoreForm({ ...restoreForm, confirm: event.target.value })}
              required
            />
          </label>
          {loginError && <p className="form-error">{loginError}</p>}
          {restoreNotice && <p className="form-success">{restoreNotice}</p>}
          <button className="primary-button" type="submit">
            <Save size={17} aria-hidden="true" />
            {t('resetPassword')}
          </button>
          <p className="form-note">{t('restoreDoneNote')}</p>
        </form>
      </main>
    );
  }

  if (!authenticated) {
    return (
      <main className="admin-login">
        <form className="login-panel" onSubmit={submitLogin}>
          <div className="auth-preferences">
            <PreferenceControls />
          </div>
          <div className="login-mark">
            <Shield size={26} aria-hidden="true" />
          </div>
          <h1>{t('adminLogin')}</h1>
          <label>
            <span>{t('password')}</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoFocus
              required
            />
          </label>
          {loginError && <p className="form-error">{loginError}</p>}
          <button className="primary-button" type="submit">
            <LogIn size={17} aria-hidden="true" />
            {t('login')}
          </button>
        </form>
      </main>
    );
  }

  return (
    <main className="admin-shell">
      <header className="admin-header">
        <div>
          <h1>{t('navboxAdmin')}</h1>
          <p>{t('adminStats', { sites: sites.length, tags: tags.length })}</p>
        </div>
        <div className="admin-header-actions">
          <PreferenceControls />
          <button type="button" onClick={() => (window.location.href = '/')}>
            {t('visitorHome')}
          </button>
          <button type="button" onClick={handleLogout}>
            <LogOut size={17} aria-hidden="true" />
            {t('logout')}
          </button>
        </div>
      </header>

      <nav className="admin-tabs" aria-label={t('adminModules')}>
        <AdminTabButton active={tab === 'sites'} onClick={() => setTab('sites')} icon={Settings} label={t('sitesTab')} />
        <AdminTabButton active={tab === 'tags'} onClick={() => setTab('tags')} icon={Tags} label={t('tagsTab')} />
        <AdminTabButton active={tab === 'io'} onClick={() => setTab('io')} icon={FileUp} label={t('importExportTab')} />
        <AdminTabButton active={tab === 'password'} onClick={() => setTab('password')} icon={KeyRound} label={t('passwordTab')} />
      </nav>

      {notice && <div className="toast">{notice}</div>}
      {loading && <AdminState text={t('refreshing')} />}

      {!loading && tab === 'sites' && (
        <section className="admin-section">
          <div className="admin-actions">
            <button className="primary-button" type="button" onClick={openNewSite}>
              <Plus size={17} aria-hidden="true" />
              {t('newSite')}
            </button>
            <button type="button" onClick={saveSiteOrder}>
              <Save size={17} aria-hidden="true" />
              {t('saveOrder')}
            </button>
            <button type="button" onClick={handleBatchDelete}>
              <Trash2 size={17} aria-hidden="true" />
              {t('batchDelete')}
            </button>
          </div>

          <BatchTagBar
            tags={tags}
            selected={batchTagSet}
            onToggle={(id) => toggleString(batchTagIds, id, setBatchTagIds)}
            onAdd={() => handleBatchTags('add')}
            onRemove={() => handleBatchTags('remove')}
          />

          <div className="admin-table-wrap">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>{t('drag')}</th>
                  <th>{t('select')}</th>
                  <th>{t('site')}</th>
                  <th>Tag</th>
                  <th>{t('openMethod')}</th>
                  <th>{t('favoriteColumn')}</th>
                  <th>{t('sort')}</th>
                  <th>{t('actions')}</th>
                </tr>
              </thead>
              <tbody>
                {sites.map((site) => (
                  <tr
                    className={draggingSiteId === site.id ? 'dragging-row' : undefined}
                    key={site.id}
                    onDragOver={allowSortDrop}
                    onDrop={(event) => dropSite(event, site.id)}
                  >
                    <td>
                      <span
                        className="drag-handle"
                        draggable
                        onDragStart={(event) => startSiteDrag(event, site.id)}
                        onDragEnd={() => setDraggingSiteId('')}
                        role="button"
                        tabIndex={0}
                        title={t('dragSort')}
                        aria-label={`${t('dragSort')} ${site.title}`}
                      >
                        <GripVertical size={16} aria-hidden="true" />
                      </span>
                    </td>
                    <td>
                      <input
                        type="checkbox"
                        checked={selectedSiteSet.has(site.id)}
                        onChange={() => toggleSiteSelection(site.id)}
                        aria-label={t('selectSite', { title: site.title })}
                      />
                    </td>
                    <td>
                      <strong>{site.title}</strong>
                      <span>{site.default_url}</span>
                    </td>
                    <td>{site.tags.map((tag) => tag.name).join(', ') || t('uncategorized')}</td>
                    <td>{site.open_method === 'current_window' ? t('currentTab') : t('newTab')}</td>
                    <td>{site.is_favorite ? t('yes') : t('no')}</td>
                    <td>
                      <input
                        className="order-input"
                        type="number"
                        value={siteOrder[site.id] ?? site.sort_order}
                        onChange={(event) => setSiteOrder({ ...siteOrder, [site.id]: Number(event.target.value) })}
                      />
                    </td>
                    <td>
                      <div className="row-actions">
                        <button type="button" onClick={() => openEditSite(site)} title={t('editSite')}>
                          <Pencil size={16} aria-hidden="true" />
                        </button>
                        <button type="button" onClick={() => removeSite(site.id)} title={t('deleteSite')}>
                          <Trash2 size={16} aria-hidden="true" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {!loading && tab === 'tags' && (
        <section className="admin-section">
          <div className="admin-actions">
            <button className="primary-button" type="button" onClick={openNewTag}>
              <Plus size={17} aria-hidden="true" />
              {t('newTag')}
            </button>
            <button type="button" onClick={saveTagOrder}>
              <Save size={17} aria-hidden="true" />
              {t('saveOrder')}
            </button>
          </div>
          <div className="admin-table-wrap">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>{t('drag')}</th>
                  <th>Tag</th>
                  <th>{t('status')}</th>
                  <th>{t('siteCount')}</th>
                  <th>{t('sort')}</th>
                  <th>{t('actions')}</th>
                </tr>
              </thead>
              <tbody>
                {tags.map((tag) => (
                  <tr
                    className={draggingTagId === tag.id ? 'dragging-row' : undefined}
                    key={tag.id}
                    onDragOver={allowSortDrop}
                    onDrop={(event) => dropTag(event, tag.id)}
                  >
                    <td>
                      <span
                        className="drag-handle"
                        draggable
                        onDragStart={(event) => startTagDrag(event, tag.id)}
                        onDragEnd={() => setDraggingTagId('')}
                        role="button"
                        tabIndex={0}
                        title={t('dragSort')}
                        aria-label={`${t('dragSort')} ${tag.name}`}
                      >
                        <GripVertical size={16} aria-hidden="true" />
                      </span>
                    </td>
                    <td>
                      <strong>
                        <span className="color-swatch" style={{ background: tag.color || '#2f7d6d' }} />
                        {tag.name}
                      </strong>
                      <span>{tag.icon || t('noIcon')}</span>
                    </td>
                    <td>{tag.is_default ? t('defaultStatus') : tag.is_enabled ? t('enabledStatus') : t('disabledStatus')}</td>
                    <td>{tag.site_count}</td>
                    <td>
                      <input
                        className="order-input"
                        type="number"
                        value={tagOrder[tag.id] ?? tag.sort_order}
                        onChange={(event) => setTagOrder({ ...tagOrder, [tag.id]: Number(event.target.value) })}
                      />
                    </td>
                    <td>
                      <div className="row-actions">
                        {!tag.is_default && (
                          <button type="button" onClick={() => makeDefaultTag(tag.id)} title={t('setDefault')}>
                            <Check size={16} aria-hidden="true" />
                          </button>
                        )}
                        <button type="button" onClick={() => openEditTag(tag)} title={t('editTag')}>
                          <Pencil size={16} aria-hidden="true" />
                        </button>
                        <button type="button" onClick={() => removeTag(tag.id)} title={t('deleteTag')}>
                          <Trash2 size={16} aria-hidden="true" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {!loading && tab === 'io' && (
        <section className="admin-section admin-grid-two">
          <div className="admin-tool-panel">
            <h2>{t('export')}</h2>
            <div className="stack">
              <button className="primary-button" type="button" onClick={() => downloadExport('all')}>
                <Download size={17} aria-hidden="true" />
                {t('exportAll')}
              </button>
              <button type="button" onClick={() => downloadExport('sites')}>
                {t('exportSelectedSites')}
              </button>
              <div className="checkbox-list">
                {tags.map((tag) => (
                  <label key={tag.id}>
                    <input
                      type="checkbox"
                      checked={exportTagSet.has(tag.id)}
                      onChange={() => toggleString(exportTagIds, tag.id, setExportTagIds)}
                    />
                    <span>{tag.name}</span>
                  </label>
                ))}
              </div>
              <button type="button" onClick={() => downloadExport('tags')}>
                {t('exportSelectedTags')}
              </button>
            </div>
          </div>

          <div className="admin-tool-panel">
            <h2>{t('import')}</h2>
            <label className="file-button">
              <Upload size={17} aria-hidden="true" />
              {t('chooseZipImport')}
              <input type="file" accept=".zip,application/zip" onChange={handleImport} />
            </label>
            {importReport && (
              <div className="report-box">
                <strong>{t('importResult')}</strong>
                <span>{t('importedSummary', {
                  sites: importReport.imported.sites,
                  tags: importReport.imported.tags,
                  icons: importReport.imported.icons
                })}</span>
                <span>{t('skippedSummary', {
                  sites: importReport.skipped.sites,
                  tags: importReport.skipped.tags,
                  icons: importReport.skipped.icons
                })}</span>
                {importReport.conflicts.slice(0, 5).map((conflict, index) => (
                  <span key={`${conflict.type}-${conflict.id}-${index}`}>{conflict.type}: {conflict.reason}</span>
                ))}
              </div>
            )}
          </div>
        </section>
      )}

      {!loading && tab === 'password' && (
        <section className="admin-section narrow-section">
          <form className="admin-form" onSubmit={submitPassword}>
            <label>
              <span>{t('currentPassword')}</span>
              <input
                type="password"
                value={passwordForm.current}
                onChange={(event) => setPasswordForm({ ...passwordForm, current: event.target.value })}
                required
              />
            </label>
            <label>
              <span>{t('newPassword')}</span>
              <input
                type="password"
                value={passwordForm.next}
                onChange={(event) => setPasswordForm({ ...passwordForm, next: event.target.value })}
                minLength={8}
                required
              />
            </label>
            <button className="primary-button" type="submit">
              <Save size={17} aria-hidden="true" />
              {t('changePassword')}
            </button>
          </form>
        </section>
      )}

      {sitePanelOpen && (
        <SidePanel title={editingSiteId ? t('editSiteTitle') : t('newSiteTitle')} onClose={() => setSitePanelOpen(false)}>
          <SiteForm
            draft={siteDraft}
            tags={tags}
            onChange={setSiteDraft}
            onSubmit={submitSite}
            onIconUpload={handleIconUpload}
            onFetchIcon={handleFetchIcon}
            iconFetching={iconFetching}
          />
        </SidePanel>
      )}

      {tagPanelOpen && (
        <SidePanel title={editingTagId ? t('editTagTitle') : t('newTagTitle')} onClose={() => setTagPanelOpen(false)}>
          <TagForm draft={tagDraft} onChange={setTagDraft} onSubmit={submitTag} />
        </SidePanel>
      )}
    </main>
  );
}

function AdminTabButton({
  active,
  icon: Icon,
  label,
  onClick
}: {
  active: boolean;
  icon: typeof Settings;
  label: string;
  onClick: () => void;
}) {
  return (
    <button className={active ? 'admin-tab active' : 'admin-tab'} type="button" onClick={onClick}>
      <Icon size={17} aria-hidden="true" />
      <span>{label}</span>
    </button>
  );
}

function BatchTagBar({
  tags,
  selected,
  onToggle,
  onAdd,
  onRemove
}: {
  tags: Tag[];
  selected: Set<string>;
  onToggle: (id: string) => void;
  onAdd: () => void;
  onRemove: () => void;
}) {
  const { t } = usePreferences();

  return (
    <div className="batch-bar">
      <div className="checkbox-list compact">
        {tags.map((tag) => (
          <label key={tag.id}>
            <input type="checkbox" checked={selected.has(tag.id)} onChange={() => onToggle(tag.id)} />
            <span>{tag.name}</span>
          </label>
        ))}
      </div>
      <button type="button" onClick={onAdd}>{t('batchAddTag')}</button>
      <button type="button" onClick={onRemove}>{t('batchRemoveTag')}</button>
    </div>
  );
}

function SidePanel({ title, children, onClose }: { title: string; children: ReactNode; onClose: () => void }) {
  const { t } = usePreferences();

  return (
    <div className="panel-backdrop">
      <aside className="side-panel">
        <header>
          <h2>{title}</h2>
          <button type="button" onClick={onClose} title={t('close')}>
            <X size={18} aria-hidden="true" />
          </button>
        </header>
        {children}
      </aside>
    </div>
  );
}

function SiteForm({
  draft,
  tags,
  onChange,
  onSubmit,
  onIconUpload,
  onFetchIcon,
  iconFetching
}: {
  draft: SiteSaveReq;
  tags: Tag[];
  onChange: (draft: SiteSaveReq) => void;
  onSubmit: (event: FormEvent) => void;
  onIconUpload: (event: ChangeEvent<HTMLInputElement>) => void;
  onFetchIcon: () => void;
  iconFetching: boolean;
}) {
  const { t } = usePreferences();

  function patch(next: Partial<SiteSaveReq>) {
    onChange({ ...draft, ...next });
  }

  function toggleTag(id: string) {
    patch({
      tag_ids: draft.tag_ids.includes(id) ? draft.tag_ids.filter((item) => item !== id) : [...draft.tag_ids, id]
    });
  }

  return (
    <form className="admin-form" onSubmit={onSubmit}>
      <label>
        <span>{t('title')}</span>
        <input value={draft.title} onChange={(event) => patch({ title: event.target.value })} required />
      </label>
      <label>
        <span>{t('defaultURL')}</span>
        <input value={draft.default_url} onChange={(event) => patch({ default_url: event.target.value })} required />
      </label>
      <label>
        <span>{t('lanURL')}</span>
        <input value={draft.lan_url} onChange={(event) => patch({ lan_url: event.target.value })} />
      </label>
      <label>
        <span>{t('description')}</span>
        <textarea value={draft.description} onChange={(event) => patch({ description: event.target.value })} rows={3} />
      </label>
      <div className="form-grid">
        <label>
          <span>{t('openMethod')}</span>
          <select value={draft.open_method} onChange={(event) => patch({ open_method: event.target.value })}>
            <option value="new_window">{t('newWindow')}</option>
            <option value="current_window">{t('currentWindow')}</option>
          </select>
        </label>
        <label>
          <span>{t('sort')}</span>
          <input type="number" value={draft.sort_order} onChange={(event) => patch({ sort_order: Number(event.target.value) })} />
        </label>
      </div>
      <div className="form-grid">
        <label>
          <span>{t('iconType')}</span>
          <select value={draft.icon_type} onChange={(event) => patch({ icon_type: event.target.value })}>
            <option value="text">{t('textIcon')}</option>
            <option value="image">{t('imageIcon')}</option>
            <option value="online">{t('onlineIcon')}</option>
          </select>
        </label>
        <label>
          <span>{t('backgroundColor')}</span>
          <input value={draft.background_color} onChange={(event) => patch({ background_color: event.target.value })} />
        </label>
      </div>
      <label>
        <span>{t('iconValue')}</span>
        <input value={draft.icon_value} onChange={(event) => patch({ icon_value: event.target.value })} />
      </label>
      <div className="icon-actions">
        <button type="button" onClick={onFetchIcon} disabled={iconFetching || !draft.default_url.trim()}>
          <RefreshCw className={iconFetching ? 'spin' : ''} size={17} aria-hidden="true" />
          {iconFetching ? t('fetching') : t('fetchFromWebsite')}
        </button>
        <label className="file-button inline-file">
          <Upload size={17} aria-hidden="true" />
          {t('uploadIcon')}
          <input type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/x-icon" onChange={onIconUpload} />
        </label>
      </div>
      <div className="checkbox-row">
        <label>
          <input type="checkbox" checked={draft.only_name} onChange={(event) => patch({ only_name: event.target.checked })} />
          <span>{t('onlyName')}</span>
        </label>
        <label>
          <input type="checkbox" checked={draft.is_favorite} onChange={(event) => patch({ is_favorite: event.target.checked })} />
          <span>{t('favorite')}</span>
        </label>
      </div>
      <div className="checkbox-list">
        {tags.map((tag) => (
          <label key={tag.id}>
            <input type="checkbox" checked={draft.tag_ids.includes(tag.id)} onChange={() => toggleTag(tag.id)} />
            <span>{tag.name}</span>
          </label>
        ))}
      </div>
      <button className="primary-button" type="submit">
        <Save size={17} aria-hidden="true" />
        {t('saveSite')}
      </button>
    </form>
  );
}

function TagForm({
  draft,
  onChange,
  onSubmit
}: {
  draft: TagSaveReq;
  onChange: (draft: TagSaveReq) => void;
  onSubmit: (event: FormEvent) => void;
}) {
  const { t } = usePreferences();

  function patch(next: Partial<TagSaveReq>) {
    onChange({ ...draft, ...next });
  }

  return (
    <form className="admin-form" onSubmit={onSubmit}>
      <label>
        <span>{t('name')}</span>
        <input value={draft.name} onChange={(event) => patch({ name: event.target.value })} required />
      </label>
      <label>
        <span>{t('icon')}</span>
        <input value={draft.icon} onChange={(event) => patch({ icon: event.target.value })} />
      </label>
      <div className="form-grid">
        <label>
          <span>{t('color')}</span>
          <input type="color" value={draft.color || '#2f7d6d'} onChange={(event) => patch({ color: event.target.value })} />
        </label>
        <label>
          <span>{t('sort')}</span>
          <input type="number" value={draft.sort_order} onChange={(event) => patch({ sort_order: Number(event.target.value) })} />
        </label>
      </div>
      <label className="checkbox-line">
        <input type="checkbox" checked={draft.is_enabled} onChange={(event) => patch({ is_enabled: event.target.checked })} />
        <span>{t('enabled')}</span>
      </label>
      <button className="primary-button" type="submit">
        <Save size={17} aria-hidden="true" />
        {t('saveTag')}
      </button>
    </form>
  );
}

function AdminState({ text }: { text: string }) {
  return (
    <div className="admin-state">
      <span>{text}</span>
    </div>
  );
}

function toOrderItems(values: Record<string, number>): OrderItem[] {
  return Object.entries(values).map(([id, sortOrder]) => ({ id, sort_order: Number(sortOrder) }));
}

function buildSequentialOrder<T extends { id: string }>(items: T[]): Record<string, number> {
  return Object.fromEntries(items.map((item, index) => [item.id, (index + 1) * 10]));
}

function moveItemById<T extends { id: string }>(items: T[], sourceId: string, targetId: string): T[] {
  const sourceIndex = items.findIndex((item) => item.id === sourceId);
  const targetIndex = items.findIndex((item) => item.id === targetId);
  if (sourceIndex < 0 || targetIndex < 0) {
    return items;
  }

  const next = [...items];
  const [source] = next.splice(sourceIndex, 1);
  const insertIndex = sourceIndex < targetIndex ? targetIndex - 1 : targetIndex;
  next.splice(insertIndex, 0, source);
  return next;
}
