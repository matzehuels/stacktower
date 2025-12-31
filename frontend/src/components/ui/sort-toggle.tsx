/**
 * Generic sort toggle component for switching between sort options.
 * 
 * @example
 * <SortToggle
 *   value="popular"
 *   onChange={setSortBy}
 *   options={[
 *     { value: 'popular', label: 'Popular', icon: <TrendingUp /> },
 *     { value: 'recent', label: 'Recent', icon: <Clock /> }
 *   ]}
 * />
 */

import { cn } from '@/lib/utils';

export interface SortOption<T extends string> {
  value: T;
  label: string;
  icon?: React.ReactNode;
}

interface SortToggleProps<T extends string> {
  value: T;
  onChange: (value: T) => void;
  options: SortOption<T>[];
  className?: string;
}

export function SortToggle<T extends string>({ 
  value, 
  onChange, 
  options,
  className 
}: SortToggleProps<T>) {
  return (
    <div className={cn('flex gap-0.5 p-0.5 bg-muted rounded-md', className)}>
      {options.map((opt) => (
        <button
          key={opt.value}
          onClick={() => onChange(opt.value)}
          className={cn(
            'px-2.5 py-1 text-xs font-medium rounded transition-colors whitespace-nowrap flex items-center gap-1',
            value === opt.value
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          )}
        >
          {opt.icon}
          {opt.label}
        </button>
      ))}
    </div>
  );
}

