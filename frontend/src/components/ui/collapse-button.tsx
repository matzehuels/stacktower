/**
 * Reusable collapse button for panels and sidebars.
 */

import { ChevronsLeft, ChevronsRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface CollapseButtonProps {
  onClick: () => void;
  /** Whether the panel/sidebar is collapsed */
  collapsed?: boolean;
  /** Position from the edge (e.g., '-right-3' or '-left-3') */
  position: string;
  /** Distance from top (e.g., 'top-16') */
  top: string;
  /** Title for the button */
  title?: string;
  /** Additional className */
  className?: string;
}

export function CollapseButton({ 
  onClick, 
  collapsed = false, 
  position, 
  top, 
  title,
  className 
}: CollapseButtonProps) {
  return (
    <Button
      variant="outline"
      size="icon"
      onClick={onClick}
      className={cn(
        'absolute h-5 w-5 rounded-full shadow-sm z-50',
        'bg-background border text-muted-foreground',
        'hover:text-foreground hover:bg-muted',
        position,
        top,
        className
      )}
      title={title}
    >
      {collapsed ? (
        <ChevronsLeft className="h-3 w-3" />
      ) : (
        <ChevronsRight className="h-3 w-3" />
      )}
    </Button>
  );
}

