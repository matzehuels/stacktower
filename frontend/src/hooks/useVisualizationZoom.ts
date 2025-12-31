/**
 * Hook for managing zoom and pan state for visualizations.
 * 
 * Provides zoom controls (in/out/reset) and pan functionality
 * with mouse drag support and mode toggle.
 */

import { useState, useCallback, useRef } from 'react';

export interface ZoomPanState {
  zoom: number;
  pan: { x: number; y: number };
  isPanning: boolean;
  panModeEnabled: boolean;
}

export interface ZoomPanHandlers {
  zoomIn: () => void;
  zoomOut: () => void;
  resetZoom: () => void;
  togglePanMode: () => void;
  startPan: (startX: number, startY: number) => void;
  updatePan: (currentX: number, currentY: number) => void;
  endPan: () => void;
}

export function useVisualizationZoom(initialZoom = 75) {
  const [zoom, setZoom] = useState(initialZoom);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [isPanning, setIsPanning] = useState(false);
  const [panModeEnabled, setPanModeEnabled] = useState(false);
  const panStartRef = useRef({ x: 0, y: 0 });
  const panOffsetRef = useRef({ x: 0, y: 0 });

  const zoomIn = useCallback(() => {
    setZoom(prev => Math.min(prev + 25, 200));
  }, []);

  const zoomOut = useCallback(() => {
    setZoom(prev => Math.max(prev - 25, 25));
  }, []);

  const resetZoom = useCallback(() => {
    setZoom(75);
    setPan({ x: 0, y: 0 });
  }, []);

  const togglePanMode = useCallback(() => {
    setPanModeEnabled(prev => !prev);
  }, []);

  const startPan = useCallback((startX: number, startY: number) => {
    setIsPanning(true);
    panStartRef.current = { x: startX, y: startY };
    panOffsetRef.current = pan;
  }, [pan]);

  const updatePan = useCallback((currentX: number, currentY: number) => {
    if (!isPanning) return;
    
    const deltaX = currentX - panStartRef.current.x;
    const deltaY = currentY - panStartRef.current.y;
    setPan({
      x: panOffsetRef.current.x + deltaX,
      y: panOffsetRef.current.y + deltaY,
    });
  }, [isPanning]);

  const endPan = useCallback(() => {
    setIsPanning(false);
  }, []);

  const state: ZoomPanState = {
    zoom,
    pan,
    isPanning,
    panModeEnabled,
  };

  const handlers: ZoomPanHandlers = {
    zoomIn,
    zoomOut,
    resetZoom,
    togglePanMode,
    startPan,
    updatePan,
    endPan,
  };

  return { state, handlers };
}

