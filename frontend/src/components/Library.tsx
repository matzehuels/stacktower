/**
 * User's library component - saved towers and private repos.
 */

import { useState } from 'react';
import { Library as LibraryIcon, RefreshCw } from 'lucide-react';
import { toast } from 'sonner';
import { useLibrary, useRemoveFromLibrary } from '@/hooks/queries';
import { deleteRender, getRender } from '@/lib/api';
import { Button, EmptyState, LoadingGrid } from '@/components/ui';
import { PackageListItem, RepoListItem } from '@/components/library-items';
import type { JobResponse, LibraryItem, RepoItem } from '@/types/api';

interface LibraryProps {
  onSelect: (job: JobResponse, inLibrary?: boolean) => void;
}

export function Library({ onSelect }: LibraryProps) {
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
        <LoadingGrid count={5} columns={1} aspectRatio="8/1" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-muted-foreground mb-4">
            Failed to load your library. Please try again.
          </p>
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
      <EmptyState
        icon={
          <div className="w-16 h-16 mx-auto bg-muted rounded-xl flex items-center justify-center">
            <LibraryIcon className="w-8 h-8 text-muted-foreground" />
          </div>
        }
        title="Your library is empty"
        description="Visualize packages or save from explore"
        className="flex-1 flex items-center justify-center"
      />
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
          {data?.packages?.map((item) => (
            <PackageListItem
              key={`${item.language}-${item.package}`}
              item={item}
              onSelect={handleSelectPackage}
              onRemove={handleRemovePackage}
              isRemoving={isRemoving}
            />
          ))}

          {/* Private repos */}
          {data?.repos?.map((item) => (
            <RepoListItem
              key={item.id}
              item={item}
              onSelect={handleSelectRepo}
              onDelete={handleDeleteRepo}
              isDeleting={deletingId === item.id}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

