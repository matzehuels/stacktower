/**
 * Floating overlay controls for zoom and pan operations.
 * 
 * Provides:
 * - Mode toggle (select vs pan)
 * - Zoom in/out buttons
 * - Current zoom level display
 * - Reset/fit button
 */

import { ZoomIn, ZoomOut, RotateCcw, Hand, MousePointer2 } from 'lucide-react';
import { Button } from '@/components/ui/button';

export interface ZoomPanControlsProps {
  zoom: number;
  panModeEnabled: boolean;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onReset: () => void;
  onTogglePanMode: () => void;
}

export function ZoomPanControls({
  zoom,
  panModeEnabled,
  onZoomIn,
  onZoomOut,
  onReset,
  onTogglePanMode,
}: ZoomPanControlsProps) {
  return (
    <div className="absolute top-3 left-1/2 -translate-x-1/2 z-10 flex items-center gap-0.5 bg-card/95 backdrop-blur border rounded-md p-0.5 shadow-sm">
      {/* Mode selection */}
      <Button
        variant={!panModeEnabled ? 'secondary' : 'ghost'}
        size="icon"
        onClick={() => onTogglePanMode()}
        className="h-7 w-7"
        title="Select mode"
      >
        <MousePointer2 className="h-3.5 w-3.5" />
      </Button>
      <Button
        variant={panModeEnabled ? 'secondary' : 'ghost'}
        size="icon"
        onClick={() => onTogglePanMode()}
        className="h-7 w-7"
        title="Pan mode"
      >
        <Hand className="h-3.5 w-3.5" />
      </Button>
      
      <div className="w-px h-4 bg-border mx-0.5" />
      
      {/* Zoom controls */}
      <Button
        variant="ghost"
        size="icon"
        onClick={onZoomOut}
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
        onClick={onZoomIn}
        disabled={zoom >= 300}
        className="h-7 w-7"
        title="Zoom in"
      >
        <ZoomIn className="h-3.5 w-3.5" />
      </Button>
      
      <div className="w-px h-4 bg-border mx-0.5" />
      
      {/* Reset button */}
      <Button
        variant="ghost"
        size="icon"
        onClick={onReset}
        className="h-7 w-7"
        title="Reset view"
      >
        <RotateCcw className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

