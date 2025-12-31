/**
 * Shared language filter component with icon buttons.
 */

import { LanguageIcon } from '@/components/icons';
import { LANGUAGES } from '@/config/constants';
import { cn } from '@/lib/utils';
import type { Language } from '@/config/constants';

interface LanguageFilterProps {
  /** Currently selected language */
  value: Language | 'all';
  /** Callback when language changes */
  onChange: (language: Language | 'all') => void;
  /** Size variant */
  size?: 'sm' | 'default';
  /** Additional className */
  className?: string;
}

export function LanguageFilter({ 
  value, 
  onChange, 
  size = 'default',
  className 
}: LanguageFilterProps) {
  const isSmall = size === 'sm';
  
  return (
    <div className={cn(
      'flex gap-0.5 p-0.5 bg-muted rounded-md overflow-x-auto shrink-0',
      className
    )}>
      <button
        onClick={() => onChange('all')}
        className={cn(
          'font-medium rounded transition-colors whitespace-nowrap cursor-pointer',
          isSmall ? 'px-2 py-0.5 text-xs' : 'px-2.5 py-1 text-xs',
          value === 'all'
            ? 'bg-background text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground'
        )}
      >
        All
      </button>
      {LANGUAGES.map((lang) => (
        <button
          key={lang.value}
          onClick={() => onChange(lang.value)}
          className={cn(
            'rounded transition-colors whitespace-nowrap cursor-pointer',
            isSmall ? 'p-1' : 'p-1.5',
            value === lang.value
              ? 'bg-background shadow-sm'
              : 'text-muted-foreground hover:text-foreground'
          )}
          title={lang.label}
        >
          <LanguageIcon 
            language={lang.value} 
            className={isSmall ? 'h-3 w-3' : 'h-3.5 w-3.5'} 
          />
        </button>
      ))}
    </div>
  );
}

