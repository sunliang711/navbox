const version = import.meta.env.VITE_NAVBOX_VERSION || 'dev';

export function VersionBadge() {
  return <span className="app-version">{version}</span>;
}
