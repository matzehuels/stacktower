/**
 * Hook for managing bidirectional highlighting between SVG visualization
 * and dependency list.
 * 
 * Supports:
 * - Hover highlighting (synced between SVG and list)
 * - Click selection (from SVG elements)
 * - Auto-scroll to selected items in list
 * - Works with tower (.block), nodelink, and graphviz (.node) visualizations
 */

import { useState, useEffect, useCallback } from 'react';

export function useSvgHighlighting(
  svgContainerRef: React.RefObject<HTMLDivElement | null>,
  svgData: string | undefined
) {
  const [hoveredPackage, setHoveredPackage] = useState<string | null>(null);
  const [selectedPackage, setSelectedPackage] = useState<string | null>(null);

  // Helper to extract package name from various element types
  const getPackageNameFromElement = useCallback((el: Element): string | null => {
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
  }, []);

  // Set up event delegation for bidirectional highlighting and click handling
  useEffect(() => {
    const container = svgContainerRef.current;
    if (!container) return;

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

      const toBlock = findBlockElement(e.relatedTarget as Element | null);
      if (toBlock && toBlock.packageName === fromBlock.packageName) {
        return; // Still hovering same package
      }
      if (toBlock) {
        return; // Moving to different package
      }
      setHoveredPackage(null);
    };

    const handleClick = (e: MouseEvent) => {
      const found = findBlockElement(e.target as Element | null);
      if (found) {
        e.preventDefault();
        e.stopPropagation();
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
  }, [svgData, svgContainerRef, getPackageNameFromElement]);

  // Sync hoveredPackage state to SVG highlight classes
  useEffect(() => {
    if (!svgContainerRef.current) return;
    const svgElement = svgContainerRef.current.querySelector('svg');
    if (!svgElement) return;

    // Tower visualization: Apply highlight to .block and .block-text elements
    svgElement.querySelectorAll('.block').forEach((block) => {
      const blockId = block.id?.replace('block-', '');
      block.classList.toggle('highlight', hoveredPackage !== null && blockId === hoveredPackage);
    });
    svgElement.querySelectorAll('.block-text').forEach((text) => {
      const blockId = (text as HTMLElement).dataset.block;
      text.classList.toggle('highlight', hoveredPackage !== null && blockId === hoveredPackage);
    });

    // Nodelink/Graphviz: Apply highlight to .node elements
    svgElement.querySelectorAll('.node').forEach((node) => {
      const titleEl = node.querySelector('title');
      if (titleEl?.textContent) {
        const nodeName = titleEl.textContent.replace(/_sub_\d+$/, '');
        const shouldHighlight = hoveredPackage !== null && nodeName === hoveredPackage;
        
        node.classList.toggle('highlight', shouldHighlight);
        
        if (shouldHighlight) {
          const path = node.querySelector('path');
          const text = node.querySelector('text');
          if (path) (path as SVGPathElement).style.strokeWidth = '2';
          if (text) (text as SVGTextElement).style.fontWeight = 'bold';
        } else {
          const path = node.querySelector('path');
          const text = node.querySelector('text');
          if (path) (path as SVGPathElement).style.strokeWidth = '';
          if (text) (text as SVGTextElement).style.fontWeight = '';
        }
      }
    });
  }, [hoveredPackage, svgContainerRef]);

  const clearSelection = useCallback(() => {
    setSelectedPackage(null);
  }, []);

  return {
    state: {
      hoveredPackage,
      selectedPackage,
    },
    actions: {
      setHoveredPackage,
      setSelectedPackage,
      clearSelection,
    },
  };
}

