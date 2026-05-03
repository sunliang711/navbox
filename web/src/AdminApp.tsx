import {
  ArrowDown,
  ArrowUp,
  Check,
  Download,
  FileUp,
  GripVertical,
  KeyRound,
  LogIn,
  LogOut,
  MoreHorizontal,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Search,
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
import { VersionBadge } from './version';
import type { ImportReport, OrderItem, Site, SiteSaveReq, Tag, TagSaveReq } from './types';

type AdminTab = 'sites' | 'tags' | 'io' | 'password';
type ExportMode = 'all' | 'sites' | 'tags';
type FavoriteFilter = 'all' | 'favorites';

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
  const [adminSearch, setAdminSearch] = useState('');
  const [siteTagFilterId, setSiteTagFilterId] = useState('all');
  const [siteFavoriteFilter, setSiteFavoriteFilter] = useState<FavoriteFilter>('all');
  const [sites, setSites] = useState<Site[]>([]);
  const [tags, setTags] = useState<Tag[]>([]);
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState('');
  const [selectedSiteIds, setSelectedSiteIds] = useState<string[]>([]);
  const [batchTagIds, setBatchTagIds] = useState<string[]>([]);
  const [exportMode, setExportMode] = useState<ExportMode>('all');
  const [exportSiteIds, setExportSiteIds] = useState<string[]>([]);
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
  const [siteSelectionMode, setSiteSelectionMode] = useState(false);
  const [mobileSiteSortMode, setMobileSiteSortMode] = useState(false);
  const [mobileTagSortMode, setMobileTagSortMode] = useState(false);
  const [mobileBatchTagsOpen, setMobileBatchTagsOpen] = useState(false);
  const [mobileAdminMenuOpen, setMobileAdminMenuOpen] = useState(false);
  const [mobileSiteActionsOpen, setMobileSiteActionsOpen] = useState(false);
  const [mobileTagActionsOpen, setMobileTagActionsOpen] = useState(false);
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

  useEffect(() => {
    setMobileBatchTagsOpen(false);
    setMobileAdminMenuOpen(false);
    setMobileSiteActionsOpen(false);
    setMobileTagActionsOpen(false);
    setMobileSiteSortMode(false);
    setMobileTagSortMode(false);
  }, [tab]);

  const selectedSiteSet = useMemo(() => new Set(selectedSiteIds), [selectedSiteIds]);
  const batchTagSet = useMemo(() => new Set(batchTagIds), [batchTagIds]);
  const exportSiteSet = useMemo(() => new Set(exportSiteIds), [exportSiteIds]);
  const exportTagSet = useMemo(() => new Set(exportTagIds), [exportTagIds]);
  const normalizedAdminSearch = adminSearch.trim().toLocaleLowerCase();
  const filteredSites = useMemo(() => {
    return sites.filter((site) => {
      if (siteTagFilterId !== 'all' && !site.tags.some((tag) => tag.id === siteTagFilterId)) {
        return false;
      }
      if (siteFavoriteFilter === 'favorites' && !site.is_favorite) {
        return false;
      }
      if (!normalizedAdminSearch) {
        return true;
      }
      return [
        site.title,
        site.description,
        site.default_url,
        site.lan_url,
        site.open_method,
        ...site.tags.map((tag) => tag.name)
      ].some((value) => value.toLocaleLowerCase().includes(normalizedAdminSearch));
    });
  }, [sites, siteTagFilterId, siteFavoriteFilter, normalizedAdminSearch]);
  const filteredTags = useMemo(() => {
    if (!normalizedAdminSearch) {
      return tags;
    }
    return tags.filter((tag) =>
      [tag.name, tag.icon, tag.is_default ? t('defaultStatus') : tag.is_enabled ? t('enabledStatus') : t('disabledStatus')]
        .some((value) => value.toLocaleLowerCase().includes(normalizedAdminSearch))
    );
  }, [tags, normalizedAdminSearch, t]);

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
    setSiteSelectionMode(false);
    setMobileBatchTagsOpen(false);
    setMobileAdminMenuOpen(false);
    setMobileSiteActionsOpen(false);
    setMobileTagActionsOpen(false);
    setExportSiteIds([]);
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
      setBatchTagIds([]);
      setSiteSelectionMode(false);
      setMobileBatchTagsOpen(false);
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
      setMobileBatchTagsOpen(false);
      await refreshData();
    } catch (error) {
      if (!handleAdminError(error)) {
        throw error;
      }
    }
  }

  async function downloadExport(mode: ExportMode) {
    const req =
      mode === 'sites'
        ? { site_ids: exportSiteIds }
        : mode === 'tags'
          ? { tag_ids: exportTagIds }
          : {};
    if (mode === 'sites' && exportSiteIds.length === 0) {
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
        setNotice(t('importFailed'));
      }
    } finally {
      event.target.value = '';
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

  function clearSiteSelection() {
    setSelectedSiteIds([]);
    setBatchTagIds([]);
    setSiteSelectionMode(false);
    setMobileBatchTagsOpen(false);
  }

  async function moveSiteByStep(siteId: string, direction: -1 | 1) {
    const currentIndex = filteredSites.findIndex((site) => site.id === siteId);
    const target = filteredSites[currentIndex + direction];
    if (currentIndex < 0 || !target) {
      return;
    }

    const nextSites = swapItemsById(sites, siteId, target.id);
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

  async function moveTagByStep(tagId: string, direction: -1 | 1) {
    const currentIndex = filteredTags.findIndex((tag) => tag.id === tagId);
    const target = filteredTags[currentIndex + direction];
    if (currentIndex < 0 || !target) {
      return;
    }

    const nextTags = swapItemsById(tags, tagId, target.id);
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
        <VersionBadge />
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
        <VersionBadge />
      </main>
    );
  }

  return (
    <main className={tab === 'sites' && selectedSiteIds.length > 0 ? 'admin-shell mobile-dock-active' : 'admin-shell'}>
      <header className="admin-header">
        <div>
          <h1>{t('navboxAdmin')}</h1>
          <p>{t('adminStats', { sites: sites.length, tags: tags.length })}</p>
        </div>
        <button className="mobile-admin-menu-trigger" type="button" onClick={() => setMobileAdminMenuOpen(true)} title={t('adminQuickMenu')}>
          <MoreHorizontal size={19} aria-hidden="true" />
          <span>{t('more')}</span>
        </button>
        <div className="admin-header-actions desktop-admin-actions">
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

      <MobileAdminMenu
        open={mobileAdminMenuOpen}
        onClose={() => setMobileAdminMenuOpen(false)}
        onHome={() => (window.location.href = '/')}
        onLogout={handleLogout}
      />

      <nav className="admin-tabs" aria-label={t('adminModules')}>
        <AdminTabButton active={tab === 'sites'} onClick={() => setTab('sites')} icon={Settings} label={t('sitesTab')} mobileLabel={t('sitesTab')} />
        <AdminTabButton active={tab === 'tags'} onClick={() => setTab('tags')} icon={Tags} label={t('tagsTab')} mobileLabel={t('tagsTab')} />
        <AdminTabButton active={tab === 'io'} onClick={() => setTab('io')} icon={FileUp} label={t('importExportTab')} mobileLabel={t('importShortTab')} />
        <AdminTabButton active={tab === 'password'} onClick={() => setTab('password')} icon={KeyRound} label={t('passwordTab')} mobileLabel={t('passwordTab')} />
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
            <button className="desktop-only" type="button" onClick={saveSiteOrder}>
              <Save size={17} aria-hidden="true" />
              {t('saveOrder')}
            </button>
            <button className="mobile-only mobile-more-action" type="button" onClick={() => setMobileSiteActionsOpen(true)}>
              <MoreHorizontal size={17} aria-hidden="true" />
              {t('more')}
            </button>
            <button className="desktop-only" type="button" onClick={handleBatchDelete} disabled={selectedSiteIds.length === 0}>
              <Trash2 size={17} aria-hidden="true" />
              {t('batchDelete')}
            </button>
          </div>

          <MobileActionMenu open={mobileSiteActionsOpen} title={t('siteActions')} onClose={() => setMobileSiteActionsOpen(false)}>
            <button type="button" onClick={() => {
              setMobileSiteActionsOpen(false);
              saveSiteOrder();
            }}>
              <Save size={17} aria-hidden="true" />
              {t('saveOrder')}
            </button>
            <button type="button" onClick={() => {
              setSiteSelectionMode((current) => !current);
              setMobileSiteActionsOpen(false);
            }}>
              <Check size={17} aria-hidden="true" />
              {siteSelectionMode ? t('done') : t('batchMode')}
            </button>
            <button type="button" onClick={() => {
              setMobileSiteSortMode((current) => !current);
              setMobileSiteActionsOpen(false);
            }}>
              <GripVertical size={17} aria-hidden="true" />
              {mobileSiteSortMode ? t('done') : t('sortMode')}
            </button>
          </MobileActionMenu>

          <AdminListControls
            search={adminSearch}
            onSearchChange={setAdminSearch}
            tags={tags}
            tagFilterId={siteTagFilterId}
            onTagFilterChange={setSiteTagFilterId}
            favoriteFilter={siteFavoriteFilter}
            onFavoriteFilterChange={setSiteFavoriteFilter}
            mode="sites"
          />

          <BatchTagBar
            tags={tags}
            selected={batchTagSet}
            selectedSiteCount={selectedSiteIds.length}
            onToggle={(id) => toggleString(batchTagIds, id, setBatchTagIds)}
            onAdd={() => handleBatchTags('add')}
            onRemove={() => handleBatchTags('remove')}
          />

          {filteredSites.length === 0 ? (
            <AdminState text={t('noSites')} />
          ) : (
            <>
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
                {filteredSites.map((site) => (
                  <tr
                    className={[
                      draggingSiteId === site.id ? 'dragging-row' : '',
                      selectedSiteSet.has(site.id) ? 'selected-row' : ''
                    ].filter(Boolean).join(' ') || undefined}
                    key={site.id}
                    onDragOver={allowSortDrop}
                    onDrop={(event) => dropSite(event, site.id)}
                  >
                    <td className="drag-cell" data-label={t('drag')}>
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
                    <td className="select-cell" data-label={t('select')}>
                      <input
                        type="checkbox"
                        checked={selectedSiteSet.has(site.id)}
                        onChange={() => toggleSiteSelection(site.id)}
                        aria-label={t('selectSite', { title: site.title })}
                      />
                    </td>
                    <td className="site-cell" data-label={t('site')}>
                      <strong>{site.title}</strong>
                      <span>{site.default_url}</span>
                    </td>
                    <td className="tag-cell" data-label="Tag">
                      <div className="table-tags">
                        {site.tags.length > 0 ? (
                          site.tags.map((tag) => (
                            <span className="table-tag" key={tag.id}>{tag.name}</span>
                          ))
                        ) : (
                          <span className="table-tag muted">{t('uncategorized')}</span>
                        )}
                      </div>
                    </td>
                    <td className="open-cell" data-label={t('openMethod')}>
                      <span className="status-pill">{site.open_method === 'current_window' ? t('currentTab') : t('newTab')}</span>
                    </td>
                    <td className="favorite-cell" data-label={t('favoriteColumn')}>
                      <span className={site.is_favorite ? 'status-pill favorite-pill active' : 'status-pill favorite-pill'}>
                        {site.is_favorite ? t('yes') : t('no')}
                      </span>
                    </td>
                    <td className="sort-cell" data-label={t('sort')}>
                      <input
                        className="order-input"
                        type="number"
                        value={siteOrder[site.id] ?? site.sort_order}
                        onChange={(event) => setSiteOrder({ ...siteOrder, [site.id]: Number(event.target.value) })}
                      />
                    </td>
                    <td className="actions-cell" data-label={t('actions')}>
                      <div className="row-actions">
                        <button type="button" onClick={() => openEditSite(site)} title={t('editSite')}>
                          <Pencil size={16} aria-hidden="true" />
                        </button>
                        <button className="row-danger" type="button" onClick={() => removeSite(site.id)} title={t('deleteSite')}>
                          <Trash2 size={16} aria-hidden="true" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="admin-mobile-list">
            {filteredSites.map((site, index) => (
              <MobileSiteCard
                key={site.id}
                site={site}
                selected={selectedSiteSet.has(site.id)}
                selectionMode={siteSelectionMode}
                sortMode={mobileSiteSortMode}
                canMoveUp={index > 0}
                canMoveDown={index < filteredSites.length - 1}
                onToggleSelection={() => toggleSiteSelection(site.id)}
                onEdit={() => openEditSite(site)}
                onDelete={() => removeSite(site.id)}
                onMoveUp={() => moveSiteByStep(site.id, -1)}
                onMoveDown={() => moveSiteByStep(site.id, 1)}
              />
            ))}
          </div>
            </>
          )}
          <MobileBatchDock
            tags={tags}
            selectedTagCount={batchTagIds.length}
            selectedSiteCount={selectedSiteIds.length}
            onOpenTags={() => setMobileBatchTagsOpen(true)}
            onAdd={() => handleBatchTags('add')}
            onRemove={() => handleBatchTags('remove')}
            onDelete={handleBatchDelete}
            onClear={clearSiteSelection}
          />
          <MobileTagSheet
            open={mobileBatchTagsOpen}
            tags={tags}
            selected={batchTagSet}
            onToggle={(id) => toggleString(batchTagIds, id, setBatchTagIds)}
            onClose={() => setMobileBatchTagsOpen(false)}
          />
        </section>
      )}

      {!loading && tab === 'tags' && (
        <section className="admin-section">
          <div className="admin-actions">
            <button className="primary-button" type="button" onClick={openNewTag}>
              <Plus size={17} aria-hidden="true" />
              {t('newTag')}
            </button>
            <button className="desktop-only" type="button" onClick={saveTagOrder}>
              <Save size={17} aria-hidden="true" />
              {t('saveOrder')}
            </button>
            <button className="mobile-only mobile-more-action" type="button" onClick={() => setMobileTagActionsOpen(true)}>
              <MoreHorizontal size={17} aria-hidden="true" />
              {t('more')}
            </button>
          </div>
          <MobileActionMenu open={mobileTagActionsOpen} title={t('tagActions')} onClose={() => setMobileTagActionsOpen(false)}>
            <button type="button" onClick={() => {
              setMobileTagActionsOpen(false);
              saveTagOrder();
            }}>
              <Save size={17} aria-hidden="true" />
              {t('saveOrder')}
            </button>
            <button type="button" onClick={() => {
              setMobileTagSortMode((current) => !current);
              setMobileTagActionsOpen(false);
            }}>
              <GripVertical size={17} aria-hidden="true" />
              {mobileTagSortMode ? t('done') : t('sortMode')}
            </button>
          </MobileActionMenu>
          <AdminListControls
            search={adminSearch}
            onSearchChange={setAdminSearch}
            tags={tags}
            tagFilterId={siteTagFilterId}
            onTagFilterChange={setSiteTagFilterId}
            favoriteFilter={siteFavoriteFilter}
            onFavoriteFilterChange={setSiteFavoriteFilter}
            mode="tags"
          />
          {filteredTags.length === 0 ? (
            <AdminState text={t('noTags')} />
          ) : (
            <>
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
                {filteredTags.map((tag) => (
                  <tr
                    className={draggingTagId === tag.id ? 'dragging-row' : undefined}
                    key={tag.id}
                    onDragOver={allowSortDrop}
                    onDrop={(event) => dropTag(event, tag.id)}
                  >
                    <td className="drag-cell" data-label={t('drag')}>
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
                    <td className="site-cell" data-label="Tag">
                      <strong>
                        <span className="color-swatch" style={{ background: tag.color || '#2f7d6d' }} />
                        {tag.name}
                      </strong>
                      <span>{tag.icon || t('noIcon')}</span>
                    </td>
                    <td className="status-cell" data-label={t('status')}>
                      <span className={tag.is_enabled ? 'status-pill active' : 'status-pill'}>
                        {tag.is_default ? t('defaultStatus') : tag.is_enabled ? t('enabledStatus') : t('disabledStatus')}
                      </span>
                    </td>
                    <td className="count-cell" data-label={t('siteCount')}>{tag.site_count}</td>
                    <td className="sort-cell" data-label={t('sort')}>
                      <input
                        className="order-input"
                        type="number"
                        value={tagOrder[tag.id] ?? tag.sort_order}
                        onChange={(event) => setTagOrder({ ...tagOrder, [tag.id]: Number(event.target.value) })}
                      />
                    </td>
                    <td className="actions-cell" data-label={t('actions')}>
                      <div className="row-actions">
                        {!tag.is_default && (
                          <button type="button" onClick={() => makeDefaultTag(tag.id)} title={t('setDefault')}>
                            <Check size={16} aria-hidden="true" />
                          </button>
                        )}
                        <button type="button" onClick={() => openEditTag(tag)} title={t('editTag')}>
                          <Pencil size={16} aria-hidden="true" />
                        </button>
                        <button className="row-danger" type="button" onClick={() => removeTag(tag.id)} title={t('deleteTag')}>
                          <Trash2 size={16} aria-hidden="true" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="admin-mobile-list">
            {filteredTags.map((tag, index) => (
              <MobileTagCard
                key={tag.id}
                tag={tag}
                sortMode={mobileTagSortMode}
                canMoveUp={index > 0}
                canMoveDown={index < filteredTags.length - 1}
                onSetDefault={() => makeDefaultTag(tag.id)}
                onEdit={() => openEditTag(tag)}
                onDelete={() => removeTag(tag.id)}
                onMoveUp={() => moveTagByStep(tag.id, -1)}
                onMoveDown={() => moveTagByStep(tag.id, 1)}
              />
            ))}
          </div>
            </>
          )}
        </section>
      )}

      {!loading && tab === 'io' && (
        <section className="admin-section admin-grid-two">
          <div className="admin-tool-panel">
            <h2>{t('export')}</h2>
            <div className="stack">
              <div className="export-scope" role="group" aria-label={t('exportScope')}>
                <button
                  className={exportMode === 'all' ? 'active' : undefined}
                  type="button"
                  onClick={() => setExportMode('all')}
                >
                  {t('exportScopeAll')}
                </button>
                <button
                  className={exportMode === 'sites' ? 'active' : undefined}
                  type="button"
                  onClick={() => setExportMode('sites')}
                >
                  {t('exportScopeSites')}
                </button>
                <button
                  className={exportMode === 'tags' ? 'active' : undefined}
                  type="button"
                  onClick={() => setExportMode('tags')}
                >
                  {t('exportScopeTags')}
                </button>
              </div>

              {exportMode === 'sites' && (
                <div className="export-picker">
                  <div className="export-picker-header">
                    <strong>{t('siteSelection')}</strong>
                    <span>{t('selectedCount', { count: exportSiteIds.length })}</span>
                    <div className="export-picker-actions">
                      <button type="button" onClick={() => setExportSiteIds(sites.map((site) => site.id))}>
                        {t('selectAll')}
                      </button>
                      <button type="button" onClick={() => setExportSiteIds([])}>
                        {t('clearSelection')}
                      </button>
                    </div>
                  </div>
                  <div className="export-list">
                    {sites.map((site) => (
                      <label className="export-option" key={site.id}>
                        <input
                          type="checkbox"
                          checked={exportSiteSet.has(site.id)}
                          onChange={() => toggleString(exportSiteIds, site.id, setExportSiteIds)}
                        />
                        <span className="export-option-main">
                          <strong>{site.title}</strong>
                          <span>{site.default_url}</span>
                        </span>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              {exportMode === 'tags' && (
                <div className="export-picker">
                  <div className="export-picker-header">
                    <strong>{t('tagSelection')}</strong>
                    <span>{t('selectedCount', { count: exportTagIds.length })}</span>
                    <div className="export-picker-actions">
                      <button type="button" onClick={() => setExportTagIds(tags.map((tag) => tag.id))}>
                        {t('selectAll')}
                      </button>
                      <button type="button" onClick={() => setExportTagIds([])}>
                        {t('clearSelection')}
                      </button>
                    </div>
                  </div>
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
                </div>
              )}

              <button className="primary-button" type="button" onClick={() => downloadExport(exportMode)}>
                <Download size={17} aria-hidden="true" />
                {t('exportNow')}
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
      <VersionBadge />
    </main>
  );
}

function MobileAdminMenu({
  open,
  onClose,
  onHome,
  onLogout
}: {
  open: boolean;
  onClose: () => void;
  onHome: () => void;
  onLogout: () => void;
}) {
  const { t } = usePreferences();
  if (!open) {
    return null;
  }

  return (
    <div className="mobile-sheet-backdrop" onClick={onClose}>
      <section className="mobile-sheet mobile-admin-menu" onClick={(event) => event.stopPropagation()}>
        <header>
          <h2>{t('adminQuickMenu')}</h2>
          <button type="button" onClick={onClose} title={t('close')}>
            <X size={18} aria-hidden="true" />
          </button>
        </header>
        <PreferenceControls />
        <div className="mobile-sheet-actions">
          <button type="button" onClick={onHome}>
            {t('visitorHome')}
          </button>
          <button type="button" onClick={onLogout}>
            <LogOut size={17} aria-hidden="true" />
            {t('logout')}
          </button>
        </div>
      </section>
    </div>
  );
}

function MobileActionMenu({
  open,
  title,
  children,
  onClose
}: {
  open: boolean;
  title: string;
  children: ReactNode;
  onClose: () => void;
}) {
  const { t } = usePreferences();
  if (!open) {
    return null;
  }

  return (
    <div className="mobile-sheet-backdrop" onClick={onClose}>
      <section className="mobile-sheet mobile-action-menu" onClick={(event) => event.stopPropagation()}>
        <header>
          <h2>{title}</h2>
          <button type="button" onClick={onClose} title={t('close')}>
            <X size={18} aria-hidden="true" />
          </button>
        </header>
        <div className="mobile-sheet-actions">
          {children}
        </div>
      </section>
    </div>
  );
}

function AdminListControls({
  search,
  onSearchChange,
  tags,
  tagFilterId,
  onTagFilterChange,
  favoriteFilter,
  onFavoriteFilterChange,
  mode
}: {
  search: string;
  onSearchChange: (value: string) => void;
  tags: Tag[];
  tagFilterId: string;
  onTagFilterChange: (value: string) => void;
  favoriteFilter: FavoriteFilter;
  onFavoriteFilterChange: (value: FavoriteFilter) => void;
  mode: 'sites' | 'tags';
}) {
  const { t } = usePreferences();

  return (
    <div className="admin-list-controls">
      <label className="admin-search-field">
        <Search size={17} aria-hidden="true" />
        <input
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder={t('adminSearchPlaceholder')}
          aria-label={t('adminSearch')}
        />
      </label>
      {mode === 'sites' && (
        <div className="admin-filter-row">
          <label className="admin-filter-select">
            <Tags size={16} aria-hidden="true" />
            <select value={tagFilterId} onChange={(event) => onTagFilterChange(event.target.value)} aria-label={t('tagFilter')}>
              <option value="all">{t('allTags')}</option>
              {tags.map((tag) => (
                <option key={tag.id} value={tag.id}>{tag.name}</option>
              ))}
            </select>
          </label>
          <button
            className={favoriteFilter === 'favorites' ? 'active' : undefined}
            type="button"
            onClick={() => onFavoriteFilterChange(favoriteFilter === 'favorites' ? 'all' : 'favorites')}
          >
            {favoriteFilter === 'favorites' ? t('favoritesOnly') : t('allSitesFilter')}
          </button>
        </div>
      )}
    </div>
  );
}

function MobileSiteCard({
  site,
  selected,
  selectionMode,
  sortMode,
  canMoveUp,
  canMoveDown,
  onToggleSelection,
  onEdit,
  onDelete,
  onMoveUp,
  onMoveDown
}: {
  site: Site;
  selected: boolean;
  selectionMode: boolean;
  sortMode: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onToggleSelection: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
}) {
  const { t } = usePreferences();

  return (
    <article className={selected ? 'mobile-admin-card selected' : 'mobile-admin-card'}>
      <div className="mobile-card-head">
        {(selectionMode || selected) && (
          <label className="mobile-select-box">
            <input
              type="checkbox"
              checked={selected}
              onChange={onToggleSelection}
              aria-label={t('selectSite', { title: site.title })}
            />
          </label>
        )}
        <div className="mobile-card-title">
          <strong>{site.title}</strong>
          <span>{site.default_url}</span>
        </div>
        <div className="row-actions mobile-row-actions">
          <button type="button" onClick={onEdit} title={t('editSite')}>
            <Pencil size={16} aria-hidden="true" />
          </button>
          <button className="row-danger" type="button" onClick={onDelete} title={t('deleteSite')}>
            <Trash2 size={16} aria-hidden="true" />
          </button>
        </div>
      </div>
      {site.description && <p className="mobile-card-description">{site.description}</p>}
      <div className="mobile-card-meta">
        <span className="status-pill">{site.open_method === 'current_window' ? t('currentTab') : t('newTab')}</span>
        <span className={site.is_favorite ? 'status-pill favorite-pill active' : 'status-pill favorite-pill'}>
          {site.is_favorite ? t('favorite') : t('no')}
        </span>
        <span className="status-pill">{t('sort')}: {site.sort_order}</span>
      </div>
      <div className="table-tags mobile-card-tags">
        {site.tags.length > 0 ? (
          site.tags.map((tag) => (
            <span className="table-tag" key={tag.id}>{tag.name}</span>
          ))
        ) : (
          <span className="table-tag muted">{t('uncategorized')}</span>
        )}
      </div>
      {sortMode && (
        <div className="mobile-sort-controls">
          <button type="button" onClick={onMoveUp} disabled={!canMoveUp}>
            <ArrowUp size={16} aria-hidden="true" />
            {t('moveUp')}
          </button>
          <button type="button" onClick={onMoveDown} disabled={!canMoveDown}>
            <ArrowDown size={16} aria-hidden="true" />
            {t('moveDown')}
          </button>
        </div>
      )}
    </article>
  );
}

function MobileTagCard({
  tag,
  sortMode,
  canMoveUp,
  canMoveDown,
  onSetDefault,
  onEdit,
  onDelete,
  onMoveUp,
  onMoveDown
}: {
  tag: Tag;
  sortMode: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onSetDefault: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
}) {
  const { t } = usePreferences();

  return (
    <article className="mobile-admin-card">
      <div className="mobile-card-head">
        <div className="mobile-card-title">
          <strong>
            <span className="color-swatch" style={{ background: tag.color || '#2f7d6d' }} />
            {tag.name}
          </strong>
          <span>{tag.icon || t('noIcon')}</span>
        </div>
        <div className="row-actions mobile-row-actions">
          {!tag.is_default && (
            <button type="button" onClick={onSetDefault} title={t('setDefault')}>
              <Check size={16} aria-hidden="true" />
            </button>
          )}
          <button type="button" onClick={onEdit} title={t('editTag')}>
            <Pencil size={16} aria-hidden="true" />
          </button>
          <button className="row-danger" type="button" onClick={onDelete} title={t('deleteTag')}>
            <Trash2 size={16} aria-hidden="true" />
          </button>
        </div>
      </div>
      <div className="mobile-card-meta">
        <span className={tag.is_enabled ? 'status-pill active' : 'status-pill'}>
          {tag.is_default ? t('defaultStatus') : tag.is_enabled ? t('enabledStatus') : t('disabledStatus')}
        </span>
        <span className="status-pill">{t('siteCount')}: {tag.site_count}</span>
        <span className="status-pill">{t('sort')}: {tag.sort_order}</span>
      </div>
      {sortMode && (
        <div className="mobile-sort-controls">
          <button type="button" onClick={onMoveUp} disabled={!canMoveUp}>
            <ArrowUp size={16} aria-hidden="true" />
            {t('moveUp')}
          </button>
          <button type="button" onClick={onMoveDown} disabled={!canMoveDown}>
            <ArrowDown size={16} aria-hidden="true" />
            {t('moveDown')}
          </button>
        </div>
      )}
    </article>
  );
}

function MobileBatchDock({
  tags,
  selectedTagCount,
  selectedSiteCount,
  onOpenTags,
  onAdd,
  onRemove,
  onDelete,
  onClear
}: {
  tags: Tag[];
  selectedTagCount: number;
  selectedSiteCount: number;
  onOpenTags: () => void;
  onAdd: () => void;
  onRemove: () => void;
  onDelete: () => void;
  onClear: () => void;
}) {
  const { t } = usePreferences();
  if (selectedSiteCount === 0) {
    return null;
  }

  return (
    <div className="mobile-batch-dock">
      <div className="mobile-batch-summary">
        <strong>{t('selectedCount', { count: selectedSiteCount })}</strong>
        <span>{selectedTagCount > 0 ? t('selectedTagsCount', { count: selectedTagCount }) : t('chooseBatchTags')}</span>
      </div>
      <div className="mobile-batch-actions">
        <button type="button" onClick={onOpenTags} disabled={tags.length === 0}>
          <Tags size={16} aria-hidden="true" />
          {t('selectBatchTags')}
        </button>
        <button type="button" onClick={onAdd} disabled={selectedTagCount === 0}>{t('batchAddTag')}</button>
        <button type="button" onClick={onRemove} disabled={selectedTagCount === 0}>{t('batchRemoveTag')}</button>
        <button className="row-danger" type="button" onClick={onDelete}>
          <Trash2 size={16} aria-hidden="true" />
        </button>
        <button type="button" onClick={onClear}>{t('clearSelection')}</button>
      </div>
    </div>
  );
}

function MobileTagSheet({
  open,
  tags,
  selected,
  onToggle,
  onClose
}: {
  open: boolean;
  tags: Tag[];
  selected: Set<string>;
  onToggle: (id: string) => void;
  onClose: () => void;
}) {
  const { t } = usePreferences();
  if (!open) {
    return null;
  }

  return (
    <div className="mobile-sheet-backdrop" onClick={onClose}>
      <section className="mobile-sheet" onClick={(event) => event.stopPropagation()}>
        <header>
          <h2>{t('batchTagSheetTitle')}</h2>
          <button type="button" onClick={onClose} title={t('close')}>
            <X size={18} aria-hidden="true" />
          </button>
        </header>
        <div className="checkbox-list mobile-sheet-options">
          {tags.map((tag) => (
            <label key={tag.id}>
              <input type="checkbox" checked={selected.has(tag.id)} onChange={() => onToggle(tag.id)} />
              <span>{tag.name}</span>
            </label>
          ))}
        </div>
        <button className="primary-button" type="button" onClick={onClose}>{t('done')}</button>
      </section>
    </div>
  );
}

function AdminTabButton({
  active,
  icon: Icon,
  label,
  mobileLabel,
  onClick
}: {
  active: boolean;
  icon: typeof Settings;
  label: string;
  mobileLabel: string;
  onClick: () => void;
}) {
  return (
    <button className={active ? 'admin-tab active' : 'admin-tab'} type="button" onClick={onClick}>
      <Icon size={17} aria-hidden="true" />
      <span className="admin-tab-label">{label}</span>
      <span className="admin-tab-mobile-label">{mobileLabel}</span>
    </button>
  );
}

function BatchTagBar({
  tags,
  selected,
  selectedSiteCount,
  onToggle,
  onAdd,
  onRemove
}: {
  tags: Tag[];
  selected: Set<string>;
  selectedSiteCount: number;
  onToggle: (id: string) => void;
  onAdd: () => void;
  onRemove: () => void;
}) {
  const { t } = usePreferences();
  const actionDisabled = selectedSiteCount === 0 || selected.size === 0;

  return (
    <div className="batch-bar desktop-only">
      <div className="batch-summary">{t('selectedCount', { count: selectedSiteCount })}</div>
      <div className="checkbox-list compact">
        {tags.map((tag) => (
          <label key={tag.id}>
            <input type="checkbox" checked={selected.has(tag.id)} onChange={() => onToggle(tag.id)} />
            <span>{tag.name}</span>
          </label>
        ))}
      </div>
      <button type="button" onClick={onAdd} disabled={actionDisabled}>{t('batchAddTag')}</button>
      <button type="button" onClick={onRemove} disabled={actionDisabled}>{t('batchRemoveTag')}</button>
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
    <form className="admin-form site-form" onSubmit={onSubmit}>
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
        <span>{t('openMethod')}</span>
        <select value={draft.open_method} onChange={(event) => patch({ open_method: event.target.value })}>
          <option value="new_window">{t('newWindow')}</option>
          <option value="current_window">{t('currentWindow')}</option>
        </select>
      </label>
      <label className="site-description-field">
        <span>{t('description')}</span>
        <textarea value={draft.description} onChange={(event) => patch({ description: event.target.value })} rows={3} />
      </label>
      <div className="form-grid form-grid-three">
        <label>
          <span>{t('sort')}</span>
          <input type="number" value={draft.sort_order} onChange={(event) => patch({ sort_order: Number(event.target.value) })} />
        </label>
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
      <div className="form-grid icon-field-row">
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
            <input
              type="file"
              accept="image/png,image/jpeg,image/gif,image/webp,image/x-icon,image/svg+xml,.svg"
              onChange={onIconUpload}
            />
          </label>
        </div>
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

function swapItemsById<T extends { id: string }>(items: T[], sourceId: string, targetId: string): T[] {
  const sourceIndex = items.findIndex((item) => item.id === sourceId);
  const targetIndex = items.findIndex((item) => item.id === targetId);
  if (sourceIndex < 0 || targetIndex < 0) {
    return items;
  }

  const next = [...items];
  [next[sourceIndex], next[targetIndex]] = [next[targetIndex], next[sourceIndex]];
  return next;
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
