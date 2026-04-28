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
import { loadRecentSites, saveRecentSite } from './recent';
import './styles.css';
import type { RecentSite, Site, Tag } from './types';

type ViewMode = 'all' | 'favorite' | 'recent';

const preferLanKey = 'navbox_prefer_lan';

export function App() {
  if (window.location.pathname.startsWith('/admin')) {
    return <AdminApp />;
  }

  return <VisitorApp />;
}

function VisitorApp() {
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
        setTags(enabledTags);
        if (defaultTag) {
          setSelectedTagIds([defaultTag.id]);
        }
        setReady(true);
      } catch {
        if (!cancelled) {
          setError('首页数据加载失败');
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
  }, []);

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
          setError('网站列表加载失败');
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
  }, [ready, search, selectedTagIds, view]);

  const selectedTagSet = useMemo(() => new Set(selectedTagIds), [selectedTagIds]);
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
    setSelectedTagIds((current) =>
      current.includes(tagId) ? current.filter((item) => item !== tagId) : [...current, tagId]
    );
  }

  function switchView(nextView: ViewMode) {
    setView(nextView);
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
          <div className="search-box">
            <Search size={18} aria-hidden="true" />
            <input
              type="search"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="按回车键聚焦并搜索应用"
              aria-label="搜索网站"
            />
            {search && (
              <button className="icon-button" type="button" onClick={() => setSearch('')} title="清空搜索">
                <X size={16} aria-hidden="true" />
              </button>
            )}
          </div>

          <div className="filter-bar">
            <nav className="pill-nav" aria-label="Tag 筛选">
              <button
                className={view === 'all' && selectedTagIds.length === 0 ? 'filter-pill active' : 'filter-pill'}
                type="button"
                onClick={() => {
                  setSelectedTagIds([]);
                  setView('all');
                }}
              >
                全部工具
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
                常用工具
              </button>
              <button
                className={view === 'recent' ? 'filter-pill active' : 'filter-pill'}
                type="button"
                onClick={() => switchView('recent')}
              >
                最近访问
              </button>
              <a className="filter-pill admin-pill" href="/admin">管理后台</a>
            </nav>
            <label className={preferLan ? 'lan-toggle active' : 'lan-toggle'}>
              <input type="checkbox" checked={preferLan} onChange={togglePreferLan} />
              <span className="toggle-track" aria-hidden="true">
                <span className="toggle-thumb" />
              </span>
              <span>内网优先</span>
            </label>
          </div>
        </header>

        <section className="site-area" aria-live="polite">
          {loading && (
            <div className="state">
              <Loader2 className="spin" size={24} aria-hidden="true" />
              <span>加载中</span>
            </div>
          )}

          {!loading && error && (
            <div className="state error-state">
              <span>{error}</span>
            </div>
          )}

          {!loading && !error && displaySites.length === 0 && (
            <div className="state">
              <span>没有匹配的网站</span>
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
  const primaryTag = site.tags[0];
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
          </div>
          {!site.only_name && site.description && <p>{site.description}</p>}
          <div className="site-meta-row">
            {primaryTag && <span className="site-tag">{primaryTag.name}</span>}
            {hasLanURL && (
              <span
                className={usingLanURL ? 'site-tag lan-tag active' : 'site-tag lan-tag'}
                title={usingLanURL ? '当前使用 LAN URL' : '已配置 LAN URL'}
              >
                LAN
              </span>
            )}
          </div>
        </div>
        <div className="site-card-tools">
          {opensInNewTab ? (
            <span className="open-method-icon" title="新标签页打开" aria-label="新标签页打开" role="img">
              <ExternalLink size={15} aria-hidden="true" />
            </span>
          ) : (
            <span className="open-method-icon" title="当前页打开" aria-label="当前页打开" role="img">
              <ArrowRight size={15} aria-hidden="true" />
            </span>
          )}
          {site.is_favorite && <Heart className="favorite-icon" size={16} aria-label="常用" />}
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
