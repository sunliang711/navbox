import { Check, ImageOff, Images, Loader2, Search, X } from 'lucide-react';
import { type CSSProperties, useEffect, useMemo, useRef, useState } from 'react';
import { isUnauthorizedError, listAdminIcons } from './api';
import { usePreferences } from './preferences';
import type { IconLibraryItem } from './types';

const iconPageSize = 48;

export function SiteIconPreview({
  title,
  iconType,
  iconValue,
  backgroundColor,
  size = 'md',
  onImageStatusChange
}: {
  title: string;
  iconType: string;
  iconValue: string;
  backgroundColor?: string;
  size?: 'sm' | 'md' | 'lg';
  onImageStatusChange?: (broken: boolean) => void;
}) {
  const trimmedValue = iconValue.trim();
  const isImage = Boolean(
    trimmedValue && (iconType === 'image' || iconType === 'online' || trimmedValue.startsWith('/uploads/'))
  );
  const [broken, setBroken] = useState(false);

  useEffect(() => {
    setBroken(false);
    onImageStatusChange?.(false);
  }, [trimmedValue, iconType]);

  const bgStyle: CSSProperties | undefined = backgroundColor?.trim()
    ? { backgroundColor: backgroundColor.trim() }
    : undefined;

  const sizeClass = size === 'sm' ? 'site-icon-sm' : size === 'lg' ? 'site-icon-lg' : 'site-icon-md';

  if (isImage) {
    if (broken) {
      return (
        <div className={`site-icon image-icon is-broken ${sizeClass}`} style={bgStyle}>
          <ImageOff size={size === 'sm' ? 14 : 18} aria-hidden="true" />
        </div>
      );
    }
    return (
      <div className={`site-icon image-icon ${sizeClass}`} style={bgStyle}>
        <img
          src={trimmedValue}
          alt=""
          onLoad={() => {
            setBroken(false);
            onImageStatusChange?.(false);
          }}
          onError={() => {
            setBroken(true);
            onImageStatusChange?.(true);
          }}
        />
      </div>
    );
  }

  const label = (trimmedValue || title || 'NB').trim().slice(0, 2).toUpperCase();
  return (
    <div className={`site-icon text-icon ${sizeClass}`} style={bgStyle}>
      {label}
    </div>
  );
}

function formatIconBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatIconFormat(mimeType: string, fileName: string): string {
  const lower = mimeType.toLowerCase();
  if (lower.includes('svg')) {
    return 'SVG';
  }
  if (lower.includes('png')) {
    return 'PNG';
  }
  if (lower.includes('webp')) {
    return 'WEBP';
  }
  if (lower.includes('jpeg') || lower.includes('jpg')) {
    return 'JPG';
  }
  if (lower.includes('gif')) {
    return 'GIF';
  }
  if (lower.includes('icon')) {
    return 'ICO';
  }
  const ext = fileName.split('.').pop();
  return ext ? ext.toUpperCase() : 'IMG';
}

export function IconPickerDialog({
  open,
  currentIconValue,
  siteTitle,
  backgroundColor,
  onSelect,
  onClose,
  onUnauthorized,
  reauthVersion = 0
}: {
  open: boolean;
  currentIconValue: string;
  siteTitle: string;
  backgroundColor: string;
  onSelect: (icon: IconLibraryItem) => void;
  onClose: () => void;
  onUnauthorized: () => void;
  reauthVersion?: number;
}) {
  const { t } = usePreferences();
  const dialogRef = useRef<HTMLDivElement | null>(null);
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [unusedOnly, setUnusedOnly] = useState(false);
  const [items, setItems] = useState<IconLibraryItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadError, setLoadError] = useState('');
  const [selectedId, setSelectedId] = useState('');
  const callbacksRef = useRef({ onClose, onUnauthorized, t, currentIconValue });
  callbacksRef.current = { onClose, onUnauthorized, t, currentIconValue };

  useEffect(() => {
    if (!open) {
      return;
    }
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    dialogRef.current?.focus();
    return () => {
      document.body.style.overflow = prevOverflow;
      previousFocus?.focus();
    };
  }, [open]);

  useEffect(() => {
    if (!open) {
      setSearchInput('');
      setSearchQuery('');
      setUnusedOnly(false);
      setLoadError('');
      setSelectedId('');
      return;
    }
  }, [open]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const timer = window.setTimeout(() => {
      setSearchQuery(searchInput.trim());
    }, 180);
    return () => window.clearTimeout(timer);
  }, [open, searchInput]);

  useEffect(() => {
    if (!open) {
      return;
    }
    let cancelled = false;
    setLoading(true);
    setLoadError('');
    listAdminIcons({ q: searchQuery, unused: unusedOnly, limit: iconPageSize, offset: 0 })
      .then((res) => {
        if (cancelled) {
          return;
        }
        setItems(res.items);
        setTotal(res.total);
        const currentVal = callbacksRef.current.currentIconValue.trim();
        setSelectedId((prev) => {
          if (prev && res.items.some((item) => item.id === prev)) {
            return prev;
          }
          const matched = res.items.find((item) => item.url === currentVal);
          return matched ? matched.id : (res.items[0]?.id ?? '');
        });
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        if (isUnauthorizedError(error)) {
          callbacksRef.current.onUnauthorized();
          return;
        }
        setLoadError(callbacksRef.current.t('iconLibraryLoadFailed'));
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [open, searchQuery, unusedOnly, reauthVersion]);

  async function loadMore() {
    if (loadingMore || items.length >= total) {
      return;
    }
    setLoadingMore(true);
    try {
      const res = await listAdminIcons({
        q: searchQuery,
        unused: unusedOnly,
        limit: iconPageSize,
        offset: items.length
      });
      setItems((prev) => {
        const seen = new Set(prev.map((item) => item.id));
        const next = res.items.filter((item) => !seen.has(item.id));
        return [...prev, ...next];
      });
      setTotal(res.total);
    } catch (error) {
      if (isUnauthorizedError(error)) {
        onUnauthorized();
        return;
      }
      setLoadError(t('iconLibraryLoadFailed'));
    } finally {
      setLoadingMore(false);
    }
  }

  const selectedIcon = useMemo(
    () => items.find((item) => item.id === selectedId) ?? null,
    [items, selectedId]
  );

  if (!open) {
    return null;
  }

  function renderUsageCaption(item: IconLibraryItem): string {
    if (item.used_by.length === 0) {
      return t('iconUnused');
    }
    if (item.used_by.length === 1) {
      return item.used_by[0].title;
    }
    return t('iconUsedByMultiple', { title: item.used_by[0].title, count: item.used_by.length });
  }

  return (
    <div
      className="confirm-backdrop icon-picker-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <div
        ref={dialogRef}
        className="icon-picker-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="icon-picker-title"
        tabIndex={-1}
        onKeyDown={(event) => {
          if (event.key === 'Escape') {
            event.preventDefault();
            event.stopPropagation();
            onClose();
            return;
          }
          if (event.key === 'Tab') {
            const controls = Array.from(
              event.currentTarget.querySelectorAll<HTMLElement>(
                'button, input, select, textarea, [href], [tabindex]'
              )
            ).filter((el) => !el.matches(':disabled') && el.tabIndex >= 0 && el.getClientRects().length > 0);
            const first = controls[0];
            const last = controls[controls.length - 1];
            if (event.shiftKey && (document.activeElement === first || document.activeElement === dialogRef.current)) {
              event.preventDefault();
              last?.focus();
            } else if (!event.shiftKey && (document.activeElement === last || document.activeElement === dialogRef.current)) {
              event.preventDefault();
              first?.focus();
            }
          }
        }}
      >
        <header className="icon-picker-header">
          <div className="icon-picker-heading">
            <span className="icon-picker-badge" aria-hidden="true">
              <Images size={18} />
            </span>
            <div>
              <h2 id="icon-picker-title">{t('iconLibraryTitle')}</h2>
              <p>{t('iconLibrarySubtitle', { total })}</p>
            </div>
          </div>
          <button type="button" className="icon-picker-close" onClick={onClose} title={t('close')} aria-label={t('close')}>
            <X size={18} aria-hidden="true" />
          </button>
        </header>

        <div className="icon-picker-toolbar">
          <label className="icon-picker-search">
            <Search size={16} aria-hidden="true" />
            <input
              type="search"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder={t('iconSearchPlaceholder')}
              aria-label={t('iconSearchPlaceholder')}
            />
          </label>
          <div className="icon-picker-filters" role="group" aria-label={t('iconLibraryTitle')}>
            <button
              type="button"
              className={!unusedOnly ? 'is-active' : undefined}
              onClick={() => setUnusedOnly(false)}
            >
              {t('iconFilterAll')}
            </button>
            <button
              type="button"
              className={unusedOnly ? 'is-active' : undefined}
              onClick={() => setUnusedOnly(true)}
            >
              {t('iconFilterUnused')}
            </button>
          </div>
        </div>

        <div className="icon-picker-body">
          {loading ? (
            <div className="icon-picker-state">
              <Loader2 className="spin" size={20} aria-hidden="true" />
              <span>{t('loading')}</span>
            </div>
          ) : loadError ? (
            <div className="icon-picker-state is-error">
              <span>{loadError}</span>
            </div>
          ) : items.length === 0 ? (
            <div className="icon-picker-state">
              <Images size={22} aria-hidden="true" />
              <span>{t('iconLibraryEmpty')}</span>
            </div>
          ) : (
            <>
              <div className="icon-picker-grid" role="listbox" aria-label={t('iconLibraryTitle')}>
                {items.map((item) => {
                  const selected = item.id === selectedId;
                  const caption = renderUsageCaption(item);
                  return (
                    <button
                      key={item.id}
                      type="button"
                      role="option"
                      aria-selected={selected}
                      className={`icon-picker-card${selected ? ' is-selected' : ''}`}
                      onClick={() => setSelectedId(item.id)}
                      onDoubleClick={() => onSelect(item)}
                      title={`${caption} (${item.file_name})`}
                    >
                      {selected && (
                        <span className="icon-picker-check" aria-hidden="true">
                          <Check size={12} />
                        </span>
                      )}
                      <span className="icon-picker-thumb">
                        <img src={item.url} alt="" loading="lazy" />
                      </span>
                      <span className={`icon-picker-caption${item.used_by.length === 0 ? ' is-unused' : ''}`}>
                        {caption}
                      </span>
                    </button>
                  );
                })}
              </div>
              {items.length < total && (
                <div className="icon-picker-more">
                  <button type="button" onClick={loadMore} disabled={loadingMore}>
                    {loadingMore && <Loader2 className="spin" size={15} aria-hidden="true" />}
                    {t('iconLoadMore')} ({items.length}/{total})
                  </button>
                </div>
              )}
            </>
          )}
        </div>

        <footer className="icon-picker-footer">
          <div className="icon-picker-preview-panel">
            {selectedIcon ? (
              <>
                <SiteIconPreview
                  title={siteTitle || selectedIcon.used_by[0]?.title || 'NB'}
                  iconType="image"
                  iconValue={selectedIcon.url}
                  backgroundColor={backgroundColor}
                  size="lg"
                />
                <div className="icon-picker-preview-meta">
                  <div className="icon-picker-preview-title">
                    <strong>{t('iconSitePreviewLabel')}</strong>
                    <span>
                      {formatIconFormat(selectedIcon.mime_type, selectedIcon.file_name)} · {formatIconBytes(selectedIcon.size_bytes)}
                    </span>
                  </div>
                  <p className="icon-picker-preview-usage">
                    {selectedIcon.used_by.length > 0
                      ? `${t('iconUsedByLabel')}${selectedIcon.used_by.map((site) => site.title).join('、')}`
                      : t('iconUnusedBadge')}
                  </p>
                </div>
              </>
            ) : (
              <p className="icon-picker-preview-empty">{t('iconSelectPrompt')}</p>
            )}
          </div>
          <div className="icon-picker-footer-actions">
            <button type="button" onClick={onClose}>
              {t('cancel')}
            </button>
            <button
              type="button"
              className="primary-button"
              disabled={!selectedIcon}
              onClick={() => {
                if (selectedIcon) {
                  onSelect(selectedIcon);
                }
              }}
            >
              <Check size={16} aria-hidden="true" />
              {t('iconUseSelected')}
            </button>
          </div>
        </footer>
      </div>
    </div>
  );
}
