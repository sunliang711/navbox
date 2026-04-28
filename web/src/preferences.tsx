import { Languages, Moon, Sun } from 'lucide-react';
import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from 'react';

export type ThemeMode = 'light' | 'dark';
export type LanguageMode = 'zh' | 'en';

const themeKey = 'navbox_theme';
const languageKey = 'navbox_language';

const zh = {
  allTools: '全部工具',
  favoriteTools: '常用工具',
  recentVisits: '最近访问',
  selectedCategoryCount: '已选 {{count}} 个分类',
  searchPlaceholder: '请输入关键词',
  searchSites: '搜索网站',
  clearSearch: '清空搜索',
  preferLan: '内网优先',
  admin: '管理后台',
  visitorHome: '游客首页',
  tagFilter: 'Tag 筛选',
  loading: '加载中',
  noMatchingSites: '没有匹配的网站',
  homeLoadFailed: '首页数据加载失败',
  siteListLoadFailed: '网站列表加载失败',
  usingLanURL: '当前使用 LAN URL',
  hasLanURL: '已配置 LAN URL',
  openNewTab: '新标签页打开',
  openCurrentTab: '当前页打开',
  favorite: '常用',
  lightMode: '浅色',
  darkMode: '深色',
  language: '语言',
  adminLogin: 'Navbox 管理后台',
  password: '密码',
  login: '登录',
  loginFailed: '密码错误或 Session 创建失败',
  checkingSession: '正在检查 Session',
  refreshing: '正在刷新数据',
  restorePasswordTitle: '恢复管理员密码',
  restoreModeNote: '恢复模式已启用。输入启动时配置的 Restore Token，并设置新的管理员密码。',
  restoreToken: 'Restore Token',
  newPassword: '新密码',
  confirmNewPassword: '确认新密码',
  resetPassword: '重置密码',
  restoreDoneNote: '重置成功后需要删除恢复配置并重启服务，正常管理接口才会重新开放。',
  restoreSuccess: '管理员密码已重置。请关闭恢复模式并重启服务后登录。',
  restoreTokenInvalid: 'Restore Token 无效、已过期或已使用',
  passwordMin: '新密码至少 8 位',
  passwordMismatch: '两次输入的新密码不一致',
  navboxAdmin: 'Navbox 管理后台',
  adminStats: '{{sites}} 个网站，{{tags}} 个 Tag',
  logout: '退出',
  adminModules: 'admin 管理模块',
  sitesTab: '网站',
  tagsTab: 'Tag',
  importExportTab: '导入导出',
  passwordTab: '密码',
  newSite: '新建网站',
  saveOrder: '保存排序',
  batchDelete: '批量删除',
  drag: '拖拽',
  select: '选择',
  site: '网站',
  openMethod: '打开方式',
  favoriteColumn: '常用',
  sort: '排序',
  actions: '操作',
  dragSort: '拖拽排序',
  selectSite: '选择 {{title}}',
  uncategorized: '未分类',
  yes: '是',
  no: '否',
  editSite: '编辑网站',
  deleteSite: '删除网站',
  newTag: '新建 Tag',
  status: '状态',
  siteCount: '网站数',
  noIcon: '无图标',
  defaultStatus: '默认',
  enabledStatus: '启用',
  disabledStatus: '停用',
  setDefault: '设为默认',
  editTag: '编辑 Tag',
  deleteTag: '删除 Tag',
  export: '导出',
  exportAll: '导出全部',
  exportSelectedSites: '导出选中网站',
  exportSelectedTags: '导出选中 Tag',
  import: '导入',
  chooseZipImport: '选择 zip 导入',
  importResult: '导入结果',
  importedSummary: '新增网站 {{sites}}，Tag {{tags}}，icon {{icons}}',
  skippedSummary: '跳过网站 {{sites}}，Tag {{tags}}，icon {{icons}}',
  currentPassword: '当前密码',
  changePassword: '修改密码',
  editSiteTitle: '编辑网站',
  newSiteTitle: '新建网站',
  editTagTitle: '编辑 Tag',
  newTagTitle: '新建 Tag',
  batchAddTag: '批量添加 Tag',
  batchRemoveTag: '批量移除 Tag',
  close: '关闭',
  title: '标题',
  defaultURL: '默认 URL',
  lanURL: 'LAN URL',
  description: '描述',
  newWindow: '新窗口',
  currentWindow: '当前窗口',
  iconType: '图标类型',
  textIcon: '文本',
  imageIcon: '图片',
  onlineIcon: '在线',
  backgroundColor: '背景色',
  iconValue: '图标值',
  fetching: '获取中',
  fetchFromWebsite: '从网站获取',
  uploadIcon: '上传 icon',
  onlyName: '仅显示名称',
  saveSite: '保存网站',
  name: '名称',
  icon: '图标',
  color: '颜色',
  enabled: '启用',
  saveTag: '保存 Tag',
  currentTab: '当前 Tab',
  newTab: '新 Tab',
  siteUpdated: '网站已更新',
  siteCreated: '网站已创建',
  confirmDeleteSite: '确认删除这个网站？',
  siteDeleted: '网站已删除',
  iconUploaded: 'icon 已上传',
  fillDefaultURL: '请先填写默认 URL',
  iconFetched: 'icon 已获取',
  iconFetchFailed: '未获取到 icon',
  tagUpdated: 'Tag 已更新',
  tagCreated: 'Tag 已创建',
  confirmDeleteTag: '确认删除这个 Tag？关联网站不会被删除。',
  tagDeleted: 'Tag 已删除',
  defaultTagUpdated: '默认 Tag 已更新',
  siteOrderSaved: '网站排序已保存',
  tagOrderSaved: 'Tag 排序已保存',
  siteOrderUpdated: '网站排序已更新',
  siteOrderUpdateFailed: '网站排序更新失败',
  tagOrderUpdated: 'Tag 排序已更新',
  tagOrderUpdateFailed: 'Tag 排序更新失败',
  confirmBatchDelete: '确认删除选中的 {{count}} 个网站？',
  batchDeleteDone: '批量删除已完成',
  chooseSitesAndTags: '请选择网站和 Tag',
  batchAddTagDone: '批量添加 Tag 已完成',
  batchRemoveTagDone: '批量移除 Tag 已完成',
  chooseSites: '请先选择网站',
  chooseTags: '请先选择 Tag',
  exportStarted: '导出已开始',
  importDone: '导入已完成',
  passwordChanged: '密码已修改',
  sessionExpired: '登录状态已过期，请重新登录'
} as const;

export type TranslationKey = keyof typeof zh;

const en: Record<TranslationKey, string> = {
  allTools: 'All Tools',
  favoriteTools: 'Favorites',
  recentVisits: 'Recent',
  selectedCategoryCount: '{{count}} categories selected',
  searchPlaceholder: 'Enter keywords',
  searchSites: 'Search sites',
  clearSearch: 'Clear search',
  preferLan: 'Prefer LAN',
  admin: 'Admin',
  visitorHome: 'Home',
  tagFilter: 'Tag filter',
  loading: 'Loading',
  noMatchingSites: 'No matching sites',
  homeLoadFailed: 'Failed to load home data',
  siteListLoadFailed: 'Failed to load sites',
  usingLanURL: 'Using LAN URL',
  hasLanURL: 'LAN URL configured',
  openNewTab: 'Open in new tab',
  openCurrentTab: 'Open in current tab',
  favorite: 'Favorite',
  lightMode: 'Light',
  darkMode: 'Dark',
  language: 'Language',
  adminLogin: 'Navbox Admin',
  password: 'Password',
  login: 'Log in',
  loginFailed: 'Password is incorrect or session creation failed',
  checkingSession: 'Checking session',
  refreshing: 'Refreshing data',
  restorePasswordTitle: 'Restore Admin Password',
  restoreModeNote: 'Restore mode is enabled. Enter the configured Restore Token and set a new admin password.',
  restoreToken: 'Restore Token',
  newPassword: 'New password',
  confirmNewPassword: 'Confirm new password',
  resetPassword: 'Reset password',
  restoreDoneNote: 'After a successful reset, remove the restore config and restart the service to reopen normal admin APIs.',
  restoreSuccess: 'Admin password has been reset. Disable restore mode and restart the service before logging in.',
  restoreTokenInvalid: 'Restore Token is invalid, expired, or already used',
  passwordMin: 'New password must be at least 8 characters',
  passwordMismatch: 'The two new passwords do not match',
  navboxAdmin: 'Navbox Admin',
  adminStats: '{{sites}} sites, {{tags}} tags',
  logout: 'Log out',
  adminModules: 'admin modules',
  sitesTab: 'Sites',
  tagsTab: 'Tags',
  importExportTab: 'Import / Export',
  passwordTab: 'Password',
  newSite: 'New Site',
  saveOrder: 'Save Order',
  batchDelete: 'Batch Delete',
  drag: 'Drag',
  select: 'Select',
  site: 'Site',
  openMethod: 'Open Method',
  favoriteColumn: 'Favorite',
  sort: 'Sort',
  actions: 'Actions',
  dragSort: 'Drag to sort',
  selectSite: 'Select {{title}}',
  uncategorized: 'Uncategorized',
  yes: 'Yes',
  no: 'No',
  editSite: 'Edit Site',
  deleteSite: 'Delete Site',
  newTag: 'New Tag',
  status: 'Status',
  siteCount: 'Sites',
  noIcon: 'No icon',
  defaultStatus: 'Default',
  enabledStatus: 'Enabled',
  disabledStatus: 'Disabled',
  setDefault: 'Set default',
  editTag: 'Edit Tag',
  deleteTag: 'Delete Tag',
  export: 'Export',
  exportAll: 'Export All',
  exportSelectedSites: 'Export Selected Sites',
  exportSelectedTags: 'Export Selected Tags',
  import: 'Import',
  chooseZipImport: 'Choose zip to import',
  importResult: 'Import Result',
  importedSummary: 'Imported sites {{sites}}, tags {{tags}}, icons {{icons}}',
  skippedSummary: 'Skipped sites {{sites}}, tags {{tags}}, icons {{icons}}',
  currentPassword: 'Current password',
  changePassword: 'Change password',
  editSiteTitle: 'Edit Site',
  newSiteTitle: 'New Site',
  editTagTitle: 'Edit Tag',
  newTagTitle: 'New Tag',
  batchAddTag: 'Batch Add Tag',
  batchRemoveTag: 'Batch Remove Tag',
  close: 'Close',
  title: 'Title',
  defaultURL: 'Default URL',
  lanURL: 'LAN URL',
  description: 'Description',
  newWindow: 'New window',
  currentWindow: 'Current window',
  iconType: 'Icon Type',
  textIcon: 'Text',
  imageIcon: 'Image',
  onlineIcon: 'Online',
  backgroundColor: 'Background Color',
  iconValue: 'Icon Value',
  fetching: 'Fetching',
  fetchFromWebsite: 'Fetch from site',
  uploadIcon: 'Upload icon',
  onlyName: 'Name only',
  saveSite: 'Save Site',
  name: 'Name',
  icon: 'Icon',
  color: 'Color',
  enabled: 'Enabled',
  saveTag: 'Save Tag',
  currentTab: 'Current tab',
  newTab: 'New tab',
  siteUpdated: 'Site updated',
  siteCreated: 'Site created',
  confirmDeleteSite: 'Delete this site?',
  siteDeleted: 'Site deleted',
  iconUploaded: 'Icon uploaded',
  fillDefaultURL: 'Please enter the default URL first',
  iconFetched: 'Icon fetched',
  iconFetchFailed: 'No icon found',
  tagUpdated: 'Tag updated',
  tagCreated: 'Tag created',
  confirmDeleteTag: 'Delete this tag? Linked sites will not be deleted.',
  tagDeleted: 'Tag deleted',
  defaultTagUpdated: 'Default tag updated',
  siteOrderSaved: 'Site order saved',
  tagOrderSaved: 'Tag order saved',
  siteOrderUpdated: 'Site order updated',
  siteOrderUpdateFailed: 'Failed to update site order',
  tagOrderUpdated: 'Tag order updated',
  tagOrderUpdateFailed: 'Failed to update tag order',
  confirmBatchDelete: 'Delete the selected {{count}} sites?',
  batchDeleteDone: 'Batch delete completed',
  chooseSitesAndTags: 'Please choose sites and tags',
  batchAddTagDone: 'Batch tag add completed',
  batchRemoveTagDone: 'Batch tag remove completed',
  chooseSites: 'Please choose sites first',
  chooseTags: 'Please choose tags first',
  exportStarted: 'Export started',
  importDone: 'Import completed',
  passwordChanged: 'Password changed',
  sessionExpired: 'Your session has expired. Please log in again'
};

type PreferencesContextValue = {
  theme: ThemeMode;
  language: LanguageMode;
  setTheme: (theme: ThemeMode) => void;
  setLanguage: (language: LanguageMode) => void;
  t: (key: TranslationKey, values?: Record<string, string | number>) => string;
};

const PreferencesContext = createContext<PreferencesContextValue | null>(null);

export function PreferencesProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<ThemeMode>(() => loadTheme());
  const [language, setLanguageState] = useState<LanguageMode>(() => loadLanguage());

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
    try {
      window.localStorage.setItem(themeKey, theme);
    } catch {
      // localStorage 不可用时只保留当前会话状态。
    }
  }, [theme]);

  useEffect(() => {
    document.documentElement.lang = language === 'zh' ? 'zh-CN' : 'en';
    try {
      window.localStorage.setItem(languageKey, language);
    } catch {
      // localStorage 不可用时只保留当前会话状态。
    }
  }, [language]);

  const value = useMemo<PreferencesContextValue>(() => {
    const dictionary = language === 'zh' ? zh : en;
    return {
      theme,
      language,
      setTheme: setThemeState,
      setLanguage: setLanguageState,
      t: (key, values) => interpolate(dictionary[key], values)
    };
  }, [language, theme]);

  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>;
}

export function usePreferences(): PreferencesContextValue {
  const context = useContext(PreferencesContext);
  if (!context) {
    throw new Error('usePreferences must be used within PreferencesProvider');
  }
  return context;
}

export function PreferenceControls() {
  const { language, setLanguage, setTheme, t, theme } = usePreferences();

  return (
    <div className="preference-controls" aria-label={t('language')}>
      <button
        className={theme === 'light' ? 'preference-button active' : 'preference-button'}
        type="button"
        onClick={() => setTheme('light')}
        title={t('lightMode')}
      >
        <Sun size={15} aria-hidden="true" />
        <span>{t('lightMode')}</span>
      </button>
      <button
        className={theme === 'dark' ? 'preference-button active' : 'preference-button'}
        type="button"
        onClick={() => setTheme('dark')}
        title={t('darkMode')}
      >
        <Moon size={15} aria-hidden="true" />
        <span>{t('darkMode')}</span>
      </button>
      <button
        className={language === 'zh' ? 'preference-button active' : 'preference-button'}
        type="button"
        onClick={() => setLanguage(language === 'zh' ? 'en' : 'zh')}
        title={t('language')}
      >
        <Languages size={15} aria-hidden="true" />
        <span>{language === 'zh' ? '中文' : 'EN'}</span>
      </button>
    </div>
  );
}

function interpolate(text: string, values: Record<string, string | number> = {}): string {
  return Object.entries(values).reduce((result, [key, value]) => {
    return result.split(`{{${key}}}`).join(String(value));
  }, text);
}

function loadTheme(): ThemeMode {
  try {
    const value = window.localStorage.getItem(themeKey);
    return value === 'dark' ? 'dark' : 'light';
  } catch {
    return 'light';
  }
}

function loadLanguage(): LanguageMode {
  try {
    const value = window.localStorage.getItem(languageKey);
    return value === 'en' ? 'en' : 'zh';
  } catch {
    return 'zh';
  }
}
