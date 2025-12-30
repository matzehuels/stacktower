/**
 * Nebraska Guy Ranking - displays the most critical maintainers.
 */

import { useState } from 'react';
import { User, ChevronDown, Github, Package, Award } from 'lucide-react';
import type { NebraskaRanking as NebraskaRankingType } from '@/types/api';
import { cn } from '@/lib/utils';

interface NebraskaRankingProps {
  rankings: NebraskaRankingType[];
  /** Called when hovering over a package - passes the package name */
  onHighlight?: (packageName: string) => void;
  /** Called when hover ends */
  onClearHighlight?: () => void;
}

export function NebraskaRanking({ rankings, onHighlight, onClearHighlight }: NebraskaRankingProps) {
  const [expandedMaintainers, setExpandedMaintainers] = useState<Set<string>>(new Set());

  const toggleExpand = (maintainer: string) => {
    setExpandedMaintainers(prev => {
      const next = new Set(prev);
      if (next.has(maintainer)) {
        next.delete(maintainer);
      } else {
        next.add(maintainer);
      }
      return next;
    });
  };

  if (!rankings || rankings.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="text-center text-muted-foreground">
          <Award className="w-12 h-12 mx-auto mb-3 opacity-50" />
          <p className="text-sm">No maintainer data available</p>
        </div>
      </div>
    );
  }

  // Only show top 5
  const topRankings = rankings.slice(0, 5);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="px-4 py-3 border-b bg-muted/30">
        <div className="flex items-center gap-2 mb-2">
          <Award className="w-4 h-4 text-muted-foreground" />
          <h3 className="text-sm font-medium">Maintainers</h3>
        </div>
        <p className="text-[11px] text-muted-foreground leading-relaxed">
          Ranked by dependency depth—maintainers of foundational packages that many others depend on.
        </p>
      </div>

      {/* Rankings list */}
      <div className="flex-1 overflow-y-auto">
        <div className="divide-y divide-border">
          {topRankings.map((ranking, index) => (
            <MaintainerItem
              key={ranking.maintainer}
              ranking={ranking}
              rank={index + 1}
              isExpanded={expandedMaintainers.has(ranking.maintainer)}
              onToggle={() => toggleExpand(ranking.maintainer)}
              onHighlight={onHighlight}
              onClearHighlight={onClearHighlight}
            />
          ))}
        </div>
      </div>

      {/* Footer */}
      <div className="px-4 py-2 border-t border-border text-xs text-muted-foreground text-center bg-background">
        Top {topRankings.length} maintainers
      </div>
    </div>
  );
}

interface MaintainerItemProps {
  ranking: NebraskaRankingType;
  rank: number;
  isExpanded: boolean;
  onToggle: () => void;
  onHighlight?: (packageName: string) => void;
  onClearHighlight?: () => void;
}

function MaintainerItem({ ranking, rank, isExpanded, onToggle, onHighlight, onClearHighlight }: MaintainerItemProps) {
  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'owner':
        return 'bg-violet-500/10 text-violet-600 dark:text-violet-400';
      case 'lead':
        return 'bg-blue-500/10 text-blue-600 dark:text-blue-400';
      default:
        return 'bg-muted text-muted-foreground';
    }
  };

  const getRankColor = (rank: number) => {
    if (rank === 1) return 'text-yellow-600 dark:text-yellow-400';
    return 'text-muted-foreground';
  };

  // When hovering over maintainer, highlight all their packages in the SVG
  const handleMaintainerHover = () => {
    if (!onHighlight) return;
    
    // Highlight all packages by adding the highlight class to their SVG blocks
    ranking.packages.forEach(pkg => {
      const blockElement = document.getElementById(`block-${pkg.package}`);
      if (blockElement) {
        blockElement.classList.add('highlight');
      }
    });
  };

  const handleMaintainerLeave = () => {
    if (!onClearHighlight) return;
    
    // Remove highlight from all packages
    ranking.packages.forEach(pkg => {
      const blockElement = document.getElementById(`block-${pkg.package}`);
      if (blockElement) {
        blockElement.classList.remove('highlight');
      }
    });
    onClearHighlight();
  };

  return (
    <div 
      className={cn(
        'transition-colors',
        isExpanded ? 'bg-background' : 'hover:bg-muted/50'
      )}
      onMouseEnter={handleMaintainerHover}
      onMouseLeave={handleMaintainerLeave}
    >
      <button
        onClick={onToggle}
        className="w-full px-4 py-3 text-left"
      >
        <div className="flex items-center gap-3">
          {/* Rank */}
          <div className={cn('w-6 text-center font-bold text-sm tabular-nums', getRankColor(rank))}>
            #{rank}
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <User className="w-3.5 h-3.5 text-muted-foreground flex-shrink-0" />
              <span className="font-mono text-sm font-medium text-foreground truncate">
                {ranking.maintainer}
              </span>
              <span className="px-1.5 py-0.5 text-[10px] bg-muted rounded text-muted-foreground tabular-nums">
                {ranking.packages.length} {ranking.packages.length === 1 ? 'package' : 'packages'}
              </span>
            </div>
            {!isExpanded && (
              <p className="text-xs text-muted-foreground truncate mt-0.5">
                Score: {ranking.score.toFixed(2)}
              </p>
            )}
          </div>

          <ChevronDown className={cn('w-4 h-4 text-muted-foreground transition-transform', isExpanded && 'rotate-180')} />
        </div>
      </button>

      {isExpanded && (
        <div className="px-4 pb-4 space-y-3">
          {/* Header Section with Profile Link and Score */}
          <div className="flex items-center justify-between gap-3 pb-2 border-b">
            <a
              href={`https://github.com/${ranking.maintainer}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-3 py-1.5 bg-[#24292e] dark:bg-[#24292e] hover:bg-[#1b1f23] dark:hover:bg-[#1b1f23] rounded-md transition-colors"
            >
              <Github className="w-3.5 h-3.5 text-white" />
              <span className="text-xs font-medium text-white">profile</span>
            </a>
            <div className="flex items-center gap-2 text-sm">
              <span className="text-muted-foreground text-xs">Score:</span>
              <span className="font-mono font-semibold tabular-nums">{ranking.score.toFixed(2)}</span>
            </div>
          </div>

          {/* Packages */}
          <div>
            <p className="text-xs font-medium text-muted-foreground mb-2">
              Maintains ({ranking.packages.length})
            </p>
            <div className="space-y-1.5">
              {ranking.packages.map((pkg) => (
                <div
                  key={pkg.package}
                  className="flex items-center gap-2 p-2 bg-muted/50 rounded-md hover:bg-muted transition-colors"
                  onMouseEnter={() => onHighlight?.(pkg.package)}
                  onMouseLeave={() => onClearHighlight?.()}
                >
                  <Package className="w-3 h-3 text-muted-foreground flex-shrink-0" />
                  <span className="flex-1 font-mono text-xs truncate">{pkg.package}</span>
                  <span className={cn('px-1.5 py-0.5 text-[10px] rounded', getRoleBadgeColor(pkg.role))}>
                    {pkg.role}
                  </span>
                  {pkg.depth !== undefined && (
                    <span className="text-[10px] text-muted-foreground tabular-nums">
                      depth {pkg.depth}
                    </span>
                  )}
                  {pkg.url && (
                    <a
                      href={pkg.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 px-2 py-0.5 bg-[#24292e] dark:bg-[#24292e] hover:bg-[#1b1f23] dark:hover:bg-[#1b1f23] rounded transition-colors"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <Github className="w-3 h-3 text-white" />
                      <span className="text-[10px] font-medium text-white">repo</span>
                    </a>
                  )}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

