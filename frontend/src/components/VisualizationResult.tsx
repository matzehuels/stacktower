/**
 * Component for displaying visualization results.
 * Clean, minimal toolbar design.
 */

import { useState, useEffect, useRef, useCallback } from 'react';
import { ArrowLeft, Trash2, Download, ZoomIn, ZoomOut, PanelRightOpen, RotateCcw, Hand, MousePointer2, Layers, Network, Bookmark, Share2, Check } from 'lucide-react';
import { toast } from 'sonner';
import { useArtifact, getArtifactUrl, useGraphData, queryKeys, useRemoveFromLibrary, useSaveToLibrary } from '@/hooks/queries';
import { getJob, deleteRender, deleteRenders } from '@/lib/api';
import { useQueryClient } from '@tanstack/react-query';
import { DependencyList } from '@/components/DependencyList';
import { ErrorPage } from '@/components/ErrorPage';
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
import { JOB_POLL_INTERVAL } from '@/config/constants';
import type { JobResponse, VizType } from '@/types/api';
import { cn } from '@/lib/utils';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

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
  const isPublicPackage = job.result?.source?.package && !job.result?.source?.manifest;
  
  // Track library status locally (optimistic updates)
  // Default to true for packages rendered from the packages tab (they auto-add to library)
  const [isInLibrary, setIsInLibrary] = useState(initialInLibrary ?? true);
  
  // Sync with prop changes
  useEffect(() => {
    if (initialInLibrary !== undefined) {
      setIsInLibrary(initialInLibrary);
    }
  }, [initialInLibrary]);

  const [showDeps, setShowDeps] = useState(true);
  const [zoom, setZoom] = useState(75); // percentage - default to 75% for better overview
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isPanning, setIsPanning] = useState(false);
  const [panStart, setPanStart] = useState({ x: 0, y: 0 });
  const [panModeEnabled, setPanModeEnabled] = useState(false); // Toggle for pan mode
  const [hoveredPackage, setHoveredPackage] = useState<string | null>(null); // Track hovered package
  const [selectedPackage, setSelectedPackage] = useState<string | null>(null); // Track clicked package from SVG
  const [justCopied, setJustCopied] = useState(false); // Track if share link was just copied
  const pollingRef = useRef<number | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const svgContainerRef = useRef<HTMLDivElement>(null);

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

  // Set up event delegation for bidirectional highlighting and click handling (SVG → dependency list)
  // Using event delegation on the container so it works regardless of when SVG elements are created
  // Supports both tower visualizations (.block elements) and nodelink/graphviz (.node elements)
  useEffect(() => {
    const container = svgContainerRef.current;
    if (!container) return;

    // Helper to extract package name from various element types
    const getPackageNameFromElement = (el: Element): string | null => {
      // Tower visualization: .block element (id="block-{name}")
      if (el.classList?.contains('block')) {
        return el.id?.replace('block-', '') || null;
      }
      // Tower visualization: .block-text element (data-block="{name}")
      if (el.classList?.contains('block-text')) {
        return (el as HTMLElement).dataset?.block || null;
      }
      // Nodelink/Graphviz: .node element (has <title> child with package name)
      if (el.classList?.contains('node')) {
        const titleEl = el.querySelector('title');
        if (titleEl?.textContent) {
          // Remove _sub_N suffix if present (subdivider nodes)
          return titleEl.textContent.replace(/_sub_\d+$/, '');
        }
      }
      return null;
    };

    // Helper to find a node element walking up from target
    const findBlockElement = (start: Element | null): { element: Element; packageName: string } | null => {
      let target = start;
      while (target && target !== container) {
        const packageName = getPackageNameFromElement(target);
        if (packageName) {
          return { element: target, packageName };
        }
        target = target.parentElement;
      }
      return null;
    };

    const handleMouseOver = (e: MouseEvent) => {
      const found = findBlockElement(e.target as Element | null);
      if (found) {
        setHoveredPackage(found.packageName);
      }
    };

    const handleMouseOut = (e: MouseEvent) => {
      const fromBlock = findBlockElement(e.target as Element | null);
      if (!fromBlock) return;

      // Check if relatedTarget (where mouse is going) is still within the same package
      const toBlock = findBlockElement(e.relatedTarget as Element | null);
      if (toBlock && toBlock.packageName === fromBlock.packageName) {
        // Still hovering over the same package (e.g., moved from block to its text label)
        return;
      }
      if (toBlock) {
        // Moving to a different package, the mouseover will handle it
        return;
      }
      // Left the package entirely
      setHoveredPackage(null);
    };

    const handleClick = (e: MouseEvent) => {
      const found = findBlockElement(e.target as Element | null);
      if (found) {
        e.preventDefault();
        e.stopPropagation();
        // Toggle selection: if already selected, deselect; otherwise select
        setSelectedPackage(prev => prev === found.packageName ? null : found.packageName);
      }
    };

    container.addEventListener('mouseover', handleMouseOver);
    container.addEventListener('mouseout', handleMouseOut);
    container.addEventListener('click', handleClick);

    return () => {
      container.removeEventListener('mouseover', handleMouseOver);
      container.removeEventListener('mouseout', handleMouseOut);
      container.removeEventListener('click', handleClick);
    };
  }, [svgData]);

  // Sync hoveredPackage state to SVG highlight classes
  // This ensures the SVG shows visual feedback when hovering on either the SVG blocks OR the dependency list
  // Supports both tower visualizations (.block) and nodelink/graphviz (.node)
  useEffect(() => {
    if (!svgContainerRef.current) return;
    const svgElement = svgContainerRef.current.querySelector('svg');
    if (!svgElement) return;

    // Tower visualization: Apply or clear highlight classes on .block and .block-text elements
    svgElement.querySelectorAll('.block').forEach((block) => {
      const blockId = block.id?.replace('block-', '');
      block.classList.toggle('highlight', hoveredPackage !== null && blockId === hoveredPackage);
    });
    svgElement.querySelectorAll('.block-text').forEach((text) => {
      const blockId = (text as HTMLElement).dataset.block;
      text.classList.toggle('highlight', hoveredPackage !== null && blockId === hoveredPackage);
    });

    // Nodelink/Graphviz: Apply or clear highlight on .node elements
    svgElement.querySelectorAll('.node').forEach((node) => {
      const titleEl = node.querySelector('title');
      if (titleEl?.textContent) {
        const nodeName = titleEl.textContent.replace(/_sub_\d+$/, '');
        const shouldHighlight = hoveredPackage !== null && nodeName === hoveredPackage;
        
        // Add/remove highlight class to the node group
        node.classList.toggle('highlight', shouldHighlight);
        
        // For graphviz nodes, increase stroke width on the path and make text bold
        if (shouldHighlight) {
          const path = node.querySelector('path');
          const text = node.querySelector('text');
          if (path) {
            path.setAttribute('data-original-stroke-width', path.getAttribute('stroke-width') || '1');
            path.setAttribute('stroke-width', '3');
          }
          if (text) {
            text.setAttribute('font-weight', 'bold');
          }
        } else {
          // Restore original values
          const path = node.querySelector('path');
          const text = node.querySelector('text');
          if (path && path.hasAttribute('data-original-stroke-width')) {
            path.setAttribute('stroke-width', path.getAttribute('data-original-stroke-width') || '1');
            path.removeAttribute('data-original-stroke-width');
          }
          if (text) {
            text.removeAttribute('font-weight');
          }
        }
      }
    });
  }, [hoveredPackage]);

  // Highlight a package in the SVG visualization (called from dependency list hover)
  // This sets the hoveredPackage state, and the useEffect above syncs it to the SVG
  const handleHighlightPackage = useCallback((packageName: string) => {
    setHoveredPackage(packageName);
  }, []);

  // Clear all highlights in the SVG (called when leaving dependency list items)
  const handleClearHighlight = useCallback(() => {
    setHoveredPackage(null);
  }, []);

  const handleZoomIn = () => setZoom(z => Math.min(z + 25, 300));
  const handleZoomOut = () => setZoom(z => Math.max(z - 25, 25));
  const handleFitToScreen = () => {
    setZoom(75);
    setPan({ x: 0, y: 0 });
  };

  // Mouse wheel zoom - only active when pan mode is enabled
  const handleWheel = useCallback((e: React.WheelEvent) => {
    if (!panModeEnabled) return;
    e.preventDefault();
    const delta = e.deltaY > 0 ? -10 : 10;
    setZoom(z => Math.min(Math.max(z + delta, 25), 300));
  }, [panModeEnabled]);

  // Pan handlers - only active when pan mode is enabled
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (!panModeEnabled) return;
    if (e.button !== 0) return; // Only left click
    setIsPanning(true);
    setPanStart({ x: e.clientX - pan.x, y: e.clientY - pan.y });
  }, [pan, panModeEnabled]);

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (!panModeEnabled || !isPanning) return;
    setPan({
      x: e.clientX - panStart.x,
      y: e.clientY - panStart.y,
    });
  }, [isPanning, panStart, panModeEnabled]);

  const handleMouseUp = useCallback(() => {
    setIsPanning(false);
  }, []);

  const handleMouseLeave = useCallback(() => {
    setIsPanning(false);
  }, []);

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

  const handleShare = async () => {
    if (!job.job_id || job.job_id === 'cached') {
      toast.error('Cannot share this visualization', {
        description: 'Only saved visualizations can be shared.'
      });
      return;
    }

    // Generate shareable URL
    const shareUrl = `${window.location.origin}?render=${job.job_id}`;
    
    try {
      await navigator.clipboard.writeText(shareUrl);
      setJustCopied(true);
      toast.success('Share link copied!', {
        description: 'Anyone with this link can view this visualization.'
      });
      
      // Reset the "just copied" state after 2 seconds
      setTimeout(() => setJustCopied(false), 2000);
    } catch (err) {
      toast.error('Failed to copy link', {
        description: 'Please try again or copy the URL manually.'
      });
    }
  };

  // Poll for job updates when pending/processing
  const pollJob = useCallback(async () => {
    if (!job.job_id || job.job_id === 'cached') return;
    
    try {
      const jobResponse = await getJob(job.job_id);
      if (onJobUpdate) {
        onJobUpdate(jobResponse);
      }
    } catch {
      // Ignore polling errors
    }
  }, [job.job_id, onJobUpdate]);

  useEffect(() => {
    // Start polling if job is pending or processing
    if (job.status === 'pending' || job.status === 'processing') {
      pollingRef.current = window.setInterval(pollJob, JOB_POLL_INTERVAL);
    }

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    };
  }, [job.status, pollJob]);

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
    return (
      <ErrorPage
        title="Failed to build tower"
        message="We couldn't build your dependency visualization. The package might not exist or there was a network issue."
        onBack={onReset}
        onRetry={onReset}
      />
    );
  }

  // Completed state
  if (job.status === 'completed') {
    return (
      <div className="flex-1 flex flex-col min-h-0">
        {/* Top bar with actions */}
        <div className="h-12 border-b bg-card shrink-0">
          <div className="h-full px-4 flex items-center justify-between">
            {/* Left side: Back button + Viz type */}
            <div className="flex items-center gap-2">
              {/* Back button */}
              <Button variant="ghost" size="sm" onClick={onReset} className="h-7 px-2 gap-1.5">
                <ArrowLeft className="h-3.5 w-3.5" />
                <span className="hidden sm:inline text-xs">Back</span>
              </Button>

              <div className="w-px h-5 bg-border" />

              {/* Viz type selector - only for authenticated users */}
              {isAuthenticated ? (
                <Select 
                  value={currentVizType} 
                  onValueChange={(v) => onVizTypeChange?.(v as VizType)}
                  disabled={!onVizTypeChange}
                >
                  <SelectTrigger className="w-32 h-7 text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent position="popper" sideOffset={4}>
                    <SelectItem value="tower" className="text-xs">
                      <div className="flex items-center gap-1.5">
                        <Layers className="h-3.5 w-3.5" />
                        <span>Tower</span>
                      </div>
                    </SelectItem>
                    <SelectItem value="nodelink" className="text-xs">
                      <div className="flex items-center gap-1.5">
                        <Network className="h-3.5 w-3.5" />
                        <span>Node Link</span>
                      </div>
                    </SelectItem>
                  </SelectContent>
                </Select>
              ) : (
                <div className="flex items-center gap-1.5 px-2 py-1 text-xs text-muted-foreground">
                  <Layers className="h-3.5 w-3.5" />
                  <span>Tower</span>
                </div>
              )}
            </div>

            {/* Right side: Actions */}
            <div className="flex items-center gap-1">
              {/* Share button - show for all completed renders */}
              {job.job_id && job.job_id !== 'cached' && job.status === 'completed' && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleShare}
                  className="h-7 px-2 gap-1.5 text-xs"
                  title="Share this visualization"
                >
                  {justCopied ? (
                    <Check className="h-3.5 w-3.5 text-green-600 dark:text-green-400" />
                  ) : (
                    <Share2 className="h-3.5 w-3.5" />
                  )}
                  <span className="hidden sm:inline">
                    {justCopied ? 'Copied!' : 'Share'}
                  </span>
                </Button>
              )}
              
              {/* Export dropdown */}
              {(svgPath || job.result?.png || job.result?.pdf || graphPath) && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 px-2 gap-1.5 text-xs"
                      title="Export visualization"
                    >
                      <Download className="h-3.5 w-3.5" />
                      <span className="hidden sm:inline">Export</span>
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-32">
                    {svgPath && (
                      <DropdownMenuItem asChild>
                        <a
                          href={getArtifactUrl(svgPath)}
                          download
                          className="flex items-center gap-2 cursor-pointer"
                        >
                          <Download className="h-3.5 w-3.5" />
                          <span>SVG</span>
                        </a>
                      </DropdownMenuItem>
                    )}
                    {job.result?.png && (
                      <DropdownMenuItem asChild>
                        <a
                          href={getArtifactUrl(job.result.png)}
                          download
                          className="flex items-center gap-2 cursor-pointer"
                        >
                          <Download className="h-3.5 w-3.5" />
                          <span>PNG</span>
                        </a>
                      </DropdownMenuItem>
                    )}
                    {job.result?.pdf && (
                      <DropdownMenuItem asChild>
                        <a
                          href={getArtifactUrl(job.result.pdf)}
                          download
                          className="flex items-center gap-2 cursor-pointer"
                        >
                          <Download className="h-3.5 w-3.5" />
                          <span>PDF</span>
                        </a>
                      </DropdownMenuItem>
                    )}
                    {graphPath && (
                      <DropdownMenuItem asChild>
                        <a
                          href={getArtifactUrl(graphPath)}
                          download
                          className="flex items-center gap-2 cursor-pointer"
                        >
                          <Download className="h-3.5 w-3.5" />
                          <span>JSON</span>
                        </a>
                      </DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              )}

              {/* Bookmark/Delete button - only for authenticated users */}
              {isAuthenticated && job.job_id && job.job_id !== 'cached' && (
                isPublicPackage ? (
                  // Public packages: Bookmark toggle
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={isDeleting}
                    onClick={isInLibrary ? handleRemoveFromLibrary : handleSaveToLibrary}
                    className={cn(
                      'h-7 px-2 gap-1.5 text-xs',
                      isInLibrary && 'text-foreground'
                    )}
                    title={isInLibrary ? "Remove from library" : "Save to library"}
                  >
                    {isDeleting ? (
                      <span className="w-3.5 h-3.5 border-2 border-current rounded-full border-t-transparent animate-spin" />
                    ) : (
                      <Bookmark className={cn("h-3.5 w-3.5", isInLibrary && "fill-current")} />
                    )}
                    <span className="hidden sm:inline">
                      {isInLibrary ? "Saved" : "Save"}
                    </span>
                  </Button>
                ) : (
                  // Private repos: Delete (with confirmation)
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={isDeleting}
                        className="h-7 px-2 gap-1.5 text-xs text-muted-foreground hover:text-destructive"
                      >
                        {isDeleting ? (
                          <span className="w-3.5 h-3.5 border-2 border-current rounded-full border-t-transparent animate-spin" />
                        ) : (
                          <Trash2 className="h-3.5 w-3.5" />
                        )}
                        <span className="hidden sm:inline">Delete</span>
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete visualization?</AlertDialogTitle>
                        <AlertDialogDescription>
                          This will permanently delete this visualization.
                          This action cannot be undone.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={handleDelete}
                          className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                )
              )}
            </div>
          </div>
        </div>

        {/* Main content area */}
        <div className="flex-1 flex min-h-0">
          {/* Visualization panel */}
          <div className="flex-1 flex flex-col min-w-0 relative">
            {/* Zoom and pan controls */}
            <div className="absolute top-3 left-1/2 -translate-x-1/2 z-10 flex items-center gap-0.5 bg-card/95 backdrop-blur border rounded-md p-0.5 shadow-sm">
              {/* Mode selection */}
              <Button
                variant={!panModeEnabled ? 'secondary' : 'ghost'}
                size="icon"
                onClick={() => setPanModeEnabled(false)}
                className="h-7 w-7"
                title="Select mode"
              >
                <MousePointer2 className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant={panModeEnabled ? 'secondary' : 'ghost'}
                size="icon"
                onClick={() => setPanModeEnabled(true)}
                className="h-7 w-7"
                title="Pan mode"
              >
                <Hand className="h-3.5 w-3.5" />
              </Button>
              <div className="w-px h-4 bg-border mx-0.5" />
              <Button
                variant="ghost"
                size="icon"
                onClick={handleZoomOut}
                disabled={zoom <= 25}
                className="h-7 w-7"
                title="Zoom out"
              >
                <ZoomOut className="h-3.5 w-3.5" />
              </Button>
              <span className="text-[10px] font-mono text-muted-foreground w-8 text-center tabular-nums">
                {zoom}%
              </span>
              <Button
                variant="ghost"
                size="icon"
                onClick={handleZoomIn}
                disabled={zoom >= 300}
                className="h-7 w-7"
                title="Zoom in"
              >
                <ZoomIn className="h-3.5 w-3.5" />
              </Button>
              <div className="w-px h-4 bg-border mx-0.5" />
              <Button
                variant="ghost"
                size="icon"
                onClick={handleFitToScreen}
                className="h-7 w-7"
                title="Reset view"
              >
                <RotateCcw className="h-3.5 w-3.5" />
              </Button>
            </div>

            <div 
              ref={containerRef}
              className={cn(
                'flex-1 overflow-hidden bg-background relative',
                panModeEnabled && (isPanning ? 'cursor-grabbing' : 'cursor-grab')
              )}
              onWheel={handleWheel}
              onMouseDown={handleMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseLeave}
            >
              {svgLoading ? (
                <div className="h-full flex items-center justify-center">
                  <div className="w-6 h-6 border-2 border-foreground/20 rounded-full border-t-foreground animate-spin" />
                </div>
              ) : svgData ? (
                <div 
                  className="h-full w-full flex items-center justify-center select-none"
                  style={{
                    transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom / 100})`,
                    transformOrigin: 'center center',
                    transition: isPanning ? 'none' : 'transform 0.1s ease-out',
                  }}
                >
                  <div
                    ref={svgContainerRef}
                    className={cn(
                      '[&>svg]:w-auto [&>svg]:h-auto',
                      panModeEnabled && 'pointer-events-none'
                    )}
                    dangerouslySetInnerHTML={{ __html: svgData }}
                  />
                </div>
              ) : (
                <div className="h-full flex items-center justify-center text-sm text-muted-foreground">
                  Failed to load visualization
                </div>
              )}

              {/* Mode hint */}
              <div className="absolute bottom-3 left-3 text-[10px] text-muted-foreground bg-card/80 backdrop-blur px-2 py-1 rounded border">
                {panModeEnabled ? (
                  'Scroll to zoom · Drag to pan'
                ) : (
                  <>Click <Hand className="h-2.5 w-2.5 inline" /> to enable zoom & pan</>
                )}
              </div>
            </div>
          </div>

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
                    onHighlight={handleHighlightPackage}
                    onClearHighlight={handleClearHighlight}
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

