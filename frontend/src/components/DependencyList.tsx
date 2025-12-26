import { useState, useMemo } from 'react';
import type { GraphData, GraphNode } from '../types/api';

interface DependencyListProps {
  data: GraphData;
}

type SortKey = 'name' | 'stars';
type SortOrder = 'asc' | 'desc';

export function DependencyList({ data }: DependencyListProps) {
  const [search, setSearch] = useState('');
  const [sortKey, setSortKey] = useState<SortKey>('stars');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());

  // Filter out virtual/subdivider nodes and sort
  const filteredNodes = useMemo(() => {
    let nodes = data.nodes.filter(n => 
      n.kind !== 'subdivider' && 
      n.kind !== 'auxiliary' &&
      n.id !== '__project__'
    );

    // Search filter
    if (search) {
      const lowerSearch = search.toLowerCase();
      nodes = nodes.filter(n => 
        n.id.toLowerCase().includes(lowerSearch) ||
        n.meta?.summary?.toLowerCase().includes(lowerSearch) ||
        n.meta?.description?.toLowerCase().includes(lowerSearch)
      );
    }

    // Sort
    nodes.sort((a, b) => {
      let cmp = 0;
      switch (sortKey) {
        case 'name':
          cmp = a.id.localeCompare(b.id);
          break;
        case 'stars':
          cmp = (b.meta?.repo_stars || 0) - (a.meta?.repo_stars || 0);
          break;
      }
      return sortOrder === 'asc' ? cmp : -cmp;
    });

    return nodes;
  }, [data.nodes, search, sortKey, sortOrder]);

  const toggleExpand = (id: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // Calculate stats
  const stats = useMemo(() => {
    const realNodes = data.nodes.filter(n => n.kind !== 'subdivider' && n.kind !== 'auxiliary' && n.id !== '__project__');
    const withStars = realNodes.filter(n => n.meta?.repo_stars);
    const totalStars = withStars.reduce((sum, n) => sum + (n.meta?.repo_stars || 0), 0);
    const archived = realNodes.filter(n => n.meta?.repo_archived).length;
    
    return {
      total: realNodes.length,
      avgStars: withStars.length ? Math.round(totalStars / withStars.length) : 0,
      archived,
    };
  }, [data]);

  return (
    <div className="flex flex-col h-full -m-4">
      {/* Mini stats */}
      <div className="flex gap-3 px-4 py-3 border-b border-[var(--color-border)] bg-[var(--color-bg)]">
        <div className="flex-1 text-center">
          <div className="text-lg font-bold text-[var(--color-text)]">{stats.total}</div>
          <div className="text-xs text-[var(--color-text-muted)]">Total</div>
        </div>
        <div className="flex-1 text-center">
          <div className="text-lg font-bold text-[var(--color-text)]">{stats.avgStars.toLocaleString()}</div>
          <div className="text-xs text-[var(--color-text-muted)]">Avg ⭐</div>
        </div>
        {stats.archived > 0 && (
          <div className="flex-1 text-center">
            <div className="text-lg font-bold text-[var(--color-warning)]">{stats.archived}</div>
            <div className="text-xs text-[var(--color-text-muted)]">Archived</div>
          </div>
        )}
      </div>

      {/* Search & Sort */}
      <div className="flex gap-2 px-4 py-3 border-b border-[var(--color-border)]">
        <div className="relative flex-1">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter..."
            className="input py-1.5 pl-8 text-sm"
          />
          <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
        <select
          value={`${sortKey}-${sortOrder}`}
          onChange={(e) => {
            const [key, order] = e.target.value.split('-') as [SortKey, SortOrder];
            setSortKey(key);
            setSortOrder(order);
          }}
          className="input w-auto py-1.5 text-sm"
        >
          <option value="stars-desc">⭐ Most</option>
          <option value="stars-asc">⭐ Least</option>
          <option value="name-asc">A → Z</option>
          <option value="name-desc">Z → A</option>
        </select>
      </div>

      {/* Dependency list */}
      <div className="flex-1 overflow-y-auto">
        {filteredNodes.length === 0 ? (
          <p className="text-center text-[var(--color-text-muted)] py-8 text-sm">
            {search ? 'No matches' : 'No dependencies'}
          </p>
        ) : (
          <div className="divide-y divide-[var(--color-border)]">
            {filteredNodes.map(node => (
              <DependencyItem 
                key={node.id} 
                node={node} 
                isExpanded={expandedNodes.has(node.id)}
                onToggle={() => toggleExpand(node.id)}
                edges={data.edges}
              />
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="px-4 py-2 border-t border-[var(--color-border)] text-xs text-[var(--color-text-muted)] text-center bg-[var(--color-bg)]">
        {filteredNodes.length} of {stats.total}
      </div>
    </div>
  );
}

interface DependencyItemProps {
  node: GraphNode;
  isExpanded: boolean;
  onToggle: () => void;
  edges: { from: string; to: string }[];
}

function DependencyItem({ node, isExpanded, onToggle, edges }: DependencyItemProps) {
  const meta = node.meta || {};
  const dependsOn = edges.filter(e => e.from === node.id).map(e => e.to);
  const dependedBy = edges.filter(e => e.to === node.id).map(e => e.from);

  return (
    <div className={`${isExpanded ? 'bg-[var(--color-bg)]' : 'hover:bg-[var(--color-bg-hover)]'} transition-colors`}>
      <button
        onClick={onToggle}
        className="w-full px-4 py-3 text-left"
      >
        <div className="flex items-center gap-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm font-medium text-[var(--color-text)] truncate">
                {node.id}
              </span>
              {meta.version && (
                <span className="badge badge-neutral text-xs">
                  {meta.version}
                </span>
              )}
              {meta.repo_archived && (
                <span className="badge badge-warning text-xs">
                  Archived
                </span>
              )}
            </div>
            {meta.summary && !isExpanded && (
              <p className="text-xs text-[var(--color-text-muted)] truncate mt-0.5">
                {meta.summary}
              </p>
            )}
          </div>
          <div className="flex items-center gap-3 flex-shrink-0">
            {meta.repo_stars !== undefined && meta.repo_stars > 0 && (
              <span className="flex items-center gap-1 text-xs text-[var(--color-text-muted)]">
                <span className="text-amber-500">⭐</span>
                {meta.repo_stars.toLocaleString()}
              </span>
            )}
            <svg 
              className={`w-4 h-4 text-[var(--color-text-muted)] transition-transform ${isExpanded ? 'rotate-180' : ''}`} 
              fill="none" 
              stroke="currentColor" 
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 9l-7 7-7-7" />
            </svg>
          </div>
        </div>
      </button>

      {isExpanded && (
        <div className="px-4 pb-4 space-y-3">
          {/* Description */}
          {(meta.description || meta.summary) && (
            <p className="text-sm text-[var(--color-text-secondary)]">
              {meta.description || meta.summary}
            </p>
          )}

          {/* Links */}
          <div className="flex flex-wrap gap-2">
            {meta.repo_url && (
              <a
                href={meta.repo_url}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-ghost text-xs py-1 px-2"
              >
                <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                </svg>
                Repo
              </a>
            )}
            {meta.homepage && meta.homepage !== meta.repo_url && (
              <a
                href={meta.homepage}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-ghost text-xs py-1 px-2"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                </svg>
                Site
              </a>
            )}
            {meta.license && (
              <span className="badge badge-neutral">
                {meta.license}
              </span>
            )}
          </div>

          {/* Dependencies */}
          {(dependsOn.length > 0 || dependedBy.length > 0) && (
            <div className="space-y-2 pt-2 border-t border-[var(--color-border)]">
              {dependsOn.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-[var(--color-text-muted)] mb-1.5">
                    Depends on ({dependsOn.length})
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {dependsOn.slice(0, 8).map(dep => (
                      <span key={dep} className="px-1.5 py-0.5 text-xs font-mono bg-blue-50 text-blue-700 rounded">
                        {dep}
                      </span>
                    ))}
                    {dependsOn.length > 8 && (
                      <span className="px-1.5 py-0.5 text-xs text-[var(--color-text-muted)]">
                        +{dependsOn.length - 8}
                      </span>
                    )}
                  </div>
                </div>
              )}
              {dependedBy.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-[var(--color-text-muted)] mb-1.5">
                    Required by ({dependedBy.length})
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {dependedBy.slice(0, 8).map(dep => (
                      <span key={dep} className="px-1.5 py-0.5 text-xs font-mono bg-emerald-50 text-emerald-700 rounded">
                        {dep}
                      </span>
                    ))}
                    {dependedBy.length > 8 && (
                      <span className="px-1.5 py-0.5 text-xs text-[var(--color-text-muted)]">
                        +{dependedBy.length - 8}
                      </span>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Last commit */}
          {meta.repo_last_commit && (
            <p className="text-xs text-[var(--color-text-muted)]">
              Last commit: {new Date(meta.repo_last_commit).toLocaleDateString()}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
