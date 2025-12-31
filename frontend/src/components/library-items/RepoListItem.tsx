/**
 * Repository list item for the user's library.
 * 
 * Displays a private repository visualization with:
 * - Language icon
 * - Repository name with "private" badge
 * - Node/edge counts
 * - Available viz type (tower/nodelink)
 * - Created timestamp
 * - Delete action with confirmation
 */

import { memo } from 'react';
import { Box, GitBranch, Layers, Network, Trash2 } from 'lucide-react';
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
import type { RepoItem, Language } from '@/types/api';

interface RepoListItemProps {
  item: RepoItem;
  onSelect: (item: RepoItem) => void;
  onDelete: (item: RepoItem) => void;
  isDeleting: boolean;
}

export const RepoListItem = memo(function RepoListItem({
  item,
  onSelect,
  onDelete,
  isDeleting,
}: RepoListItemProps) {
  const repoName = item.source.repo || item.source.package || 'Unknown';
  const primaryRender = item.renders[0];

  return (
    <div
      key={item.id}
      className="group flex items-center gap-3 px-3 py-3 border rounded-lg hover:bg-muted/50 transition-colors"
    >
      <div className="w-8 h-8 rounded-lg bg-muted flex items-center justify-center flex-shrink-0">
        <LanguageIcon 
          language={item.source.language as Language} 
          className="w-4 h-4" 
        />
      </div>

      <button
        onClick={() => onSelect(item)}
        className="flex-1 min-w-0 text-left"
        disabled={isDeleting}
      >
        <div className="flex items-center gap-2">
          <span className="font-mono text-sm font-medium text-foreground truncate">
            {repoName}
          </span>
          <span className="text-xs bg-muted px-1.5 py-0.5 rounded text-muted-foreground">
            private
          </span>
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
          
          {primaryRender && (
            <span className="flex items-center gap-0.5">
              {primaryRender.viz_type === 'tower' ? (
                <Layers className="w-3 h-3" />
              ) : (
                <Network className="w-3 h-3" />
              )}
            </span>
          )}
          
          <span className="ml-auto">
            {formatRelativeTime(item.created_at)}
          </span>
        </div>
      </button>

      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            disabled={isDeleting}
            className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-destructive hover:bg-destructive/10"
          >
            {isDeleting ? (
              <span className="h-4 w-4 border-2 border-current rounded-full border-t-transparent animate-spin" />
            ) : (
              <Trash2 className="h-4 w-4" />
            )}
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete visualization?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the visualization for <span className="font-mono font-medium">{repoName}</span>.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => onDelete(item)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
});

