import {
  DndContext,
  type DragEndEvent,
  DragOverlay,
  type DragStartEvent,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors
} from '@dnd-kit/core';
import {
  SortableContext,
  arrayMove,
  rectSortingStrategy,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import {
  ArrowDown,
  ArrowDownToLine,
  ArrowUp,
  ArrowUpToLine,
  GripVertical,
  Heart,
  Info,
  LayoutGrid,
  List,
  Loader2,
  RotateCcw,
  Save,
  X
} from 'lucide-react';
import { type CSSProperties, useEffect, useMemo, useState } from 'react';
import { usePreferences } from './preferences';
import type { Site } from './types';

type SortViewMode = 'list' | 'grid';

export function SiteSortPanel({
  sites,
  filteredSites,
  isFiltered,
  saving,
  onSave,
  onClose
}: {
  sites: Site[];
  filteredSites: Site[];
  isFiltered: boolean;
  saving: boolean;
  onSave: (orderedFullSites: Site[]) => Promise<void>;
  onClose: () => void;
}) {
  const { t } = usePreferences();
  const [draftSites, setDraftSites] = useState<Site[]>(() => [...sites]);
  const [viewMode, setViewMode] = useState<SortViewMode>('list');
  const [activeId, setActiveId] = useState<string | null>(null);
  const [editingRankId, setEditingRankId] = useState('');
  const [rankInput, setRankInput] = useState('');

  // Keep draft synchronized if underlying sites list changes externally (e.g. after save)
  useEffect(() => {
    setDraftSites([...sites]);
  }, [sites]);

  const filteredIdSet = useMemo(() => new Set(filteredSites.map((s) => s.id)), [filteredSites]);

  const visibleSites = useMemo(() => {
    if (!isFiltered) {
      return draftSites;
    }
    return draftSites.filter((site) => filteredIdSet.has(site.id));
  }, [draftSites, filteredIdSet, isFiltered]);

  const changedCount = useMemo(() => {
    const originalIndex = new Map(sites.map((site, index) => [site.id, index]));
    let moved = 0;
    draftSites.forEach((site, index) => {
      if (originalIndex.get(site.id) !== index) {
        moved++;
      }
    });
    return moved;
  }, [draftSites, sites]);

  const hasChanges = changedCount > 0;

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 6
      }
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates
    })
  );

  function applyVisibleReorder(nextVisible: Site[]) {
    if (!isFiltered) {
      setDraftSites(nextVisible);
      return;
    }
    // Preserve exact indices of hidden sites while replacing only filtered slots
    let cursor = 0;
    setDraftSites((current) =>
      current.map((site) => (filteredIdSet.has(site.id) ? nextVisible[cursor++] : site))
    );
  }

  function handleDragStart(event: DragStartEvent) {
    setActiveId(String(event.active.id));
    setEditingRankId('');
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveId(null);
    const { active, over } = event;
    if (!over || active.id === over.id) {
      return;
    }
    const oldIndex = visibleSites.findIndex((site) => site.id === String(active.id));
    const newIndex = visibleSites.findIndex((site) => site.id === String(over.id));
    if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) {
      return;
    }
    applyVisibleReorder(arrayMove(visibleSites, oldIndex, newIndex));
  }

  function moveVisibleToIndex(id: string, targetIndex: number) {
    const oldIndex = visibleSites.findIndex((site) => site.id === id);
    if (oldIndex < 0) {
      return;
    }
    const clamped = Math.max(0, Math.min(visibleSites.length - 1, targetIndex));
    if (oldIndex === clamped) {
      return;
    }
    applyVisibleReorder(arrayMove(visibleSites, oldIndex, clamped));
  }

  function startRankEdit(siteId: string, currentRank: number) {
    setEditingRankId(siteId);
    setRankInput(String(currentRank));
  }

  function commitRankEdit(siteId: string) {
    const parsed = Number.parseInt(rankInput, 10);
    setEditingRankId('');
    if (!Number.isFinite(parsed)) {
      return;
    }
    moveVisibleToIndex(siteId, parsed - 1);
  }

  function handleReset() {
    setDraftSites([...sites]);
    setEditingRankId('');
  }

  function handleExit() {
    if (saving) {
      return;
    }
    if (hasChanges && !window.confirm(t('sortUnsavedConfirm'))) {
      return;
    }
    onClose();
  }

  async function handleSave() {
    if (!hasChanges || saving) {
      return;
    }
    await onSave(draftSites);
  }

  const activeSite = activeId ? visibleSites.find((site) => site.id === activeId) ?? null : null;
  const activeRank = activeSite ? visibleSites.findIndex((site) => site.id === activeSite.id) + 1 : 0;

  return (
    <div className="sort-workspace">
      <div className="sort-workspace-bar">
        <div className="sort-workspace-copy">
          <div className="sort-workspace-title-row">
            <strong>{t('sortWorkspaceTitle')}</strong>
            <span className={hasChanges ? 'status-pill status-info' : 'status-pill'}>
              {hasChanges ? t('sortChangedCount', { count: changedCount }) : t('sortUnchanged')}
            </span>
          </div>
          <span>{t('sortWorkspaceHint')}</span>
        </div>

        <div className="sort-workspace-controls">
          <div className="export-scope sort-view-toggle" role="group" aria-label={t('sortWorkspaceTitle')}>
            <button
              type="button"
              className={viewMode === 'list' ? 'active' : undefined}
              onClick={() => setViewMode('list')}
            >
              <List size={15} aria-hidden="true" />
              {t('sortViewList')}
            </button>
            <button
              type="button"
              className={viewMode === 'grid' ? 'active' : undefined}
              onClick={() => setViewMode('grid')}
            >
              <LayoutGrid size={15} aria-hidden="true" />
              {t('sortViewGrid')}
            </button>
          </div>

          <button
            type="button"
            className="secondary-btn"
            disabled={!hasChanges || saving}
            onClick={handleReset}
          >
            <RotateCcw size={15} aria-hidden="true" />
            {t('sortReset')}
          </button>

          <button
            type="button"
            className="primary-button"
            disabled={!hasChanges || saving}
            onClick={() => void handleSave()}
          >
            {saving ? <Loader2 size={15} className="spin" aria-hidden="true" /> : <Save size={15} aria-hidden="true" />}
            {t('saveOrder')}
          </button>

          <button type="button" className="secondary-btn" disabled={saving} onClick={handleExit}>
            <X size={15} aria-hidden="true" />
            {t('sortExit')}
          </button>
        </div>
      </div>

      {isFiltered && (
        <div className="sort-filtered-banner">
          <Info size={15} aria-hidden="true" />
          <span>{t('sortFilteredNotice', { count: visibleSites.length })}</span>
        </div>
      )}

      {visibleSites.length === 0 ? (
        <div className="admin-state">
          <span>{t('noSites')}</span>
        </div>
      ) : (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
          onDragCancel={() => setActiveId(null)}
        >
          <SortableContext
            items={visibleSites.map((site) => site.id)}
            strategy={viewMode === 'grid' ? rectSortingStrategy : verticalListSortingStrategy}
          >
            <div className={viewMode === 'grid' ? 'sort-grid-container' : 'sort-list-container'}>
              {visibleSites.map((site, index) => (
                <SortableSiteItem
                  key={site.id}
                  site={site}
                  rank={index + 1}
                  total={visibleSites.length}
                  viewMode={viewMode}
                  editingRank={editingRankId === site.id}
                  rankInput={rankInput}
                  onRankInputChange={setRankInput}
                  onStartRankEdit={() => startRankEdit(site.id, index + 1)}
                  onCommitRankEdit={() => commitRankEdit(site.id)}
                  onCancelRankEdit={() => setEditingRankId('')}
                  onMoveTop={() => moveVisibleToIndex(site.id, 0)}
                  onMoveUp={() => moveVisibleToIndex(site.id, index - 1)}
                  onMoveDown={() => moveVisibleToIndex(site.id, index + 1)}
                  onMoveBottom={() => moveVisibleToIndex(site.id, visibleSites.length - 1)}
                />
              ))}
            </div>
          </SortableContext>

          <DragOverlay adjustScale={false} dropAnimation={{ duration: 180, easing: 'cubic-bezier(0.2, 0, 0, 1)' }}>
            {activeSite ? (
              <div className={viewMode === 'grid' ? 'sort-grid-card sort-overlay' : 'sort-list-row sort-overlay'}>
                <span className="sort-drag-grip">
                  <GripVertical size={16} aria-hidden="true" />
                </span>
                <span className="sort-rank-badge">#{activeRank}</span>
                <SiteThumb site={activeSite} />
                <div className="sort-item-copy">
                  <strong>{activeSite.title}</strong>
                  <span>{activeSite.default_url}</span>
                </div>
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>
      )}
    </div>
  );
}

function SortableSiteItem({
  site,
  rank,
  total,
  viewMode,
  editingRank,
  rankInput,
  onRankInputChange,
  onStartRankEdit,
  onCommitRankEdit,
  onCancelRankEdit,
  onMoveTop,
  onMoveUp,
  onMoveDown,
  onMoveBottom
}: {
  site: Site;
  rank: number;
  total: number;
  viewMode: SortViewMode;
  editingRank: boolean;
  rankInput: string;
  onRankInputChange: (val: string) => void;
  onStartRankEdit: () => void;
  onCommitRankEdit: () => void;
  onCancelRankEdit: () => void;
  onMoveTop: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onMoveBottom: () => void;
}) {
  const { t } = usePreferences();
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id: site.id });

  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition
  };

  const isFirst = rank === 1;
  const isLast = rank === total;

  const rankControl = editingRank ? (
    <input
      className="sort-rank-input"
      type="number"
      min={1}
      max={total}
      value={rankInput}
      autoFocus
      onClick={(e) => e.stopPropagation()}
      onChange={(e) => onRankInputChange(e.target.value)}
      onBlur={onCommitRankEdit}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          onCommitRankEdit();
        } else if (e.key === 'Escape') {
          e.preventDefault();
          onCancelRankEdit();
        }
      }}
      aria-label={t('sortJumpRankTitle', { max: total })}
    />
  ) : (
    <button
      type="button"
      className="sort-rank-badge"
      onClick={(e) => {
        e.stopPropagation();
        onStartRankEdit();
      }}
      title={t('sortJumpRankTitle', { max: total })}
    >
      #{rank}
    </button>
  );

  const stepButtons = (
    <div className="sort-step-buttons" onClick={(e) => e.stopPropagation()}>
      <button type="button" disabled={isFirst} onClick={onMoveTop} title={t('sortMoveTop')} aria-label={t('sortMoveTop')}>
        <ArrowUpToLine size={14} aria-hidden="true" />
      </button>
      <button type="button" disabled={isFirst} onClick={onMoveUp} title={t('moveUp')} aria-label={t('moveUp')}>
        <ArrowUp size={14} aria-hidden="true" />
      </button>
      <button type="button" disabled={isLast} onClick={onMoveDown} title={t('moveDown')} aria-label={t('moveDown')}>
        <ArrowDown size={14} aria-hidden="true" />
      </button>
      <button type="button" disabled={isLast} onClick={onMoveBottom} title={t('sortMoveBottom')} aria-label={t('sortMoveBottom')}>
        <ArrowDownToLine size={14} aria-hidden="true" />
      </button>
    </div>
  );

  if (viewMode === 'grid') {
    return (
      <article
        ref={setNodeRef}
        style={style}
        className={isDragging ? 'sort-grid-card is-dragging' : 'sort-grid-card'}
      >
        <div className="sort-grid-card-top">
          <span className="sort-drag-grip" {...attributes} {...listeners} title={t('dragSort')}>
            <GripVertical size={16} aria-hidden="true" />
          </span>
          {rankControl}
          {site.is_favorite && <Heart size={13} className="favorite-icon" aria-hidden="true" />}
          <div className="sort-grid-card-spacer" />
          {stepButtons}
        </div>
        <div className="sort-grid-card-body" {...attributes} {...listeners}>
          <SiteThumb site={site} />
          <div className="sort-item-copy">
            <strong>{site.title}</strong>
            <span>{site.description || site.default_url}</span>
          </div>
        </div>
        {site.tags.length > 0 && (
          <div className="table-tags sort-item-tags">
            {site.tags.slice(0, 3).map((tag) => (
              <span className="table-tag" key={tag.id}>
                {tag.name}
              </span>
            ))}
          </div>
        )}
      </article>
    );
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={isDragging ? 'sort-list-row is-dragging' : 'sort-list-row'}
    >
      <span className="sort-drag-grip" {...attributes} {...listeners} title={t('dragSort')}>
        <GripVertical size={16} aria-hidden="true" />
      </span>
      {rankControl}
      <div className="sort-list-main" {...attributes} {...listeners}>
        <SiteThumb site={site} />
        <div className="sort-item-copy">
          <strong>
            {site.title}
            {site.is_favorite && <Heart size={13} className="favorite-icon" aria-hidden="true" />}
          </strong>
          <span>{site.default_url}</span>
        </div>
      </div>
      <div className="table-tags sort-list-tags desktop-only">
        {site.tags.length > 0 ? (
          site.tags.slice(0, 3).map((tag) => (
            <span className="table-tag" key={tag.id}>
              {tag.name}
            </span>
          ))
        ) : (
          <span className="table-tag muted">{t('uncategorized')}</span>
        )}
      </div>
      {stepButtons}
    </div>
  );
}

function SiteThumb({ site }: { site: Site }) {
  const letter = (site.icon_value || site.title || '?').trim().slice(0, 2);
  const bg = site.background_color || undefined;

  if ((site.icon_type === 'image' || site.icon_type === 'online') && site.icon_value) {
    return (
      <span className="sort-site-thumb" style={bg ? { background: bg } : undefined}>
        <img src={site.icon_value} alt="" loading="lazy" />
      </span>
    );
  }
  return (
    <span className="sort-site-thumb sort-site-thumb-text" style={bg ? { background: bg } : undefined}>
      {letter}
    </span>
  );
}
