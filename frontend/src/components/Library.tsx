/**
 * User's library component - saved towers and private repos.
 */

import { useState } from 'react';
import { Library as LibraryIcon, RefreshCw, Trash2, GitBranch, Box, Layers, Network, Star } from 'lucide-react';
import { toast } from 'sonner';
import { useLibrary, useRemoveFromLibrary } from '@/hooks/queries';
import { deleteRender, getRender } from '@/lib/api';
import { LanguageIcon } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
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
import type { JobResponse, LibraryItem, RepoItem, Language } from '@/types/api';

interface Props {
  onSelect: (job: JobResponse, inLibrary?: boolean) => void;
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

export function Library({ onSelect }: Props) {
  const { data, isLoading, error, refetch } = useLibrary(50);
  const { mutate: removeFromLibrary, isPending: isRemoving } = useRemoveFromLibrary();
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const handleRemovePackage = (item: LibraryItem) => {
    removeFromLibrary(
      { language: item.language, pkg: item.package },
      {
        onSuccess: () => {
          toast.success('Removed from library');
        },
        onError: (err) => {
          toast.error('Failed to remove', { description: err.message });
        },
      }
    );
  };

  const handleDeleteRepo = async (item: RepoItem) => {
    setDeletingId(item.id);
    try {
      await deleteRender(item.id);
      toast.success('Repo visualization deleted');
      refetch();
    } catch (err) {
      toast.error('Failed to delete', { 
        description: err instanceof Error ? err.message : 'Unknown error' 
      });
    } finally {
      setDeletingId(null);
    }
  };

  const handleSelectPackage = async (item: LibraryItem) => {
    const vizTypes = item.viz_types || [];
    
    // Find the first viz type with artifacts (prefer tower)
    const preferredOrder: Array<'tower' | 'nodelink'> = ['tower', 'nodelink'];
    const primaryVizType = preferredOrder
      .map(vt => vizTypes.find(v => v.viz_type === vt && v.artifact_svg))
      .find(v => v);
    
    if (!primaryVizType) {
      toast.error('No visualization available');
      return;
    }
    
    try {
      // Fetch the full render data (includes layout with Nebraska rankings)
      const jobResponse = await getRender(primaryVizType.render_id);
      
      // Add available viz types and related render IDs
      const availableVizTypes = vizTypes
        .map(v => v.viz_type)
        .filter((vt): vt is 'tower' | 'nodelink' => vt === 'tower' || vt === 'nodelink');
      
      if (jobResponse.result) {
        jobResponse.result.available_viz_types = availableVizTypes;
        jobResponse.result.related_render_ids = vizTypes.map(v => v.render_id);
      }
      
      onSelect(jobResponse, true); // Items from library are always in library
    } catch (error) {
      console.error('Failed to fetch render:', error);
      toast.error('Failed to load visualization');
    }
  };

  const handleSelectRepo = (item: RepoItem) => {
    const primaryRender = item.renders[0];
    if (!primaryRender?.artifacts) {
      toast.error('No visualization available');
      return;
    }
    
    const jobResponse: JobResponse = {
      job_id: item.id,
      status: 'completed',
      created_at: item.created_at,
      result: {
        svg: primaryRender.artifacts.svg,
        png: primaryRender.artifacts.png,
        pdf: primaryRender.artifacts.pdf,
        graph_path: item.graph_url,
        nodes: item.node_count,
        edges: item.edge_count,
        viz_type: primaryRender.viz_type as 'tower' | 'nodelink',
        available_viz_types: [primaryRender.viz_type as 'tower' | 'nodelink'],
        source: {
          language: item.source.language,
          package: item.source.repo || item.source.package,
        },
        related_render_ids: [item.id],
      }
    };
    onSelect(jobResponse); // Repos don't have library status
  };

  if (isLoading) {
    return (
      <div className="flex-1 flex flex-col max-w-3xl mx-auto w-full px-6 py-8">
        <div className="flex items-center justify-between mb-6">
          <Skeleton className="h-7 w-32" />
          <Skeleton className="h-9 w-9" />
        </div>
        <div className="space-y-2">
          {[...Array(5)].map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <p className="text-destructive mb-4">{error.message}</p>
          <Button variant="outline" onClick={() => refetch()}>
            Try again
          </Button>
        </div>
      </div>
    );
  }

  const hasPackages = data?.packages && data.packages.length > 0;
  const hasRepos = data?.repos && data.repos.length > 0;
  const isEmpty = !hasPackages && !hasRepos;

  if (isEmpty) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <div className="w-16 h-16 mx-auto mb-4 bg-muted rounded-xl flex items-center justify-center">
            <LibraryIcon className="w-8 h-8 text-muted-foreground" />
          </div>
          <p className="text-foreground font-medium">Your library is empty</p>
          <p className="text-sm text-muted-foreground mt-1">
            Visualize packages or save from explore
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col max-w-3xl mx-auto w-full px-6 py-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-lg font-semibold text-foreground">Library</h2>
          <p className="text-sm text-muted-foreground">
            {(data?.packages?.length || 0) + (data?.repos?.length || 0)} items
          </p>
        </div>
        <Button variant="ghost" size="icon" onClick={() => refetch()}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto -mx-2">
        <div className="space-y-2">
          {/* Public packages */}
          {data?.packages?.map((item) => {
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
                  onClick={() => handleSelectPackage(item)}
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
                      {formatDate(item.saved_at)}
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
                        onClick={() => handleRemovePackage(item)}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                      >
                        Remove
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            );
          })}

          {/* Private repos */}
          {data?.repos?.map((item) => {
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
                  onClick={() => handleSelectRepo(item)}
                  className="flex-1 min-w-0 text-left"
                  disabled={deletingId === item.id}
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
                      {formatDate(item.created_at)}
                    </span>
                  </div>
                </button>

                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      disabled={deletingId === item.id}
                      className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                    >
                      {deletingId === item.id ? (
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
                        onClick={() => handleDeleteRepo(item)}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                      >
                        Delete
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

