import {
  ArrowRight,
  ExternalLink,
  Heart,
  Loader2,
  Search,
  X
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { getPublicConfig, listSites, listTags } from './api';
import { AdminApp } from './AdminApp';
import { PreferenceControls, PreferencesProvider, usePreferences } from './preferences';
import { loadRecentSites, saveRecentSite } from './recent';
import './styles.css';
import type { RecentSite, Site, Tag } from './types';

type ViewMode = 'all' | 'favorite' | 'recent';

const preferLanKey = 'navbox_prefer_lan';
const tagQueryKey = 'tags';
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
  const [search, setSearch] = useState('');
  const [view, setView] = useState<ViewMode>('all');
  const [preferLan, setPreferLan] = useState(() => loadPreferLan());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadInitialData() {
      setLoading(true);
      setError('');
      try {
        const [tagList, config] = await Promise.all([listTags(), getPublicConfig()]);
        if (cancelled) {
          return;
        }
        const enabledTags = tagList.filter((tag) => tag.is_enabled);
        const defaultTag = enabledTags.find((tag) => tag.id === config.default_tag_id);
        const queryState = loadVisitorQueryState(enabledTags);
        setTags(enabledTags);
        if (queryState.view !== 'all') {
          setView(queryState.view);
          setSelectedTagIds([]);
        } else if (queryState.hasQueryState) {
          setSelectedTagIds(queryState.tagIds);
        } else if (defaultTag) {
          setSelectedTagIds([defaultTag.id]);
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
  }, [ready, search, selectedTagIds, t, view]);

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

  function toggleTag(tagId: string) {
    setView('all');
    setSelectedTagIds((current) => {
      const next = current.includes(tagId) ? current.filter((item) => item !== tagId) : [...current, tagId];
      saveVisitorQueryState('all', next);
      return next;
    });
  }

  function switchView(nextView: ViewMode) {
    setView(nextView);
    saveVisitorQueryState(nextView, []);
    if (nextView !== 'all') {
      setSelectedTagIds([]);
    }
  }

  function openSite(site: Site) {
    const url = getSiteURL(site, preferLan);
    if (!url) {
      return;
    }
    setRecentSites(saveRecentSite(site));
    if (site.open_method === 'current_window') {
      window.location.assign(url);
      return;
    }
    window.open(url, '_blank', 'noopener,noreferrer');
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
            <div className="toolbar-actions">
              <PreferenceControls />
              <label className={preferLan ? 'lan-toggle active' : 'lan-toggle'}>
                <input type="checkbox" checked={preferLan} onChange={togglePreferLan} />
                <span className="toggle-track" aria-hidden="true">
                  <span className="toggle-thumb" />
                </span>
                <span>{t('preferLan')}</span>
              </label>
              <a className="admin-link" href="/admin">{t('admin')}</a>
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
                  saveVisitorQueryState('all', []);
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
                <SiteCard key={site.id} site={site} preferLan={preferLan} onOpen={openSite} />
              ))}
            </div>
          )}
        </section>
      </section>
    </main>
  );
}

function SiteCard({
  site,
  preferLan,
  onOpen
}: {
  site: Site;
  preferLan: boolean;
  onOpen: (site: Site) => void;
}) {
  const { t } = usePreferences();
  const opensInNewTab = site.open_method === 'new_window';
  const hasLanURL = site.lan_url.trim() !== '';
  const usingLanURL = preferLan && hasLanURL;

  return (
    <button className="site-card" type="button" onClick={() => onOpen(site)}>
      <div className="site-main">
        <SiteIcon site={site} />
        <div className="site-copy">
          <div className="site-title-row">
            <h2>{site.title}</h2>
            {hasLanURL && (
              <span
                className={usingLanURL ? 'site-network-badge active' : 'site-network-badge'}
                title={usingLanURL ? t('usingLanURL') : t('hasLanURL')}
              >
                LAN
              </span>
            )}
          </div>
          {!site.only_name && site.description && <p>{site.description}</p>}
        </div>
        <div className="site-card-tools">
          {opensInNewTab ? (
            <span className="open-method-icon" title={t('openNewTab')} aria-label={t('openNewTab')} role="img">
              <ExternalLink size={15} aria-hidden="true" />
            </span>
          ) : (
            <span className="open-method-icon" title={t('openCurrentTab')} aria-label={t('openCurrentTab')} role="img">
              <ArrowRight size={15} aria-hidden="true" />
            </span>
          )}
          {site.is_favorite && <Heart className="favorite-icon" size={16} aria-label={t('favorite')} />}
        </div>
      </div>
    </button>
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

function loadVisitorQueryState(tags: Tag[]): { hasQueryState: boolean; tagIds: string[]; view: ViewMode } {
  const params = new URLSearchParams(window.location.search);
  const queryView = params.get(viewQueryKey);
  const view = isViewMode(queryView) ? queryView : 'all';
  const tagSet = new Set(tags.map((tag) => tag.id));
  const tagIds = (params.get(tagQueryKey) || '')
    .split(',')
    .map((tagId) => tagId.trim())
    .filter((tagId) => tagSet.has(tagId));

  return {
    hasQueryState: params.has(tagQueryKey) || params.has(viewQueryKey),
    tagIds,
    view
  };
}

function saveVisitorQueryState(view: ViewMode, tagIds: string[]) {
  const url = new URL(window.location.href);
  url.searchParams.delete(tagQueryKey);
  url.searchParams.delete(viewQueryKey);

  if (view === 'all' && tagIds.length > 0) {
    url.searchParams.set(tagQueryKey, tagIds.join(','));
  } else {
    url.searchParams.set(viewQueryKey, view);
  }

  window.history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
}

function isViewMode(view: string | null): view is ViewMode {
  return view === 'all' || view === 'favorite' || view === 'recent';
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
