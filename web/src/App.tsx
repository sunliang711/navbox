import {
  ArrowRight,
  Copy,
  ExternalLink,
  Heart,
  Loader2,
  MoreHorizontal,
  Search,
  X
} from 'lucide-react';
import { type MouseEvent, useEffect, useMemo, useRef, useState } from 'react';
import { listSites, listTags } from './api';
import { AdminApp } from './AdminApp';
import { PreferenceControls, PreferencesProvider, usePreferences } from './preferences';
import { loadRecentSites, saveRecentSite } from './recent';
import { VersionBadge } from './version';
import './styles.css';
import type { RecentSite, Site, Tag, TagMatchMode } from './types';

type ViewMode = 'all' | 'favorite' | 'recent';
type OpenMethod = 'new_window' | 'current_window';

const preferLanKey = 'navbox_prefer_lan';
const previewEnabledKey = 'navbox_preview_enabled';
const tagQueryKey = 'tags';
const tagMatchQueryKey = 'match';
const viewQueryKey = 'view';

export function App() {
  return (
    <PreferencesProvider>
      {window.location.pathname.startsWith('/admin') ? <AdminApp /> : <VisitorApp />}
    </PreferencesProvider>
  );
}

function VisitorApp() {
  const { t } = usePreferences();
  const [tags, setTags] = useState<Tag[]>([]);
  const [sites, setSites] = useState<Site[]>([]);
  const [recentSites, setRecentSites] = useState<RecentSite[]>(() => loadRecentSites());
  const [selectedTagIds, setSelectedTagIds] = useState<string[]>([]);
  const [tagMatch, setTagMatch] = useState<TagMatchMode>('all');
  const [search, setSearch] = useState('');
  const [view, setView] = useState<ViewMode>('all');
  const [preferLan, setPreferLan] = useState(() => loadPreferLan());
  const [previewEnabled, setPreviewEnabled] = useState(() => loadPreviewEnabled());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [ready, setReady] = useState(false);
  const [notice, setNotice] = useState('');
  const [settingsOpen, setSettingsOpen] = useState(false);
  const toolbarMenuRef = useRef<HTMLDivElement | null>(null);
  const toolbarMenuButtonRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadInitialData() {
      setLoading(true);
      setError('');
      try {
        const tagList = await listTags();
        if (cancelled) {
          return;
        }
        const enabledTags = tagList.filter((tag) => tag.is_enabled);
        const queryState = loadVisitorQueryState(enabledTags);
        setTags(enabledTags);
        setTagMatch(queryState.tagMatch);
        if (queryState.view !== 'all') {
          setView(queryState.view);
          setSelectedTagIds([]);
        } else if (queryState.hasQueryState) {
          setSelectedTagIds(queryState.tagIds);
        }
        setReady(true);
      } catch {
        if (!cancelled) {
          setError(t('homeLoadFailed'));
          setReady(true);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadInitialData();
    return () => {
      cancelled = true;
    };
  }, [t]);

  useEffect(() => {
    if (!ready || view === 'recent') {
      return;
    }

    let cancelled = false;
    async function loadSiteData() {
      setLoading(true);
      setError('');
      try {
        const data = await listSites({
          search: search.trim(),
          tagIds: selectedTagIds,
          tagMatch: selectedTagIds.length > 0 ? tagMatch : undefined,
          view: view === 'all' ? undefined : view
        });
        if (!cancelled) {
          setSites(data);
        }
      } catch {
        if (!cancelled) {
          setError(t('siteListLoadFailed'));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    loadSiteData();
    return () => {
      cancelled = true;
    };
  }, [ready, search, selectedTagIds, t, tagMatch, view]);

  useEffect(() => {
    if (!notice) {
      return;
    }
    const timer = window.setTimeout(() => setNotice(''), 1800);
    return () => window.clearTimeout(timer);
  }, [notice]);

  useEffect(() => {
    if (!settingsOpen) {
      return;
    }

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setSettingsOpen(false);
      }
    }

    function closeOnOutside(event: PointerEvent) {
      const target = event.target;
      if (!(target instanceof Node)) {
        setSettingsOpen(false);
        return;
      }
      if (toolbarMenuRef.current?.contains(target) || toolbarMenuButtonRef.current?.contains(target)) {
        return;
      }
      setSettingsOpen(false);
    }

    window.addEventListener('pointerdown', closeOnOutside, true);
    window.addEventListener('keydown', closeOnEscape);
    return () => {
      window.removeEventListener('pointerdown', closeOnOutside, true);
      window.removeEventListener('keydown', closeOnEscape);
    };
  }, [settingsOpen]);

  const selectedTagSet = useMemo(() => new Set(selectedTagIds), [selectedTagIds]);
  const currentTitle = useMemo(() => {
    if (view === 'favorite') {
      return t('favoriteTools');
    }
    if (view === 'recent') {
      return t('recentVisits');
    }
    if (selectedTagIds.length === 1) {
      return tags.find((tag) => tag.id === selectedTagIds[0])?.name || t('allTools');
    }
    if (selectedTagIds.length > 1) {
      return t('selectedCategoryCount', { count: selectedTagIds.length });
    }
    return t('allTools');
  }, [selectedTagIds, t, tags, view]);
  const displaySites = view === 'recent' ? filterRecentSites(recentSites, search) : sites;

  function togglePreferLan() {
    setPreferLan((current) => {
      const next = !current;
      savePreferLan(next);
      return next;
    });
  }

  function togglePreview() {
    setPreviewEnabled((current) => {
      const next = !current;
      savePreviewEnabled(next);
      return next;
    });
  }

  function toggleTag(tagId: string) {
    setView('all');
    setSelectedTagIds((current) => {
      const next = current.includes(tagId) ? current.filter((item) => item !== tagId) : [...current, tagId];
      saveVisitorQueryState('all', next, tagMatch);
      return next;
    });
  }

  function switchView(nextView: ViewMode) {
    setView(nextView);
    saveVisitorQueryState(nextView, [], tagMatch);
    if (nextView !== 'all') {
      setSelectedTagIds([]);
    }
  }

  function switchTagMatch(nextMatch: TagMatchMode) {
    setTagMatch(nextMatch);
    saveVisitorQueryState(view, selectedTagIds, nextMatch);
  }

  function openSite(site: Site, method: OpenMethod = normalizeOpenMethod(site.open_method)) {
    const url = getSiteURL(site, preferLan);
    if (!url) {
      return;
    }
    setRecentSites(saveRecentSite(site));
    if (method === 'current_window') {
      window.location.assign(url);
      return;
    }
    window.open(url, '_blank', 'noopener,noreferrer');
  }

  async function copySiteURL(url: string) {
    if (!url) {
      return;
    }
    await copyText(url);
    setNotice(t('urlCopied'));
  }

  return (
    <main className="app-shell">
      <section className="content">
        <header className="toolbar">
          <div className="toolbar-top">
            <div className="search-box">
              <Search size={18} aria-hidden="true" />
              <input
                type="search"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t('searchPlaceholder')}
                aria-label={t('searchSites')}
              />
              {search && (
                <button className="icon-button" type="button" onClick={() => setSearch('')} title={t('clearSearch')}>
                  <X size={16} aria-hidden="true" />
                </button>
              )}
            </div>
            <div
              className={settingsOpen ? 'toolbar-actions open' : 'toolbar-actions'}
              ref={toolbarMenuRef}
            >
              <PreferenceControls />
              <button
                className="toolbar-menu-button"
                type="button"
                ref={toolbarMenuButtonRef}
                onClick={() => setSettingsOpen((current) => !current)}
                title={t('displaySettings')}
                aria-label={t('displaySettings')}
                aria-expanded={settingsOpen}
                aria-controls="visitor-toolbar-menu"
              >
                <MoreHorizontal size={18} aria-hidden="true" />
              </button>
              {settingsOpen && (
                <div className="toolbar-menu" id="visitor-toolbar-menu">
                  <label className={previewEnabled ? 'toolbar-menu-item toolbar-toggle preview-toggle active' : 'toolbar-menu-item toolbar-toggle preview-toggle'}>
                    <span>{t('preview')}</span>
                    <input type="checkbox" checked={previewEnabled} onChange={togglePreview} />
                    <span className="toggle-track" aria-hidden="true">
                      <span className="toggle-thumb" />
                    </span>
                  </label>
                  <label className={preferLan ? 'toolbar-menu-item toolbar-toggle lan-toggle active' : 'toolbar-menu-item toolbar-toggle lan-toggle'}>
                    <span>{t('preferLan')}</span>
                    <input type="checkbox" checked={preferLan} onChange={togglePreferLan} />
                    <span className="toggle-track" aria-hidden="true">
                      <span className="toggle-thumb" />
                    </span>
                  </label>
                  <a className="toolbar-menu-item admin-link" href="/admin">{t('admin')}</a>
                </div>
              )}
            </div>
          </div>

          <div className="filter-bar">
            <nav className="pill-nav" aria-label={t('tagFilter')}>
              <button
                className={view === 'all' && selectedTagIds.length === 0 ? 'filter-pill active' : 'filter-pill'}
                type="button"
                onClick={() => {
                  setSelectedTagIds([]);
                  setView('all');
                  saveVisitorQueryState('all', [], tagMatch);
                }}
              >
                {t('allTools')}
              </button>
              {tags.map((tag) => (
                <button
                  className={selectedTagSet.has(tag.id) ? 'filter-pill active' : 'filter-pill'}
                  type="button"
                  key={tag.id}
                  onClick={() => toggleTag(tag.id)}
                >
                  {tag.name}
                </button>
              ))}
              <button
                className={view === 'favorite' ? 'filter-pill active' : 'filter-pill'}
                type="button"
                onClick={() => switchView('favorite')}
              >
                {t('favoriteTools')}
              </button>
              <button
                className={view === 'recent' ? 'filter-pill active' : 'filter-pill'}
                type="button"
                onClick={() => switchView('recent')}
              >
                {t('recentVisits')}
              </button>
            </nav>
            {view === 'all' && selectedTagIds.length > 1 && (
              <div className="tag-match-toggle" role="group" aria-label={t('tagMatchMode')}>
                <button
                  className={tagMatch === 'all' ? 'active' : undefined}
                  type="button"
                  onClick={() => switchTagMatch('all')}
                >
                  {t('matchAllTags')}
                </button>
                <button
                  className={tagMatch === 'any' ? 'active' : undefined}
                  type="button"
                  onClick={() => switchTagMatch('any')}
                >
                  {t('matchAnyTag')}
                </button>
              </div>
            )}
          </div>
        </header>

        <div className="visitor-section-head">
          <h1>{currentTitle}</h1>
        </div>

        <section className="site-area" aria-live="polite">
          {loading && (
            <div className="state">
              <Loader2 className="spin" size={24} aria-hidden="true" />
              <span>{t('loading')}</span>
            </div>
          )}

          {!loading && error && (
            <div className="state error-state">
              <span>{error}</span>
            </div>
          )}

          {!loading && !error && displaySites.length === 0 && (
            <div className="state">
              <span>{t('noMatchingSites')}</span>
            </div>
          )}

          {!loading && !error && displaySites.length > 0 && (
            <div className="site-grid">
              {displaySites.map((site) => (
                <SiteCard
                  key={site.id}
                  site={site}
                  preferLan={preferLan}
                  previewEnabled={previewEnabled}
                  onCopyURL={copySiteURL}
                  onOpen={openSite}
                />
              ))}
            </div>
          )}
        </section>
      </section>
      <VersionBadge />
      {notice && <div className="toast">{notice}</div>}
    </main>
  );
}

function SiteCard({
  site,
  preferLan,
  previewEnabled,
  onCopyURL,
  onOpen
}: {
  site: Site;
  preferLan: boolean;
  previewEnabled: boolean;
  onCopyURL: (url: string) => void;
  onOpen: (site: Site, method?: OpenMethod) => void;
}) {
  const { t } = usePreferences();
  const openMethod = normalizeOpenMethod(site.open_method);
  const opensInNewTab = openMethod === 'new_window';
  const hasLanURL = site.lan_url.trim() !== '';
  const usingLanURL = preferLan && hasLanURL;
  const previewTimer = useRef<number | null>(null);
  const menuButtonRef = useRef<HTMLButtonElement | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [menuPosition, setMenuPosition] = useState<{ x: number; y: number } | null>(null);
  const previewURL = getSiteURL(site, preferLan);
  const canPreview = previewEnabled && isPreviewableURL(previewURL);

  useEffect(() => {
    return () => {
      if (previewTimer.current !== null) {
        window.clearTimeout(previewTimer.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!previewEnabled) {
      hideSitePreview();
    }
  }, [previewEnabled]);

  useEffect(() => {
    if (!menuPosition) {
      return;
    }

    function closeOnOutside(event: PointerEvent) {
      const target = event.target;
      if (!(target instanceof Node)) {
        closeMenu();
        return;
      }
      if (menuRef.current?.contains(target) || menuButtonRef.current?.contains(target)) {
        return;
      }
      closeMenu();
    }

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        closeMenu();
      }
    }

    window.addEventListener('pointerdown', closeOnOutside, true);
    window.addEventListener('keydown', closeOnEscape);
    window.addEventListener('scroll', closeMenu, true);
    return () => {
      window.removeEventListener('pointerdown', closeOnOutside, true);
      window.removeEventListener('keydown', closeOnEscape);
      window.removeEventListener('scroll', closeMenu, true);
    };
  }, [menuPosition]);

  function showSitePreview() {
    if (!canPreview) {
      return;
    }
    previewTimer.current = window.setTimeout(() => {
      setShowPreview(true);
    }, 350);
  }

  function hideSitePreview() {
    if (previewTimer.current !== null) {
      window.clearTimeout(previewTimer.current);
      previewTimer.current = null;
    }
    setShowPreview(false);
  }

  function closeMenu() {
    setMenuPosition(null);
  }

  function openMenuAt(x: number, y: number) {
    hideSitePreview();
    setMenuPosition(clampMenuPosition(x, y));
  }

  function openContextMenu(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    openMenuAt(event.clientX, event.clientY);
  }

  function openButtonMenu(event: MouseEvent<HTMLButtonElement>) {
    event.preventDefault();
    event.stopPropagation();
    const rect = event.currentTarget.getBoundingClientRect();
    openMenuAt(rect.right - siteMenuWidth, rect.bottom + 8);
  }

  function openWith(method: OpenMethod) {
    closeMenu();
    onOpen(site, method);
  }

  function copyURL(url: string) {
    closeMenu();
    onCopyURL(url);
  }

  return (
    <div className="site-card-wrap" onContextMenu={openContextMenu} onMouseEnter={showSitePreview} onMouseLeave={hideSitePreview}>
      <button className="site-card" type="button" onClick={() => onOpen(site)}>
        <div className="site-main">
          <SiteIcon site={site} />
          <div className="site-copy">
            <div className="site-title-row">
              <h2>{site.title}</h2>
              {site.is_favorite && <Heart className="favorite-icon" size={16} aria-label={t('favorite')} />}
            </div>
            {!site.only_name && site.description && <p>{site.description}</p>}
          </div>
          <div className={hasLanURL ? 'site-card-side' : 'site-card-side no-lan'}>
            {hasLanURL && (
              <span
                className={usingLanURL ? 'site-network-badge active' : 'site-network-badge'}
                title={usingLanURL ? t('usingLanURL') : t('hasLanURL')}
              >
                LAN
              </span>
            )}
            {opensInNewTab ? (
              <span className="open-method-icon" title={t('openNewTab')} aria-label={t('openNewTab')} role="img">
                <ExternalLink size={15} aria-hidden="true" />
              </span>
            ) : (
              <span className="open-method-icon" title={t('openCurrentTab')} aria-label={t('openCurrentTab')} role="img">
                <ArrowRight size={15} aria-hidden="true" />
              </span>
            )}
          </div>
        </div>
      </button>
      <button
        className="site-menu-button"
        type="button"
        ref={menuButtonRef}
        onClick={openButtonMenu}
        title={t('siteMoreActions')}
        aria-label={t('siteMoreActions')}
      >
        <MoreHorizontal size={16} aria-hidden="true" />
      </button>
      {menuPosition && (
        <div
          className="site-context-menu"
          ref={menuRef}
          style={{ left: menuPosition.x, top: menuPosition.y }}
          role="menu"
        >
          <button type="button" role="menuitem" onClick={() => openWith('new_window')}>
            <ExternalLink size={15} aria-hidden="true" />
            <span>{t('openNewTab')}</span>
          </button>
          <button type="button" role="menuitem" onClick={() => openWith('current_window')}>
            <ArrowRight size={15} aria-hidden="true" />
            <span>{t('openCurrentTab')}</span>
          </button>
          <div className="site-menu-separator" />
          <button type="button" role="menuitem" onClick={() => copyURL(previewURL)}>
            <Copy size={15} aria-hidden="true" />
            <span>{t('copyCurrentURL')}</span>
          </button>
          <button type="button" role="menuitem" onClick={() => copyURL(site.default_url)}>
            <Copy size={15} aria-hidden="true" />
            <span>{t('copyDefaultURL')}</span>
          </button>
          {hasLanURL && (
            <button type="button" role="menuitem" onClick={() => copyURL(site.lan_url)}>
              <Copy size={15} aria-hidden="true" />
              <span>{t('copyLanURL')}</span>
            </button>
          )}
        </div>
      )}
      {showPreview && (
        <div className="site-preview" aria-hidden="true">
          <iframe
            src={previewURL}
            title={site.title}
            loading="lazy"
            referrerPolicy="no-referrer"
            sandbox="allow-forms allow-same-origin allow-scripts"
          />
        </div>
      )}
    </div>
  );
}

function SiteIcon({ site }: { site: Site }) {
  if (site.icon_value && (site.icon_type === 'image' || site.icon_value.startsWith('/uploads/'))) {
    return (
      <div className="site-icon image-icon">
        <img src={site.icon_value} alt="" />
      </div>
    );
  }

  const text = site.icon_value || site.title;
  return <div className="site-icon text-icon">{text.trim().slice(0, 2).toUpperCase()}</div>;
}

function filterRecentSites(sites: RecentSite[], search: string): RecentSite[] {
  const keyword = search.trim().toLowerCase();
  if (!keyword) {
    return sites;
  }
  return sites.filter((site) => {
    const tagText = site.tags.map((tag) => tag.name).join(' ');
    return [site.title, site.description, site.default_url, site.lan_url, tagText]
      .join(' ')
      .toLowerCase()
      .includes(keyword);
  });
}

function getSiteURL(site: Site, preferLan: boolean): string {
  if (preferLan && site.lan_url.trim()) {
    return site.lan_url;
  }
  return site.default_url;
}

function isPreviewableURL(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch {
    return false;
  }
}

const siteMenuWidth = 210;
const siteMenuHeight = 232;

function normalizeOpenMethod(value: string): OpenMethod {
  return value === 'current_window' ? 'current_window' : 'new_window';
}

function clampMenuPosition(x: number, y: number): { x: number; y: number } {
  const maxX = Math.max(8, window.innerWidth - siteMenuWidth - 8);
  const maxY = Math.max(8, window.innerHeight - siteMenuHeight - 8);
  return {
    x: Math.min(Math.max(8, x), maxX),
    y: Math.min(Math.max(8, y), maxY)
  };
}

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch {
      // 浏览器拒绝 Clipboard API 时回退到传统复制方式。
    }
  }

  const input = document.createElement('textarea');
  input.value = value;
  input.style.position = 'fixed';
  input.style.left = '-9999px';
  document.body.appendChild(input);
  try {
    input.focus();
    input.select();
    document.execCommand('copy');
  } finally {
    document.body.removeChild(input);
  }
}

function loadVisitorQueryState(
  tags: Tag[]
): { hasQueryState: boolean; tagIds: string[]; tagMatch: TagMatchMode; view: ViewMode } {
  const params = new URLSearchParams(window.location.search);
  const queryView = params.get(viewQueryKey);
  const view = isViewMode(queryView) ? queryView : 'all';
  const queryTagMatch = params.get(tagMatchQueryKey);
  const tagMatch = isTagMatchMode(queryTagMatch) ? queryTagMatch : 'all';
  const tagSet = new Set(tags.map((tag) => tag.id));
  const tagIds = (params.get(tagQueryKey) || '')
    .split(',')
    .map((tagId) => tagId.trim())
    .filter((tagId) => tagSet.has(tagId));

  return {
    hasQueryState: params.has(tagQueryKey) || params.has(tagMatchQueryKey) || params.has(viewQueryKey),
    tagIds,
    tagMatch,
    view
  };
}

function saveVisitorQueryState(view: ViewMode, tagIds: string[], tagMatch: TagMatchMode) {
  const url = new URL(window.location.href);
  url.searchParams.delete(tagQueryKey);
  url.searchParams.delete(tagMatchQueryKey);
  url.searchParams.delete(viewQueryKey);

  if (view === 'all' && tagIds.length > 0) {
    url.searchParams.set(tagQueryKey, tagIds.join(','));
    if (tagMatch === 'any') {
      url.searchParams.set(tagMatchQueryKey, tagMatch);
    }
  } else {
    url.searchParams.set(viewQueryKey, view);
  }

  window.history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
}

function isViewMode(view: string | null): view is ViewMode {
  return view === 'all' || view === 'favorite' || view === 'recent';
}

function isTagMatchMode(tagMatch: string | null): tagMatch is TagMatchMode {
  return tagMatch === 'all' || tagMatch === 'any';
}

function loadPreferLan(): boolean {
  try {
    return window.localStorage.getItem(preferLanKey) === 'true';
  } catch {
    return false;
  }
}

function savePreferLan(value: boolean) {
  try {
    window.localStorage.setItem(preferLanKey, String(value));
  } catch {
    // localStorage 不可用时只保留当前会话状态。
  }
}

function loadPreviewEnabled(): boolean {
  try {
    return window.localStorage.getItem(previewEnabledKey) !== 'false';
  } catch {
    return true;
  }
}

function savePreviewEnabled(value: boolean) {
  try {
    window.localStorage.setItem(previewEnabledKey, String(value));
  } catch {
    // localStorage 不可用时只保留当前会话状态。
  }
}
