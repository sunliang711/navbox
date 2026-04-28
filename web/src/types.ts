export type ApiResponse<T> = {
  code: number;
  data: T;
  message: string;
};

export type Tag = {
  id: string;
  name: string;
  icon: string;
  color: string;
  sort_order: number;
  is_default: boolean;
  is_enabled: boolean;
  site_count: number;
  created_at: string;
  updated_at: string;
};

export type Site = {
  id: string;
  title: string;
  description: string;
  default_url: string;
  lan_url: string;
  open_method: string;
  icon_type: string;
  icon_value: string;
  background_color: string;
  only_name: boolean;
  is_favorite: boolean;
  sort_order: number;
  tags: Tag[];
  created_at: string;
  updated_at: string;
};

export type PublicConfig = {
  default_tag_id: string;
};

export type TagMatchMode = 'all' | 'any';

export type SiteQuery = {
  search?: string;
  tagIds?: string[];
  tagMatch?: TagMatchMode;
  view?: string;
};

export type RecentSite = Site & {
  visited_at: string;
};

export type SessionResp = {
  authenticated: boolean;
};

export type RestoreStatus = {
  enabled: boolean;
  mode: string;
};

export type RestorePasswordResp = {
  restored: boolean;
};

export type SiteSaveReq = {
  title: string;
  description: string;
  default_url: string;
  lan_url: string;
  open_method: string;
  icon_type: string;
  icon_value: string;
  background_color: string;
  only_name: boolean;
  is_favorite: boolean;
  sort_order: number;
  tag_ids: string[];
};

export type TagSaveReq = {
  name: string;
  icon: string;
  color: string;
  sort_order: number;
  is_enabled: boolean;
};

export type BatchTagAction = 'add' | 'remove';

export type OrderItem = {
  id: string;
  sort_order: number;
};

export type IconResp = {
  id: string;
  file_name: string;
  file_path: string;
  url: string;
  sha256: string;
  mime_type: string;
  size_bytes: number;
  created_at: string;
};

export type ImportReport = {
  imported: ImportCount;
  skipped: ImportCount;
  conflicts: ImportConflict[];
};

export type ImportCount = {
  sites: number;
  tags: number;
  icons: number;
  relations: number;
};

export type ImportConflict = {
  type: string;
  id?: string;
  name?: string;
  reason: string;
};

export type ExportFile = {
  blob: Blob;
  fileName: string;
};
