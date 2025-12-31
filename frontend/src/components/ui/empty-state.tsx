/**
 * Shared empty state component for consistent "no data" UI.
 * 
 * @example
 * <EmptyState
 *   icon={<Compass className="w-8 h-8" />}
 *   title="No towers yet"
 *   description="Be the first to create one!"
 *   action={<Button onClick={onCreate}>Create Tower</Button>}
 * />
 */

interface EmptyStateProps {
  /** Icon to display (optional) */
  icon?: React.ReactNode;
  /** Main title text */
  title: string;
  /** Optional description text */
  description?: string;
  /** Optional action button or element */
  action?: React.ReactNode;
  /** Custom className for container */
  className?: string;
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={className || "flex items-center justify-center h-64"}>
      <div className="text-center">
        {icon && <div className="w-10 h-10 mx-auto text-muted-foreground mb-3">{icon}</div>}
        <p className="font-medium">{title}</p>
        {description && <p className="text-sm text-muted-foreground mt-1">{description}</p>}
        {action && <div className="mt-4">{action}</div>}
      </div>
    </div>
  );
}


