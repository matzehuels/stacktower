/**
 * TowerCard component for explore/discovery grid.
 * Clean, minimal card design.
 */

import { memo } from 'react';
import { Layers, GitBranch, Users, Bookmark, BookmarkPlus } from 'lucide-react';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';
import { formatRelativeTime } from '@/lib/date';
import { LanguageIcon } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { useArtifact, useSaveToLibrary, useRemoveFromLibrary } from '@/hooks/queries';
import type { ExploreEntry } from '@/types/api';
import type { Language } from '@/config/constants';

interface TowerCardProps {
  entry: ExploreEntry;
  onClick?: () => void;
  className?: string;
  isAuthenticated?: boolean;
}

export const TowerCard = memo(function TowerCard({ entry, onClick, className, isAuthenticated = true }: TowerCardProps) {
  const towerViz = entry.viz_types.find((v) => v.viz_type === 'tower');
  const previewViz = towerViz || entry.viz_types[0];
  
  const artifactId = previewViz?.artifact_svg?.replace('/api/v1/artifacts/', '');
  const { data: svgData } = useArtifact(artifactId);

  const { mutate: saveToLibrary, isPending: isSaving } = useSaveToLibrary();
  const { mutate: removeFromLibrary, isPending: isRemoving } = useRemoveFromLibrary();

  const language = entry.source.language as Language;
  const packageName = entry.source.package || 'Unknown';
  const timeAgo = formatRelativeTime(entry.created_at);

  const hasTower = entry.viz_types.some((v) => v.viz_type === 'tower');
  const hasNodelink = entry.viz_types.some((v) => v.viz_type === 'nodelink');

  const handleBookmarkClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    
    if (entry.in_library) {
      removeFromLibrary(
        { language: entry.source.language, pkg: packageName },
        {
          onSuccess: () => toast.success('Removed from library'),
          onError: (err) => toast.error('Failed to remove', { description: err.message }),
        }
      );
    } else {
      saveToLibrary(
        { language: entry.source.language, pkg: packageName },
        {
          onSuccess: () => toast.success('Saved to library'),
          onError: (err) => toast.error('Failed to save', { description: err.message }),
        }
      );
    }
  };

  return (
    <div
      onClick={onClick}
      className={cn(
        'group relative flex flex-col rounded-lg border bg-card overflow-hidden',
        'hover:border-foreground/20 transition-colors cursor-pointer',
        className
      )}
    >
      {/* Preview area */}
      <div className="relative aspect-[4/3] bg-muted/30 overflow-hidden">
        {svgData ? (
          <img
            src={`data:image/svg+xml;base64,${btoa(unescape(encodeURIComponent(svgData)))}`}
            alt={packageName}
            className="absolute inset-0 w-full h-full object-contain p-2"
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="w-12 h-12 rounded bg-muted animate-pulse" />
          </div>
        )}
        
        {/* Bookmark button - only show for authenticated users */}
        {isAuthenticated && (
          <div className="absolute top-2 right-2">
            <Button
              variant="ghost"
              size="icon"
              onClick={handleBookmarkClick}
              disabled={isSaving || isRemoving}
              className={cn(
                'h-7 w-7 rounded-md bg-background/80 backdrop-blur-sm border',
                'opacity-0 group-hover:opacity-100 transition-opacity',
                entry.in_library && 'opacity-100'
              )}
              title={entry.in_library ? 'Remove from library' : 'Save to library'}
            >
              {isSaving || isRemoving ? (
                <span className="h-3.5 w-3.5 border-2 border-current rounded-full border-t-transparent animate-spin" />
              ) : entry.in_library ? (
                <Bookmark className="h-3.5 w-3.5 fill-current" />
              ) : (
                <BookmarkPlus className="h-3.5 w-3.5" />
              )}
            </Button>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="flex flex-col gap-1.5 p-3">
        <div className="flex items-center gap-1.5 min-w-0">
          <LanguageIcon language={language} className="h-4 w-4 shrink-0" />
          <span className="font-mono text-sm truncate">
            {packageName}
          </span>
        </div>

        <div className="flex items-center justify-between text-[11px] text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <span>{entry.node_count} deps</span>
            <span>·</span>
            {hasTower && (
              <span className="flex items-center gap-0.5">
                <Layers className="h-2.5 w-2.5" />
              </span>
            )}
            {hasNodelink && (
              <span className="flex items-center gap-0.5">
                <GitBranch className="h-2.5 w-2.5" />
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            {entry.popularity > 0 && (
              <span className="flex items-center gap-0.5" title="Users with this in library">
                <Users className="h-2.5 w-2.5" />
                {entry.popularity}
              </span>
            )}
            <span>{timeAgo}</span>
          </div>
        </div>
      </div>
    </div>
  );
});
