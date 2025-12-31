/**
 * Individual dependency item component for the dependency list.
 * 
 * Displays detailed information about a package dependency including:
 * - Package name and version
 * - Direct dependency indicator
 * - GitHub stars and repository links
 * - Dependency relationships (depends on / required by)
 * - Brittle/at-risk warnings
 * 
 * Supports:
 * - Hover highlighting (bidirectional with SVG)
 * - Click selection (from SVG)
 * - Expand/collapse for details
 */

import { useRef, useEffect, memo } from 'react';
import { ChevronDown, ArrowRight, AlertTriangle, Star, Github, ExternalLink } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { GraphNode } from '@/types/api';

export interface DependencyItemProps {
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

export const DependencyItem = memo(function DependencyItem({ 
  node, 
  isDirect, 
  isExpanded, 
  onToggle, 
  edges, 
  rootId, 
  onHighlight, 
  onClearHighlight, 
  isHighlighted, 
  isSelected, 
  onDeselect 
}: DependencyItemProps) {
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
        showExpanded ? 'bg-muted/30' : 'hover:bg-muted/50',
        isHighlighted && 'bg-muted/70',
        isSelected && 'bg-muted/70 ring-1 ring-border'
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
});

