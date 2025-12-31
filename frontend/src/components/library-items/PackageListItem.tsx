/**
 * Package list item for the user's library.
 * 
 * Displays a public package with:
 * - Language icon
 * - Package name
 * - Node/edge counts
 * - Available viz types (tower/nodelink)
 * - Saved timestamp
 * - Remove action with confirmation
 */

import { memo } from 'react';
import { Box, GitBranch, Layers, Network, Star, Trash2 } from 'lucide-react';
import { formatRelativeTime } from '@/lib/date';
import { LanguageIcon } from '@/components/icons';
import { Button } from '@/components/ui/button';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import type { LibraryItem, Language } from '@/types/api';

interface PackageListItemProps {
  item: LibraryItem;
  onSelect: (item: LibraryItem) => void;
  onRemove: (item: LibraryItem) => void;
  isRemoving: boolean;
}

export const PackageListItem = memo(function PackageListItem({
  item,
  onSelect,
  onRemove,
  isRemoving,
}: PackageListItemProps) {
  const hasTower = item.viz_types?.some(v => v.viz_type === 'tower') ?? false;
  const hasNodelink = item.viz_types?.some(v => v.viz_type === 'nodelink') ?? false;

  return (
    <div
      key={`${item.language}-${item.package}`}
      className="group flex items-center gap-3 px-3 py-3 border rounded-lg hover:bg-muted/50 transition-colors"
    >
      <div className="w-8 h-8 rounded-lg bg-muted flex items-center justify-center flex-shrink-0">
        <LanguageIcon 
          language={item.language as Language} 
          className="w-4 h-4" 
        />
      </div>

      <button
        onClick={() => onSelect(item)}
        className="flex-1 min-w-0 text-left"
      >
        <div className="font-mono text-sm font-medium text-foreground truncate">
          {item.package}
        </div>
        
        <div className="flex items-center gap-3 mt-0.5 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <Box className="w-3 h-3" />
            {item.node_count}
          </span>
          <span className="flex items-center gap-1">
            <GitBranch className="w-3 h-3" />
            {item.edge_count}
          </span>
          
          <span className="flex items-center gap-1">
            {hasTower && (
              <span className="flex items-center gap-0.5" title="Tower">
                <Layers className="w-3 h-3" />
              </span>
            )}
            {hasNodelink && (
              <span className="flex items-center gap-0.5" title="Node-Link">
                <Network className="w-3 h-3" />
              </span>
            )}
          </span>
          
          <span className="ml-auto flex items-center gap-1">
            <Star className="w-3 h-3" />
            {formatRelativeTime(item.saved_at)}
          </span>
        </div>
      </button>

      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            disabled={isRemoving}
            className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-destructive hover:bg-destructive/10"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove from library?</AlertDialogTitle>
            <AlertDialogDescription>
              <span className="font-mono font-medium">{item.package}</span> will be removed from your library.
              The tower will still be available in explore.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => onRemove(item)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
});

