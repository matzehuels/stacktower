/**
 * Language/integration card component for displaying language support info.
 */

import { LanguageIcon } from '@/components/icons';
import { cn } from '@/lib/utils';
import type { Language } from '@/config/constants';

interface LanguageCardProps {
  /** Language identifier */
  language: Language;
  /** Display name */
  name: string;
  /** Registry name or description */
  registry: string;
  /** List of manifest files */
  manifests: Array<{ filename: string }>;
  /** Whether the card is clickable */
  onClick?: () => void;
  /** Additional className */
  className?: string;
}

export function LanguageCard({ 
  language,
  name, 
  registry, 
  manifests,
  onClick,
  className 
}: LanguageCardProps) {
  const Component = onClick ? 'button' : 'div';
  
  return (
    <Component
      onClick={onClick}
      className={cn(
        'p-4 rounded-lg border bg-card transition-colors text-left w-full',
        onClick && 'cursor-pointer hover:bg-accent/50 active:scale-[0.98]',
        className
      )}
    >
      <div className="flex items-center gap-2 mb-2">
        <LanguageIcon 
          language={language} 
          className="h-5 w-5" 
        />
        <span className="font-medium capitalize">{name}</span>
      </div>
      <p className="text-xs text-muted-foreground mb-2">
        {registry}
      </p>
      <div className="flex flex-wrap gap-1">
        {manifests.slice(0, 2).map((m) => (
          <span 
            key={m.filename}
            className="text-xs px-1.5 py-0.5 rounded bg-muted font-mono"
          >
            {m.filename}
          </span>
        ))}
        {manifests.length > 2 && (
          <span className="text-xs text-muted-foreground">
            +{manifests.length - 2}
          </span>
        )}
      </div>
    </Component>
  );
}

