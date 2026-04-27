import {
  Clock,
  Copy,
  ExternalLink,
  Globe2,
  Heart,
  Layers3,
  Link2,
  Loader2,
  Monitor,
  Network,
  Search,
  Tag as TagIcon,
  X
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { getPublicConfig, listSites, listTags } from './api';
import { AdminApp } from './AdminApp';
import { loadRecentSites, saveRecentSite } from './recent';
import './styles.css';
import type { RecentSite, Site, Tag } from './types';

type ViewMode = 'all' | 'favorite' | 'recent' | 'uncategorized';

const viewItems: Array<{ id: ViewMode; label: string; icon: typeof Globe2 }> = [
  { id: 'all', label: '全部', icon: Globe2 },
  { id: 'favorite', label: '常用', icon: Heart },
  { id: 'recent', label: '最近', icon: Clock },
  { id: 'uncategorized', label: '未分类', icon: Layers3 }
];

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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
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

  useEffect(() => {
    if (!notice) {
      return;
    }
    const timer = window.setTimeout(() => setNotice(''), 1800);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const selectedTagSet = useMemo(() => new Set(selectedTagIds), [selectedTagIds]);
  const displaySites = view === 'recent' ? filterRecentSites(recentSites, search) : sites;
  const activeFilterCount = selectedTagIds.length + (search.trim() ? 1 : 0) + (view !== 'all' ? 1 : 0);

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

  function clearFilters() {
    setSearch('');
    setSelectedTagIds([]);
    setView('all');
  }

  function openSite(site: Site, url: string) {
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

  async function copyURL(site: Site, url: string) {
    if (!url) {
      return;
    }
    await copyText(url);
    setRecentSites(saveRecentSite(site));
    setNotice('链接已复制');
  }

  return (
    <main className="app-shell">
      <aside className="sidebar" aria-label="导航筛选">
        <div className="brand">
          <div className="brand-icon">
            <Monitor size={24} aria-hidden="true" />
          </div>
          <div>
            <h1>Navbox</h1>
            <span>{tags.length} Tags</span>
          </div>
        </div>

        <nav className="view-nav" aria-label="系统视图">
          {viewItems.map((item) => {
            const Icon = item.icon;
            return (
              <button
                className={view === item.id ? 'nav-button active' : 'nav-button'}
                type="button"
                key={item.id}
                onClick={() => switchView(item.id)}
              >
                <Icon size={17} aria-hidden="true" />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        <section className="tag-panel" aria-label="Tag 列表">
          <div className="panel-title">
            <TagIcon size={16} aria-hidden="true" />
            <span>Tags</span>
          </div>
          <div className="tag-list">
            {tags.map((tag) => (
              <button
                className={selectedTagSet.has(tag.id) ? 'tag-button selected' : 'tag-button'}
                type="button"
                key={tag.id}
                onClick={() => toggleTag(tag.id)}
                style={{ '--tag-color': normalizeColor(tag.color) } as React.CSSProperties}
              >
                <span className="tag-dot" aria-hidden="true" />
                <span className="tag-name">{tag.name}</span>
                <span className="tag-count">{tag.site_count}</span>
              </button>
            ))}
          </div>
        </section>
      </aside>

      <section className="content">
        <header className="toolbar">
          <div className="search-box">
            <Search size={18} aria-hidden="true" />
            <input
              type="search"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="搜索网站、描述、URL 或 Tag"
              aria-label="搜索网站"
            />
            {search && (
              <button className="icon-button" type="button" onClick={() => setSearch('')} title="清空搜索">
                <X size={16} aria-hidden="true" />
              </button>
            )}
          </div>

          <div className="toolbar-summary">
            <span>{displaySites.length} 个网站</span>
            {activeFilterCount > 0 && (
              <button className="text-button" type="button" onClick={clearFilters}>
                清除筛选
              </button>
            )}
          </div>
        </header>

        {notice && <div className="toast">{notice}</div>}

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
                <SiteCard key={site.id} site={site} onOpen={openSite} onCopy={copyURL} />
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
  onOpen,
  onCopy
}: {
  site: Site;
  onOpen: (site: Site, url: string) => void;
  onCopy: (site: Site, url: string) => Promise<void>;
}) {
  const visibleTags = site.tags.slice(0, 2);
  const moreTags = Math.max(site.tags.length - visibleTags.length, 0);
  const background = normalizeColor(site.background_color);

  return (
    <article className="site-card" style={{ '--card-accent': background } as React.CSSProperties}>
      <div className="site-main">
        <SiteIcon site={site} />
        <div className="site-copy">
          <div className="site-title-row">
            <h2>{site.title}</h2>
            {site.is_favorite && <Heart className="favorite-icon" size={16} aria-label="常用" />}
          </div>
          {!site.only_name && site.description && <p>{site.description}</p>}
          <div className="site-url">
            <Link2 size={14} aria-hidden="true" />
            <span>{site.default_url}</span>
          </div>
        </div>
      </div>

      <div className="site-tags" title={site.tags.map((tag) => tag.name).join(', ')}>
        {visibleTags.map((tag) => (
          <span className="site-tag" key={tag.id}>
            {tag.name}
          </span>
        ))}
        {moreTags > 0 && <span className="site-tag">+{moreTags}</span>}
      </div>

      <div className="site-actions">
        <button type="button" onClick={() => onOpen(site, site.default_url)} title="打开默认 URL">
          <ExternalLink size={17} aria-hidden="true" />
          <span>打开</span>
        </button>
        {site.lan_url && (
          <button type="button" onClick={() => onOpen(site, site.lan_url)} title="打开 LAN URL">
            <Network size={17} aria-hidden="true" />
            <span>LAN</span>
          </button>
        )}
        <button type="button" onClick={() => onCopy(site, site.default_url)} title="复制默认 URL">
          <Copy size={17} aria-hidden="true" />
          <span>复制</span>
        </button>
      </div>
    </article>
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

function normalizeColor(value: string): string {
  const color = value.trim();
  if (!color) {
    return '#2f7d6d';
  }
  return color;
}

async function copyText(value: string): Promise<void> {
  if (navigator.clipboard) {
    await navigator.clipboard.writeText(value);
    return;
  }

  const input = document.createElement('textarea');
  input.value = value;
  input.setAttribute('readonly', 'true');
  input.style.position = 'fixed';
  input.style.left = '-9999px';
  document.body.appendChild(input);
  input.select();
  document.execCommand('copy');
  document.body.removeChild(input);
}
