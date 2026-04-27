import {
  Check,
  Download,
  FileUp,
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
import { type ChangeEvent, type FormEvent, type ReactNode, useEffect, useMemo, useState } from 'react';
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
  importConfig,
  listSites,
  listTags,
  loginAdmin,
  logoutAdmin,
  setDefaultTag,
  updateSite,
  updateSiteOrder,
  updateTag,
  updateTagOrder,
  uploadIcon
} from './api';
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
  const [checking, setChecking] = useState(true);
  const [authenticated, setAuthenticated] = useState(false);
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
  const [importReport, setImportReport] = useState<ImportReport | null>(null);
  const [passwordForm, setPasswordForm] = useState({ current: '', next: '' });

  useEffect(() => {
    let cancelled = false;
    async function checkSession() {
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

  async function refreshData() {
    setLoading(true);
    try {
      const [siteList, tagList] = await Promise.all([listSites({}), listTags()]);
      setSites(siteList);
      setTags(tagList);
      setSiteOrder(Object.fromEntries(siteList.map((site) => [site.id, site.sort_order])));
      setTagOrder(Object.fromEntries(tagList.map((tag) => [tag.id, tag.sort_order])));
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
      setLoginError('密码错误或 Session 创建失败');
    }
  }

  async function handleLogout() {
    await logoutAdmin();
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
    if (editingSiteId) {
      await updateSite(editingSiteId, siteDraft);
      setNotice('网站已更新');
    } else {
      await createSite(siteDraft);
      setNotice('网站已创建');
    }
    setSitePanelOpen(false);
    await refreshData();
  }

  async function removeSite(id: string) {
    if (!window.confirm('确认删除这个网站？')) {
      return;
    }
    await deleteSite(id);
    setSelectedSiteIds((current) => current.filter((item) => item !== id));
    setNotice('网站已删除');
    await refreshData();
  }

  async function handleIconUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    const icon = await uploadIcon(file);
    event.target.value = '';
    setSiteDraft((current) => ({
      ...current,
      icon_type: 'image',
      icon_value: icon.url
    }));
    setNotice('icon 已上传');
  }

  async function handleFetchIcon() {
    const targetURL = siteDraft.default_url.trim();
    if (!targetURL) {
      setNotice('请先填写默认 URL');
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
      setNotice('icon 已获取');
    } catch {
      setNotice('未获取到 icon');
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
    if (editingTagId) {
      await updateTag(editingTagId, tagDraft);
      setNotice('Tag 已更新');
    } else {
      await createTag(tagDraft);
      setNotice('Tag 已创建');
    }
    setTagPanelOpen(false);
    await refreshData();
  }

  async function removeTag(id: string) {
    if (!window.confirm('确认删除这个 Tag？关联网站不会被删除。')) {
      return;
    }
    await deleteTag(id);
    setNotice('Tag 已删除');
    await refreshData();
  }

  async function makeDefaultTag(id: string) {
    await setDefaultTag(id);
    setNotice('默认 Tag 已更新');
    await refreshData();
  }

  async function saveSiteOrder() {
    await updateSiteOrder(toOrderItems(siteOrder));
    setNotice('网站排序已保存');
    await refreshData();
  }

  async function saveTagOrder() {
    await updateTagOrder(toOrderItems(tagOrder));
    setNotice('Tag 排序已保存');
    await refreshData();
  }

  async function handleBatchDelete() {
    if (selectedSiteIds.length === 0 || !window.confirm(`确认删除选中的 ${selectedSiteIds.length} 个网站？`)) {
      return;
    }
    await batchDeleteSites(selectedSiteIds);
    setSelectedSiteIds([]);
    setNotice('批量删除已完成');
    await refreshData();
  }

  async function handleBatchTags(action: 'add' | 'remove') {
    if (selectedSiteIds.length === 0 || batchTagIds.length === 0) {
      setNotice('请选择网站和 Tag');
      return;
    }
    await batchUpdateSiteTags(selectedSiteIds, batchTagIds, action);
    setNotice(action === 'add' ? '批量添加 Tag 已完成' : '批量移除 Tag 已完成');
    await refreshData();
  }

  async function downloadExport(mode: 'all' | 'sites' | 'tags') {
    const req =
      mode === 'sites'
        ? { site_ids: selectedSiteIds }
        : mode === 'tags'
          ? { tag_ids: exportTagIds }
          : {};
    if (mode === 'sites' && selectedSiteIds.length === 0) {
      setNotice('请先选择网站');
      return;
    }
    if (mode === 'tags' && exportTagIds.length === 0) {
      setNotice('请先选择 Tag');
      return;
    }

    const file = await exportConfig(req);
    const url = URL.createObjectURL(file.blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = file.fileName;
    link.click();
    URL.revokeObjectURL(url);
    setNotice('导出已开始');
  }

  async function handleImport(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    const report = await importConfig(file);
    setImportReport(report);
    setNotice('导入已完成');
    await refreshData();
  }

  async function submitPassword(event: FormEvent) {
    event.preventDefault();
    await changeAdminPassword(passwordForm.current, passwordForm.next);
    setPasswordForm({ current: '', next: '' });
    setNotice('密码已修改');
  }

  function toggleSiteSelection(id: string) {
    setSelectedSiteIds((current) => (current.includes(id) ? current.filter((item) => item !== id) : [...current, id]));
  }

  function toggleString(current: string[], id: string, setter: (value: string[]) => void) {
    setter(current.includes(id) ? current.filter((item) => item !== id) : [...current, id]);
  }

  if (checking) {
    return <AdminState text="正在检查 Session" />;
  }

  if (!authenticated) {
    return (
      <main className="admin-login">
        <form className="login-panel" onSubmit={submitLogin}>
          <div className="login-mark">
            <Shield size={26} aria-hidden="true" />
          </div>
          <h1>Navbox Admin</h1>
          <label>
            <span>密码</span>
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
            登录
          </button>
        </form>
      </main>
    );
  }

  return (
    <main className="admin-shell">
      <header className="admin-header">
        <div>
          <h1>Navbox Admin</h1>
          <p>{sites.length} 个网站，{tags.length} 个 Tag</p>
        </div>
        <div className="admin-header-actions">
          <button type="button" onClick={() => (window.location.href = '/')}>
            游客首页
          </button>
          <button type="button" onClick={handleLogout}>
            <LogOut size={17} aria-hidden="true" />
            退出
          </button>
        </div>
      </header>

      <nav className="admin-tabs" aria-label="admin 管理模块">
        <AdminTabButton active={tab === 'sites'} onClick={() => setTab('sites')} icon={Settings} label="网站" />
        <AdminTabButton active={tab === 'tags'} onClick={() => setTab('tags')} icon={Tags} label="Tag" />
        <AdminTabButton active={tab === 'io'} onClick={() => setTab('io')} icon={FileUp} label="导入导出" />
        <AdminTabButton active={tab === 'password'} onClick={() => setTab('password')} icon={KeyRound} label="密码" />
      </nav>

      {notice && <div className="toast">{notice}</div>}
      {loading && <AdminState text="正在刷新数据" />}

      {!loading && tab === 'sites' && (
        <section className="admin-section">
          <div className="admin-actions">
            <button className="primary-button" type="button" onClick={openNewSite}>
              <Plus size={17} aria-hidden="true" />
              新建网站
            </button>
            <button type="button" onClick={saveSiteOrder}>
              <Save size={17} aria-hidden="true" />
              保存排序
            </button>
            <button type="button" onClick={handleBatchDelete}>
              <Trash2 size={17} aria-hidden="true" />
              批量删除
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
                  <th>选择</th>
                  <th>网站</th>
                  <th>Tag</th>
                  <th>常用</th>
                  <th>排序</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {sites.map((site) => (
                  <tr key={site.id}>
                    <td>
                      <input
                        type="checkbox"
                        checked={selectedSiteSet.has(site.id)}
                        onChange={() => toggleSiteSelection(site.id)}
                        aria-label={`选择 ${site.title}`}
                      />
                    </td>
                    <td>
                      <strong>{site.title}</strong>
                      <span>{site.default_url}</span>
                    </td>
                    <td>{site.tags.map((tag) => tag.name).join(', ') || '未分类'}</td>
                    <td>{site.is_favorite ? '是' : '否'}</td>
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
                        <button type="button" onClick={() => openEditSite(site)} title="编辑网站">
                          <Pencil size={16} aria-hidden="true" />
                        </button>
                        <button type="button" onClick={() => removeSite(site.id)} title="删除网站">
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
              新建 Tag
            </button>
            <button type="button" onClick={saveTagOrder}>
              <Save size={17} aria-hidden="true" />
              保存排序
            </button>
          </div>
          <div className="admin-table-wrap">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>Tag</th>
                  <th>状态</th>
                  <th>网站数</th>
                  <th>排序</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {tags.map((tag) => (
                  <tr key={tag.id}>
                    <td>
                      <strong>
                        <span className="color-swatch" style={{ background: tag.color || '#2f7d6d' }} />
                        {tag.name}
                      </strong>
                      <span>{tag.icon || '无图标'}</span>
                    </td>
                    <td>{tag.is_default ? '默认' : tag.is_enabled ? '启用' : '停用'}</td>
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
                          <button type="button" onClick={() => makeDefaultTag(tag.id)} title="设为默认">
                            <Check size={16} aria-hidden="true" />
                          </button>
                        )}
                        <button type="button" onClick={() => openEditTag(tag)} title="编辑 Tag">
                          <Pencil size={16} aria-hidden="true" />
                        </button>
                        <button type="button" onClick={() => removeTag(tag.id)} title="删除 Tag">
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
            <h2>导出</h2>
            <div className="stack">
              <button className="primary-button" type="button" onClick={() => downloadExport('all')}>
                <Download size={17} aria-hidden="true" />
                导出全部
              </button>
              <button type="button" onClick={() => downloadExport('sites')}>
                导出选中网站
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
                导出选中 Tag
              </button>
            </div>
          </div>

          <div className="admin-tool-panel">
            <h2>导入</h2>
            <label className="file-button">
              <Upload size={17} aria-hidden="true" />
              选择 zip 导入
              <input type="file" accept=".zip,application/zip" onChange={handleImport} />
            </label>
            {importReport && (
              <div className="report-box">
                <strong>导入结果</strong>
                <span>新增网站 {importReport.imported.sites}，Tag {importReport.imported.tags}，icon {importReport.imported.icons}</span>
                <span>跳过网站 {importReport.skipped.sites}，Tag {importReport.skipped.tags}，icon {importReport.skipped.icons}</span>
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
              <span>当前密码</span>
              <input
                type="password"
                value={passwordForm.current}
                onChange={(event) => setPasswordForm({ ...passwordForm, current: event.target.value })}
                required
              />
            </label>
            <label>
              <span>新密码</span>
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
              修改密码
            </button>
          </form>
        </section>
      )}

      {sitePanelOpen && (
        <SidePanel title={editingSiteId ? '编辑网站' : '新建网站'} onClose={() => setSitePanelOpen(false)}>
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
        <SidePanel title={editingTagId ? '编辑 Tag' : '新建 Tag'} onClose={() => setTagPanelOpen(false)}>
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
      <button type="button" onClick={onAdd}>批量添加 Tag</button>
      <button type="button" onClick={onRemove}>批量移除 Tag</button>
    </div>
  );
}

function SidePanel({ title, children, onClose }: { title: string; children: ReactNode; onClose: () => void }) {
  return (
    <div className="panel-backdrop">
      <aside className="side-panel">
        <header>
          <h2>{title}</h2>
          <button type="button" onClick={onClose} title="关闭">
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
        <span>标题</span>
        <input value={draft.title} onChange={(event) => patch({ title: event.target.value })} required />
      </label>
      <label>
        <span>默认 URL</span>
        <input value={draft.default_url} onChange={(event) => patch({ default_url: event.target.value })} required />
      </label>
      <label>
        <span>LAN URL</span>
        <input value={draft.lan_url} onChange={(event) => patch({ lan_url: event.target.value })} />
      </label>
      <label>
        <span>描述</span>
        <textarea value={draft.description} onChange={(event) => patch({ description: event.target.value })} rows={3} />
      </label>
      <div className="form-grid">
        <label>
          <span>打开方式</span>
          <select value={draft.open_method} onChange={(event) => patch({ open_method: event.target.value })}>
            <option value="new_window">新窗口</option>
            <option value="current_window">当前窗口</option>
          </select>
        </label>
        <label>
          <span>排序</span>
          <input type="number" value={draft.sort_order} onChange={(event) => patch({ sort_order: Number(event.target.value) })} />
        </label>
      </div>
      <div className="form-grid">
        <label>
          <span>图标类型</span>
          <select value={draft.icon_type} onChange={(event) => patch({ icon_type: event.target.value })}>
            <option value="text">文本</option>
            <option value="image">图片</option>
            <option value="online">在线</option>
          </select>
        </label>
        <label>
          <span>背景色</span>
          <input value={draft.background_color} onChange={(event) => patch({ background_color: event.target.value })} />
        </label>
      </div>
      <label>
        <span>图标值</span>
        <input value={draft.icon_value} onChange={(event) => patch({ icon_value: event.target.value })} />
      </label>
      <div className="icon-actions">
        <button type="button" onClick={onFetchIcon} disabled={iconFetching || !draft.default_url.trim()}>
          <RefreshCw className={iconFetching ? 'spin' : ''} size={17} aria-hidden="true" />
          {iconFetching ? '获取中' : '从网站获取'}
        </button>
        <label className="file-button inline-file">
          <Upload size={17} aria-hidden="true" />
          上传 icon
          <input type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/x-icon" onChange={onIconUpload} />
        </label>
      </div>
      <div className="checkbox-row">
        <label>
          <input type="checkbox" checked={draft.only_name} onChange={(event) => patch({ only_name: event.target.checked })} />
          <span>仅显示名称</span>
        </label>
        <label>
          <input type="checkbox" checked={draft.is_favorite} onChange={(event) => patch({ is_favorite: event.target.checked })} />
          <span>常用</span>
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
        保存网站
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
  function patch(next: Partial<TagSaveReq>) {
    onChange({ ...draft, ...next });
  }

  return (
    <form className="admin-form" onSubmit={onSubmit}>
      <label>
        <span>名称</span>
        <input value={draft.name} onChange={(event) => patch({ name: event.target.value })} required />
      </label>
      <label>
        <span>图标</span>
        <input value={draft.icon} onChange={(event) => patch({ icon: event.target.value })} />
      </label>
      <div className="form-grid">
        <label>
          <span>颜色</span>
          <input type="color" value={draft.color || '#2f7d6d'} onChange={(event) => patch({ color: event.target.value })} />
        </label>
        <label>
          <span>排序</span>
          <input type="number" value={draft.sort_order} onChange={(event) => patch({ sort_order: Number(event.target.value) })} />
        </label>
      </div>
      <label className="checkbox-line">
        <input type="checkbox" checked={draft.is_enabled} onChange={(event) => patch({ is_enabled: event.target.checked })} />
        <span>启用</span>
      </label>
      <button className="primary-button" type="submit">
        <Save size={17} aria-hidden="true" />
        保存 Tag
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
