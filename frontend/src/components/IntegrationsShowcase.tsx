/**
 * IntegrationsShowcase displays supported languages, registries, and manifest files.
 * This data comes directly from the backend deps package.
 */

import { useIntegrations } from '@/hooks/queries';
import { LanguageIcon } from '@/components/icons';
import { Skeleton } from '@/components/ui/skeleton';
import { FileCode, Package } from 'lucide-react';
import { cn } from '@/lib/utils';

interface IntegrationsShowcaseProps {
  className?: string;
}

// Registry display names
const REGISTRY_NAMES: Record<string, string> = {
  pypi: 'PyPI',
  npm: 'npm',
  crates: 'crates.io',
  rubygems: 'RubyGems',
  packagist: 'Packagist',
  maven: 'Maven Central',
  go: 'Go Modules',
};

export function IntegrationsShowcase({ className }: IntegrationsShowcaseProps) {
  const { data: integrations, isLoading } = useIntegrations();

  if (isLoading) {
    return (
      <div className={cn('space-y-4', className)}>
        <div className="text-center">
          <Skeleton className="h-5 w-48 mx-auto" />
        </div>
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
          {[...Array(7)].map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-lg" />
          ))}
        </div>
      </div>
    );
  }

  if (!integrations?.languages?.length) {
    return null;
  }

  return (
    <div className={cn('space-y-6', className)}>
      <div className="text-center">
        <h3 className="text-sm font-medium text-muted-foreground">
          Supported Ecosystems
        </h3>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
        {integrations.languages.map((lang) => (
          <div
            key={lang.name}
            className="group relative p-4 rounded-lg border bg-card hover:bg-accent/50 transition-colors"
          >
            {/* Language header */}
            <div className="flex items-center gap-2 mb-3">
              <LanguageIcon 
                language={lang.name as 'python' | 'javascript' | 'rust' | 'go' | 'ruby' | 'php' | 'java'} 
                className="h-5 w-5" 
              />
              <span className="font-medium capitalize">{lang.name}</span>
            </div>

            {/* Registry */}
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-2">
              <Package className="h-3 w-3" />
              <span className="font-mono">
                {REGISTRY_NAMES[lang.registry.name] || lang.registry.name}
              </span>
            </div>

            {/* Manifest files */}
            {lang.manifests.length > 0 && (
              <div className="space-y-1">
                {lang.manifests.slice(0, 3).map((manifest) => (
                  <div
                    key={manifest.filename}
                    className="flex items-center gap-1.5 text-xs text-muted-foreground"
                  >
                    <FileCode className="h-3 w-3 shrink-0" />
                    <span className="font-mono truncate" title={manifest.filename}>
                      {manifest.filename}
                    </span>
                  </div>
                ))}
                {lang.manifests.length > 3 && (
                  <div className="text-xs text-muted-foreground pl-4">
                    +{lang.manifests.length - 3} more
                  </div>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

