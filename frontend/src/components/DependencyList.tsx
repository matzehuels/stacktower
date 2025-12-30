/**
 * Dependency list sidebar component.
 */

import { useState, useMemo, useRef, useEffect } from 'react';
import { Search, ChevronDown, Github, ExternalLink, Star, Package, ArrowRight, AlertTriangle, Award, Users, ChevronsRight } from 'lucide-react';
import type { GraphData, GraphNode, NebraskaRanking as NebraskaRankingType } from '@/types/api';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { NebraskaRanking } from '@/components/NebraskaRanking';

// GraphData with Nebraska rankings added (Nebraska comes from layout/render, not graph)
interface DependencyDataWithNebraska extends GraphData {
  nebraska?: NebraskaRankingType[];
}

interface DependencyListProps {
  data: DependencyDataWithNebraska;
  /** Called when hovering over a dependency - passes the package name */
  onHighlight?: (packageName: string) => void;
  /** Called when hover ends */
  onClearHighlight?: () => void;
  /** Package name currently highlighted from hover */
  highlightedPackage?: string | null;
  /** Package name selected/clicked from SVG - should be expanded */
  selectedPackage?: string | null;
  /** Called when selection changes */
  onSelectPackage?: (packageName: string | null) => void;
  /** Called when collapse button is clicked */
  onCollapse?: () => void;
}

type SortOption = 'stars-desc' | 'stars-asc' | 'name-asc' | 'name-desc';
type FilterOption = 'all' | 'direct' | 'brittle';
type ViewTab = 'dependencies' | 'nebraska';

export function DependencyList({ data, onHighlight, onClearHighlight, highlightedPackage, selectedPackage, onSelectPackage, onCollapse }: DependencyListProps) {
  const [activeTab, setActiveTab] = useState<ViewTab>('dependencies');
  const [search, setSearch] = useState('');
  const [sortOption, setSortOption] = useState<SortOption>('stars-desc');
  const [filterOption, setFilterOption] = useState<FilterOption>('all');
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());

  // Identify root node (no incoming edges) and dependencies
  const { rootNode, dependencies, directDepIds } = useMemo(() => {
    const realNodes = data.nodes.filter(n =>
      n.kind !== 'subdivider' &&
      n.kind !== 'auxiliary' &&
      n.id !== '__project__'
    );

    // Find nodes that have no incoming edges (nothing depends on them)
    const nodesWithIncoming = new Set(data.edges.map(e => e.to));
    const roots = realNodes.filter(n => !nodesWithIncoming.has(n.id));
    
    // The root is typically the one with the most outgoing edges
    let root: GraphNode | null = null;
    if (roots.length === 1) {
      root = roots[0];
    } else if (roots.length > 1) {
      // Pick the one with most dependencies
      const outgoingCount = (id: string) => data.edges.filter(e => e.from === id).length;
      root = roots.reduce((a, b) => outgoingCount(a.id) >= outgoingCount(b.id) ? a : b);
    }

    // Dependencies are all nodes except the root
    const deps = root ? realNodes.filter(n => n.id !== root.id) : realNodes;

    // Direct dependencies = nodes that root points to directly
    const directIds = new Set(
      root 
        ? data.edges.filter(e => e.from === root.id).map(e => e.to)
        : []
    );

    return { rootNode: root, dependencies: deps, directDepIds: directIds };
  }, [data.nodes, data.edges]);

  // Filter and sort dependencies
  const filteredDeps = useMemo(() => {
    let nodes = [...dependencies];

    // Filter by direct/all/brittle
    if (filterOption === 'direct') {
      nodes = nodes.filter(n => directDepIds.has(n.id));
    } else if (filterOption === 'brittle') {
      nodes = nodes.filter(n => n.brittle && !n.meta?.repo_archived);
    }

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
    const [sortKey, sortOrder] = sortOption.split('-') as ['stars' | 'name', 'asc' | 'desc'];
    nodes.sort((a, b) => {
      let cmp = 0;
      switch (sortKey) {
        case 'name':
          cmp = a.id.localeCompare(b.id);
          break;
        case 'stars':
          // Base: ascending (a - b), then invert for desc
          cmp = (a.meta?.repo_stars || 0) - (b.meta?.repo_stars || 0);
          break;
      }
      return sortOrder === 'desc' ? -cmp : cmp;
    });

    return nodes;
  }, [dependencies, search, sortOption, filterOption, directDepIds]);

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

  // Calculate stats for dependencies only
  const stats = useMemo(() => {
    const withStars = dependencies.filter(n => n.meta?.repo_stars);
    const totalStars = withStars.reduce((sum, n) => sum + (n.meta?.repo_stars || 0), 0);
    const archived = dependencies.filter(n => n.meta?.repo_archived).length;
    const brittle = dependencies.filter(n => n.brittle && !n.meta?.repo_archived).length;

    return {
      total: dependencies.length,
      direct: directDepIds.size,
      avgStars: withStars.length ? Math.round(totalStars / withStars.length) : 0,
      archived,
      brittle,
    };
  }, [dependencies, directDepIds]);

  return (
    <div className="flex flex-col h-full">
      {/* View tabs */}
      <div className="flex items-center border-b border-border bg-muted/30">
        <button
          onClick={() => setActiveTab('dependencies')}
          className={cn(
            "flex items-center gap-1.5 px-4 py-2.5 text-xs font-medium transition-colors relative",
            activeTab === 'dependencies'
              ? "text-foreground"
              : "text-muted-foreground hover:text-foreground"
          )}
        >
          <Users className="w-3.5 h-3.5" />
          Dependencies
          {activeTab === 'dependencies' && (
            <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
          )}
        </button>
        <button
          onClick={() => setActiveTab('nebraska')}
          disabled={!data.nebraska || data.nebraska.length === 0}
          className={cn(
            "flex items-center gap-1.5 px-4 py-2.5 text-xs font-medium transition-colors relative",
            activeTab === 'nebraska'
              ? "text-foreground"
              : "text-muted-foreground hover:text-foreground",
            (!data.nebraska || data.nebraska.length === 0) && "opacity-50 cursor-not-allowed"
          )}
        >
          <Award className="w-3.5 h-3.5" />
          Maintainers
          {activeTab === 'nebraska' && (
            <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
          )}
        </button>
        
        {/* Collapse button in tab bar */}
        {onCollapse && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onCollapse}
            className="h-7 w-7 ml-auto mr-2"
            title="Hide panel"
          >
            <ChevronsRight className="h-3.5 w-3.5" />
          </Button>
        )}
      </div>

      {/* Root/Project card */}
      {rootNode && (
        <div className="px-4 py-3 border-b bg-muted/30">
          <div className="flex items-start gap-2.5">
            <div className="w-8 h-8 rounded-md bg-muted flex items-center justify-center flex-shrink-0">
              <Package className="w-4 h-4 text-muted-foreground" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h3 className="font-mono text-sm font-medium truncate">{rootNode.id}</h3>
                {rootNode.meta?.version && (
                  <span className="text-[10px] text-muted-foreground">v{rootNode.meta.version}</span>
                )}
                {rootNode.meta?.repo_url && (
                  <a
                    href={rootNode.meta.repo_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-[#24292e] dark:bg-[#24292e] hover:bg-[#1b1f23] dark:hover:bg-[#1b1f23] rounded-md transition-colors ml-auto"
                  >
                    <Github className="w-3.5 h-3.5 text-white" />
                    <span className="text-xs font-medium text-white">repo</span>
                  </a>
                )}
              </div>
              {(rootNode.meta?.description || rootNode.meta?.summary) && (
                <p className="text-[11px] text-muted-foreground mt-1 line-clamp-2">
                  {rootNode.meta.description || rootNode.meta.summary}
                </p>
              )}
              {/* Stats row */}
              <div className="flex items-center gap-3 mt-2 pt-2 border-t text-[11px]">
                {rootNode.meta?.repo_stars !== undefined && rootNode.meta.repo_stars > 0 && (
                  <span className="flex items-center gap-1 text-muted-foreground">
                    <Star className="h-3 w-3 text-yellow-500 fill-yellow-500" />
                    {rootNode.meta.repo_stars.toLocaleString()}
                  </span>
                )}
                <span className="text-muted-foreground tabular-nums">{stats.direct} direct</span>
                <span className="text-muted-foreground tabular-nums">{stats.total} total</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'dependencies' ? (
        <>
          {/* Dependencies header with filter tabs */}
          <div className="flex items-center gap-2 px-4 py-2 border-b border-border bg-muted/30">
            <Button
              variant={filterOption === 'all' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setFilterOption('all')}
              className="h-7 text-xs"
            >
              All ({stats.total})
            </Button>
            <Button
              variant={filterOption === 'direct' ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setFilterOption('direct')}
              className="h-7 text-xs"
            >
              Direct ({stats.direct})
            </Button>
            {stats.brittle > 0 && (
              <Button
                variant={filterOption === 'brittle' ? 'secondary' : 'ghost'}
                size="sm"
                onClick={() => setFilterOption('brittle')}
                className="h-7 text-xs"
              >
                <AlertTriangle className="w-3 h-3 mr-1" />
                Brittle ({stats.brittle})
              </Button>
            )}
            <div className="flex-1" />
            {stats.archived > 0 && (
              <span className="text-xs text-yellow-600 dark:text-yellow-400">
                {stats.archived} archived
              </span>
            )}
          </div>

          {/* Search & Sort */}
          <div className="flex gap-2 px-4 py-2.5 border-b border-border">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Filter..."
                className="h-8 pl-8 text-sm"
              />
            </div>
            <Select value={sortOption} onValueChange={(v) => setSortOption(v as SortOption)}>
              <SelectTrigger className="h-8 w-28">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="stars-desc"><span className="flex items-center gap-1"><Star className="h-3 w-3 text-yellow-500 fill-yellow-500" /> Most</span></SelectItem>
                <SelectItem value="stars-asc"><span className="flex items-center gap-1"><Star className="h-3 w-3 text-yellow-500 fill-yellow-500" /> Least</span></SelectItem>
                <SelectItem value="name-asc">A → Z</SelectItem>
                <SelectItem value="name-desc">Z → A</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Dependency list */}
          <div className="flex-1 overflow-y-auto">
            {filteredDeps.length === 0 ? (
              <p className="text-center text-muted-foreground py-8 text-sm">
                {search ? 'No matches' : 'No dependencies'}
              </p>
            ) : (
              <div className="divide-y divide-border">
                {filteredDeps.map(node => (
                  <DependencyItem
                    key={node.id}
                    node={node}
                    isDirect={directDepIds.has(node.id)}
                    isExpanded={expandedNodes.has(node.id)}
                    onToggle={() => toggleExpand(node.id)}
                    edges={data.edges}
                    rootId={rootNode?.id}
                    onHighlight={onHighlight}
                    onClearHighlight={onClearHighlight}
                    isHighlighted={highlightedPackage === node.id}
                    isSelected={selectedPackage === node.id}
                    onDeselect={() => onSelectPackage?.(null)}
                  />
                ))}
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="px-4 py-2 border-t border-border text-xs text-muted-foreground text-center bg-background">
            {filteredDeps.length} of {
              filterOption === 'direct' ? stats.direct : 
              filterOption === 'brittle' ? stats.brittle : 
              stats.total
            }
          </div>
        </>
      ) : (
        <NebraskaRanking 
          rankings={data.nebraska || []} 
          onHighlight={onHighlight}
          onClearHighlight={onClearHighlight}
        />
      )}
    </div>
  );
}

interface DependencyItemProps {
  node: GraphNode;
  isDirect: boolean;
  isExpanded: boolean;
  onToggle: () => void;
  edges: { from: string; to: string }[];
  rootId?: string;
  onHighlight?: (packageName: string) => void;
  onClearHighlight?: () => void;
  /** Whether this item is highlighted from hover */
  isHighlighted?: boolean;
  /** Whether this item is selected (clicked from SVG) */
  isSelected?: boolean;
  /** Called to deselect this item */
  onDeselect?: () => void;
}

function DependencyItem({ node, isDirect, isExpanded, onToggle, edges, rootId, onHighlight, onClearHighlight, isHighlighted, isSelected, onDeselect }: DependencyItemProps) {
  const meta = node.meta || {};
  const dependsOn = edges.filter(e => e.from === node.id).map(e => e.to);
  const dependedBy = edges.filter(e => e.to === node.id).map(e => e.from);
  const itemRef = useRef<HTMLDivElement>(null);

  // Show expanded content when manually expanded OR selected (clicked from SVG)
  const showExpanded = isExpanded || isSelected;

  // Auto-scroll into view when highlighted (hover) or selected (click)
  useEffect(() => {
    if ((isHighlighted || isSelected) && itemRef.current) {
      itemRef.current.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
  }, [isHighlighted, isSelected]);

  // Handle click - if selected, deselect; otherwise toggle expand
  const handleClick = () => {
    if (isSelected) {
      onDeselect?.();
    } else {
      onToggle();
    }
  };

  return (
    <div 
      ref={itemRef}
      className={cn(
        'transition-colors',
        showExpanded ? 'bg-background' : 'hover:bg-muted/50',
        isHighlighted && 'bg-primary/10 ring-1 ring-primary/30',
        isSelected && 'bg-primary/15 ring-2 ring-primary/50'
      )}
      onMouseEnter={() => onHighlight?.(node.id)}
      onMouseLeave={() => onClearHighlight?.()}
    >
      <button
        onClick={handleClick}
        className="w-full px-4 py-3 text-left"
      >
        <div className="flex items-center gap-3">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm font-medium text-foreground truncate">
                {node.id}
              </span>
              {meta.version && (
                <span className="px-1.5 py-0.5 text-[10px] bg-muted rounded text-muted-foreground">
                  {meta.version}
                </span>
              )}
              {isDirect && (
                <span className="flex items-center gap-0.5 px-1.5 py-0.5 text-[10px] bg-primary/10 text-primary rounded">
                  <ArrowRight className="w-2.5 h-2.5" />
                  Direct
                </span>
              )}
              {meta.repo_archived && (
                <span className="px-1.5 py-0.5 text-[10px] bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 rounded">
                  Archived
                </span>
              )}
              {node.brittle && !meta.repo_archived && (
                <span className="p-1 text-muted-foreground" title="Potentially unmaintained or at-risk">
                  <AlertTriangle className="w-3 h-3" />
                </span>
              )}
            </div>
            {meta.summary && !showExpanded && (
              <p className="text-xs text-muted-foreground truncate mt-0.5">
                {meta.summary}
              </p>
            )}
          </div>
          <div className="flex items-center gap-3 flex-shrink-0">
            {meta.repo_stars !== undefined && meta.repo_stars > 0 && (
              <span className="flex items-center gap-1 text-xs text-muted-foreground">
                <Star className="h-3 w-3 text-yellow-500 fill-yellow-500" />
                {meta.repo_stars.toLocaleString()}
              </span>
            )}
            <ChevronDown className={cn('w-4 h-4 text-muted-foreground transition-transform', isExpanded && 'rotate-180')} />
          </div>
        </div>
      </button>

      {showExpanded && (
        <div className="px-4 pb-4 space-y-3">
          {/* Description */}
          {(meta.description || meta.summary) && (
            <p className="text-sm text-muted-foreground leading-relaxed">
              {meta.description || meta.summary}
            </p>
          )}

          {/* Brittle warning */}
          {node.brittle && !meta.repo_archived && (
            <div className="flex items-start gap-2 p-2.5 bg-muted/50 border border-border rounded-lg">
              <AlertTriangle className="w-4 h-4 text-muted-foreground flex-shrink-0 mt-0.5" />
              <div className="text-xs text-muted-foreground space-y-1">
                <p className="font-medium text-foreground">Potentially at-risk dependency</p>
                <p>
                  This package may be unmaintained based on low recent activity, 
                  few maintainers, or limited community engagement. Consider evaluating 
                  alternatives or monitoring for updates.
                </p>
              </div>
            </div>
          )}

          {/* Links */}
          <div className="flex flex-wrap gap-2">
            {meta.repo_url && (
              <a
                href={meta.repo_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 px-3 py-1.5 bg-[#24292e] dark:bg-[#24292e] hover:bg-[#1b1f23] dark:hover:bg-[#1b1f23] rounded-lg transition-colors"
              >
                <Github className="w-3.5 h-3.5 text-white" />
                <span className="text-xs font-medium text-white">repo</span>
              </a>
            )}
            {meta.homepage && meta.homepage !== meta.repo_url && (
              <a
                href={meta.homepage}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium
                           bg-muted border border-border rounded-lg
                           text-muted-foreground hover:text-foreground
                           hover:border-foreground/20 transition-colors"
              >
                <ExternalLink className="w-3.5 h-3.5" />
                Site
              </a>
            )}
            {meta.license && (
              <span className="px-2.5 py-1 text-xs bg-muted border border-border rounded-lg text-muted-foreground">
                {meta.license}
              </span>
            )}
          </div>

          {/* Dependencies */}
          {(dependsOn.length > 0 || dependedBy.length > 0) && (
            <div className="space-y-2 pt-2 border-t border-border">
              {dependsOn.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1.5">
                    Depends on ({dependsOn.length})
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {dependsOn.slice(0, 8).map(dep => (
                      <span key={dep} className="px-1.5 py-0.5 text-xs font-mono bg-blue-500/10 text-blue-600 dark:text-blue-400 rounded">
                        {dep}
                      </span>
                    ))}
                    {dependsOn.length > 8 && (
                      <span className="px-1.5 py-0.5 text-xs text-muted-foreground">
                        +{dependsOn.length - 8}
                      </span>
                    )}
                  </div>
                </div>
              )}
              {dependedBy.length > 0 && (
                <div>
                  <p className="text-xs font-medium text-muted-foreground mb-1.5">
                    Required by ({dependedBy.length})
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {dependedBy.slice(0, 8).map(dep => (
                      <span 
                        key={dep} 
                        className={cn(
                          "px-1.5 py-0.5 text-xs font-mono rounded",
                          dep === rootId 
                            ? "bg-primary/10 text-primary" 
                            : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                        )}
                      >
                        {dep}
                        {dep === rootId && ' (root)'}
                      </span>
                    ))}
                    {dependedBy.length > 8 && (
                      <span className="px-1.5 py-0.5 text-xs text-muted-foreground">
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
            <p className="text-xs text-muted-foreground">
              Last commit: {new Date(meta.repo_last_commit).toLocaleDateString()}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
