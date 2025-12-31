/**
 * Component for displaying visualization results.
 * 
 * Orchestrates the visualization view including:
 * - Job polling for status updates
 * - SVG rendering with zoom/pan controls
 * - Dependency list sidebar
 * - Share, export, and library management
 */

import { useState, useEffect, useRef } from 'react';
import { PanelRightOpen } from 'lucide-react';
import { toast } from 'sonner';
import { useArtifact, useGraphData, queryKeys, useRemoveFromLibrary, useSaveToLibrary } from '@/hooks/queries';
import { 
  useShareLink, 
  useVisualizationZoom, 
  useSvgHighlighting, 
  useJobPolling 
} from '@/hooks';
import { deleteRender, deleteRenders } from '@/lib/api';
import { parseError } from '@/lib/helpers/errors';
import { useQueryClient } from '@tanstack/react-query';
import { DependencyList } from '@/components/DependencyList';
import { ErrorPage } from '@/components/ErrorPage';
import { Skeleton } from '@/components/ui/skeleton';
import { VisualizationToolbar, SvgViewer } from '@/components/visualization';
import type { JobResponse, VizType } from '@/types/api';
import { cn } from '@/lib/utils';

interface Props {
  job: JobResponse;
  onReset: () => void;
  onJobUpdate?: (job: JobResponse) => void;
  onDelete?: () => void;
  onVizTypeChange?: (vizType: VizType) => void;
  /** Whether the package is in the user's library (for public packages) */
  inLibrary?: boolean;
  /** Whether the user is authenticated */
  isAuthenticated?: boolean;
}

export function VisualizationResult({ job, onReset, onJobUpdate, onDelete, onVizTypeChange, inLibrary: initialInLibrary, isAuthenticated = true }: Props) {
  // Current viz type from job result
  const currentVizType: VizType = job.result?.viz_type || 'tower';
  
  const svgPath = job.result?.svg;
  
  // Preserve graph path across viz type switches (graph is same for tower/nodelink)
  const [preservedGraphPath, setPreservedGraphPath] = useState<string | undefined>(job.result?.graph_path);
  useEffect(() => {
    if (job.result?.graph_path) {
      setPreservedGraphPath(job.result.graph_path);
    }
  }, [job.result?.graph_path]);
  
  const graphPath = job.result?.graph_path || preservedGraphPath;
  const { data: svgData, isLoading: svgLoading } = useArtifact(svgPath);
  const { data: graphData, isLoading: graphLoading } = useGraphData(graphPath);
  const queryClient = useQueryClient();
  const [isDeleting, setIsDeleting] = useState(false);
  const removeFromLibrary = useRemoveFromLibrary();
  const saveToLibrary = useSaveToLibrary();
  
  // Determine if this is a public package (can only be removed from library, not deleted)
  // vs private repo (can be deleted)
  const isPublicPackage = Boolean(job.result?.source?.package && !job.result?.source?.manifest);
  
  // Track library status locally (optimistic updates)
  const [isInLibrary, setIsInLibrary] = useState(initialInLibrary ?? true);
  useEffect(() => {
    if (initialInLibrary !== undefined) {
      setIsInLibrary(initialInLibrary);
    }
  }, [initialInLibrary]);

  const [showDeps, setShowDeps] = useState(true);
  const { share: handleShare, justCopied } = useShareLink();
  
  // Zoom/Pan state and handlers
  const { state: zoomPanState, handlers: zoomPanHandlers } = useVisualizationZoom(75);
  
  // SVG highlighting (bidirectional between SVG and dependency list)
  const svgContainerRef = useRef<HTMLDivElement>(null);
  const { 
    state: { hoveredPackage, selectedPackage },
    actions: { setHoveredPackage, setSelectedPackage }
  } = useSvgHighlighting(svgContainerRef, svgData);
  
  // Job polling
  useJobPolling(job, onJobUpdate);

  // Execute embedded SVG scripts after SVG is loaded
  // This is necessary because scripts in SVGs inserted via innerHTML don't execute automatically
  useEffect(() => {
    if (!svgData || !svgContainerRef.current) return;
    
    const svgElement = svgContainerRef.current.querySelector('svg');
    if (!svgElement) return;
    
    // Find and execute all script elements in the SVG
    const scripts = svgElement.querySelectorAll('script');
    scripts.forEach((oldScript) => {
      const newScript = document.createElementNS('http://www.w3.org/2000/svg', 'script');
      
      // Copy attributes
      Array.from(oldScript.attributes).forEach(attr => {
        newScript.setAttribute(attr.name, attr.value);
      });
      
      // Copy content and execute by replacing the script
      newScript.textContent = oldScript.textContent;
      oldScript.parentNode?.replaceChild(newScript, oldScript);
    });
  }, [svgData]);


  const handleSaveToLibrary = async () => {
    const source = job.result?.source;
    if (!source?.language || !source?.package) return;
    
    setIsDeleting(true);
    try {
      await saveToLibrary.mutateAsync({
        language: source.language,
        pkg: source.package,
      });
      setIsInLibrary(true);
      toast.success('Added to library');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      toast.error('Failed to save', { description: message });
    } finally {
      setIsDeleting(false);
    }
  };

  const handleRemoveFromLibrary = async () => {
    const source = job.result?.source;
    if (!source?.language || !source?.package) return;
    
    setIsDeleting(true);
    try {
      await removeFromLibrary.mutateAsync({
        language: source.language,
        pkg: source.package,
      });
      setIsInLibrary(false);
      toast.success('Removed from library');
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      onDelete?.();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      toast.error('Failed to remove', { description: message });
    } finally {
      setIsDeleting(false);
    }
  };

  const handleDelete = async () => {
    if (!job.job_id || job.job_id === 'cached') return;
    
    setIsDeleting(true);
    try {
      // If we have related render IDs, delete all viz types
      const relatedIds = job.result?.related_render_ids;
      if (relatedIds && relatedIds.length > 1) {
        await deleteRenders(relatedIds);
        toast.success('All visualizations deleted');
      } else {
        await deleteRender(job.job_id);
        toast.success('Visualization deleted');
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      onDelete?.();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Unknown error';
      toast.error('Failed to delete', { description: message });
    } finally {
      setIsDeleting(false);
    }
  };


  // Processing state
  if (job.status === 'pending' || job.status === 'processing') {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 mx-auto mb-3 border-2 border-foreground/20 rounded-full border-t-foreground animate-spin" />
          <p className="text-sm font-medium">
            {job.status === 'pending' ? 'Queued...' : 'Analyzing dependencies...'}
          </p>
        </div>
      </div>
    );
  }

  // Failed state
  if (job.status === 'failed') {
    const parsed = parseError(job.error || 'Job failed');
    return (
      <ErrorPage
        title={parsed.title}
        message={parsed.message}
        suggestion={parsed.suggestion}
        onBack={onReset}
      />
    );
  }

  // Completed state
  if (job.status === 'completed') {
    return (
      <div className="flex-1 flex flex-col min-h-0">
        {/* Toolbar */}
        <VisualizationToolbar
          job={job}
          currentVizType={currentVizType}
          isPublicPackage={isPublicPackage}
          isInLibrary={isInLibrary}
          isDeleting={isDeleting}
          isAuthenticated={isAuthenticated}
          justCopied={justCopied}
          onReset={onReset}
          onVizTypeChange={onVizTypeChange}
          onShare={handleShare}
          onDelete={handleDelete}
          onSaveToLibrary={handleSaveToLibrary}
          onRemoveFromLibrary={handleRemoveFromLibrary}
        />

        {/* Main content area */}
        <div className="flex-1 flex min-h-0">
          {/* Visualization panel */}
          <SvgViewer
            svgData={svgData}
            svgLoading={svgLoading}
            zoomPanState={zoomPanState}
            zoomPanHandlers={zoomPanHandlers}
            svgContainerRef={svgContainerRef}
          />

          {/* Dependency sidebar with collapse/expand */}
          <div className="relative flex-shrink-0">
            {/* Collapsed state - show expand tab */}
            {!showDeps && (
              <button
                onClick={() => setShowDeps(true)}
                className="absolute right-0 top-1/2 -translate-y-1/2 z-10
                           flex items-center gap-1 px-1.5 py-2
                           bg-card border border-r-0 rounded-l-md
                           text-muted-foreground hover:text-foreground hover:bg-muted
                           transition-colors shadow-sm"
                title="Show dependencies"
              >
                <PanelRightOpen className="h-3.5 w-3.5" />
                <span className="text-[10px] font-medium [writing-mode:vertical-lr] rotate-180">
                  Dependencies
                </span>
              </button>
            )}

            {/* Expanded sidebar */}
            <aside
              className={cn(
                'h-full border-l bg-card flex flex-col transition-all duration-200 overflow-hidden',
                showDeps ? 'w-72 xl:w-80' : 'w-0 border-l-0'
              )}
            >
              <div className="flex-1 overflow-hidden min-w-72 xl:min-w-80">
                {!graphPath ? (
                  <div className="h-full flex items-center justify-center text-xs text-muted-foreground p-4 text-center">
                    <p>No dependency data available</p>
                  </div>
                ) : graphLoading ? (
                  <div className="p-3 space-y-2">
                    {[...Array(5)].map((_, i) => (
                      <Skeleton key={i} className="h-12 w-full" />
                    ))}
                  </div>
                ) : graphData ? (
                  <DependencyList 
                    data={{
                      ...graphData,
                      // Nebraska rankings come from layout data in job result
                      nebraska: job.result?.layout?.nebraska || [],
                    }} 
                    onHighlight={setHoveredPackage}
                    onClearHighlight={() => setHoveredPackage(null)}
                    highlightedPackage={hoveredPackage}
                    selectedPackage={selectedPackage}
                    onSelectPackage={setSelectedPackage}
                    onCollapse={() => setShowDeps(false)}
                  />
                ) : (
                  <div className="h-full flex items-center justify-center text-xs text-muted-foreground">
                    Failed to load dependencies
                  </div>
                )}
              </div>
            </aside>
          </div>
        </div>
      </div>
    );
  }

  return null;
}

// =============================================================================
// Sub-components
// =============================================================================

