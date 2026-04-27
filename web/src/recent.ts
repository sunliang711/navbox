import type { RecentSite, Site } from './types';

const recentKey = 'navbox_recent_sites';
const maxRecentSites = 24;

export function loadRecentSites(): RecentSite[] {
  try {
    const raw = window.localStorage.getItem(recentKey);
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw) as RecentSite[];
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter((item) => item && typeof item.id === 'string');
  } catch {
    return [];
  }
}

export function saveRecentSite(site: Site): RecentSite[] {
  const next: RecentSite = {
    ...site,
    visited_at: new Date().toISOString()
  };
  const items = [next, ...loadRecentSites().filter((item) => item.id !== site.id)].slice(0, maxRecentSites);
  window.localStorage.setItem(recentKey, JSON.stringify(items));
  return items;
}
