/**
 * SVG viewer component with zoom and pan functionality.
 * 
 * Displays SVG visualizations with interactive zoom/pan controls.
 * Supports both keyboard shortcuts and mouse interactions.
 */

import { useCallback, useRef, type RefObject } from 'react';
import { Hand } from 'lucide-react';
import { ZoomPanControls } from './ZoomPanControls';
import { cn } from '@/lib/utils';
import type { ZoomPanState, ZoomPanHandlers } from '@/hooks/useVisualizationZoom';

interface SvgViewerProps {
  /** SVG content to render (HTML string) */
  svgData: string | undefined;
  /** Whether SVG is currently loading */
  svgLoading: boolean;
  /** Zoom/pan state */
  zoomPanState: ZoomPanState;
  /** Zoom/pan action handlers */
  zoomPanHandlers: ZoomPanHandlers;
  /** Ref for the SVG container (for highlighting integration) */
  svgContainerRef: RefObject<HTMLDivElement | null>;
}

export function SvgViewer({
  svgData,
  svgLoading,
  zoomPanState,
  zoomPanHandlers,
  svgContainerRef,
}: SvgViewerProps) {
  const { zoom, pan, isPanning, panModeEnabled } = zoomPanState;
  const containerRef = useRef<HTMLDivElement>(null);

  // Mouse wheel zoom - only active when pan mode is enabled
  const handleWheel = useCallback((e: React.WheelEvent) => {
    if (!panModeEnabled) return;
    e.preventDefault();
    const delta = e.deltaY > 0 ? -1 : 1;
    if (delta > 0) {
      zoomPanHandlers.zoomIn();
    } else {
      zoomPanHandlers.zoomOut();
    }
  }, [panModeEnabled, zoomPanHandlers]);

  // Pan handlers - only active when pan mode is enabled
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (!panModeEnabled || e.button !== 0) return;
    zoomPanHandlers.startPan(e.clientX, e.clientY);
  }, [panModeEnabled, zoomPanHandlers]);

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (!panModeEnabled) return;
    zoomPanHandlers.updatePan(e.clientX, e.clientY);
  }, [panModeEnabled, zoomPanHandlers]);

  const handleMouseUp = useCallback(() => {
    zoomPanHandlers.endPan();
  }, [zoomPanHandlers]);

  const handleMouseLeave = useCallback(() => {
    zoomPanHandlers.endPan();
  }, [zoomPanHandlers]);

  return (
    <div className="flex-1 flex flex-col min-w-0 relative">
      {/* Zoom and pan controls */}
      <ZoomPanControls
        zoom={zoom}
        panModeEnabled={panModeEnabled}
        onZoomIn={zoomPanHandlers.zoomIn}
        onZoomOut={zoomPanHandlers.zoomOut}
        onReset={zoomPanHandlers.resetZoom}
        onTogglePanMode={zoomPanHandlers.togglePanMode}
      />

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
  );
}

