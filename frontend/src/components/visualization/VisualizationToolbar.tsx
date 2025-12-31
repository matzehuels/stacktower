/**
 * Toolbar for visualization view with back navigation, viz type selector,
 * share, export, and delete/bookmark actions.
 */

import { ArrowLeft, Trash2, Download, Layers, Network, Bookmark, Share2, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { getArtifactUrl } from '@/hooks/queries';
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
import { cn } from '@/lib/utils';
import type { JobResponse, VizType } from '@/types/api';

export interface VisualizationToolbarProps {
  job: JobResponse;
  currentVizType: VizType;
  isPublicPackage: boolean;
  isInLibrary: boolean;
  isDeleting: boolean;
  isAuthenticated: boolean;
  justCopied: boolean;
  onReset: () => void;
  onVizTypeChange?: (vizType: VizType) => void;
  onShare: (jobId: string) => void;
  onDelete: () => void;
  onSaveToLibrary: () => void;
  onRemoveFromLibrary: () => void;
}

export function VisualizationToolbar({
  job,
  currentVizType,
  isPublicPackage,
  isInLibrary,
  isDeleting,
  isAuthenticated,
  justCopied,
  onReset,
  onVizTypeChange,
  onShare,
  onDelete,
  onSaveToLibrary,
  onRemoveFromLibrary,
}: VisualizationToolbarProps) {
  const svgPath = job.result?.svg;
  const graphPath = job.result?.graph_path;

  return (
    <div className="border-b bg-card">
      <div className="h-10 px-3 flex items-center justify-between">
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
          {/* Share button */}
          {job.job_id && job.job_id !== 'cached' && job.status === 'completed' && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onShare(job.job_id)}
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
                      className="cursor-pointer"
                    >
                      SVG
                    </a>
                  </DropdownMenuItem>
                )}
                {job.result?.png && (
                  <DropdownMenuItem asChild>
                    <a
                      href={getArtifactUrl(job.result.png)}
                      download
                      className="cursor-pointer"
                    >
                      PNG
                    </a>
                  </DropdownMenuItem>
                )}
                {job.result?.pdf && (
                  <DropdownMenuItem asChild>
                    <a
                      href={getArtifactUrl(job.result.pdf)}
                      download
                      className="cursor-pointer"
                    >
                      PDF
                    </a>
                  </DropdownMenuItem>
                )}
                {graphPath && (
                  <DropdownMenuItem asChild>
                    <a
                      href={getArtifactUrl(graphPath)}
                      download
                      className="cursor-pointer"
                    >
                      JSON
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
                onClick={isInLibrary ? onRemoveFromLibrary : onSaveToLibrary}
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
                      onClick={onDelete}
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
  );
}

