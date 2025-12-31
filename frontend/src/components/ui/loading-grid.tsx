/**
 * Shared loading grid component for consistent skeleton loading states.
 * 
 * @example
 * <LoadingGrid count={8} />
 * <LoadingGrid count={4} columns={2} aspectRatio="16/9" />
 */

import { Skeleton } from './skeleton';
import { cn } from '@/lib/utils';

interface LoadingGridProps {
  /** Number of skeleton items to display */
  count?: number;
  /** Aspect ratio for each item (e.g., "4/3", "16/9") */
  aspectRatio?: string;
  /** Grid columns configuration (default: responsive) */
  columns?: number;
  /** Custom className for container */
  className?: string;
}

export function LoadingGrid({ 
  count = 8, 
  aspectRatio = '4/3',
  columns,
  className 
}: LoadingGridProps) {
  const gridClassName = columns 
    ? `grid-cols-${columns}` 
    : 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4';

  return (
    <div className={cn('grid gap-4', gridClassName, className)}>
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton 
          key={i} 
          className="rounded-lg"
          style={{ aspectRatio }}
        />
      ))}
    </div>
  );
}


