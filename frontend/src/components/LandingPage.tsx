/**
 * Clean landing page for unauthenticated users.
 * GitHub-inspired design with focus on content.
 */

import { useState, useMemo } from 'react';
import { 
  Layers, 
  Compass, 
  Github, 
  ArrowRight, 
  TrendingUp,
  Clock,
  Loader2,
} from 'lucide-react';

// X (Twitter) icon
function XIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className}>
      <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
    </svg>
  );
}

import { useExplore, usePublicStats, useIntegrations } from '@/hooks/queries';
import { getRender } from '@/lib/api';
import { toast } from 'sonner';
import { TowerCard } from '@/components/TowerCard';
import { LanguageIcon } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { LANGUAGES } from '@/config/constants';
import type { Language } from '@/config/constants';
import type { JobResponse, ExploreEntry, ExploreSortBy } from '@/types/api';
import fastapiTower from '@/assets/fastapi.svg';

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

interface Props {
  onSelect: (job: JobResponse) => void;
  onLogin: () => void;
}

export function LandingPage({ onSelect, onLogin }: Props) {
  const [selectedLanguage, setSelectedLanguage] = useState<Language | 'all'>('all');
  const [sortBy, setSortBy] = useState<ExploreSortBy>('popular');
  
  const { data: stats } = usePublicStats();
  const { data: integrations } = useIntegrations();
  
  const languageFilter = selectedLanguage === 'all' ? undefined : selectedLanguage;
  
  const {
    data: exploreData,
    isLoading: isExploring,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
  } = useExplore({ language: languageFilter, sortBy });

  const entries = useMemo(() => {
    if (!exploreData?.pages) return [];
    return exploreData.pages.flatMap((page) => page.entries);
  }, [exploreData?.pages]);

  const handleSelect = async (entry: ExploreEntry) => {
    const towerViz = entry.viz_types.find((v) => v.viz_type === 'tower');
    const selectedViz = towerViz || entry.viz_types[0];
    
    if (!selectedViz) return;

    try {
      // Fetch the full render data (includes layout with Nebraska rankings)
      const jobResponse = await getRender(selectedViz.render_id);
      
      // Add related render IDs
      const allRenderIds = entry.viz_types.map((v) => v.render_id).filter(Boolean);
      if (jobResponse.result) {
        jobResponse.result.related_render_ids = allRenderIds;
      }
      onSelect(jobResponse);
    } catch (error) {
      console.error('Failed to fetch render:', error);
      toast.error('Failed to load visualization');
    }
  };

  return (
    <div className="min-h-screen bg-background">
      {/* Navigation */}
      <nav className="sticky top-0 z-50 bg-background border-b">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 h-14 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Layers className="w-5 h-5" />
            <span className="font-semibold">Stacktower</span>
          </div>
          <div className="flex items-center gap-4">
            <a
              href="https://x.com/stacktower"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 px-2.5 py-1.5 text-sm text-muted-foreground hover:text-foreground border rounded-full transition-colors"
              title="Follow @stacktower on X"
            >
              <XIcon className="h-3.5 w-3.5" />
              <span>@stacktower</span>
            </a>
            <Button onClick={onLogin} size="sm" className="gap-1.5">
              <Github className="h-4 w-4" />
              Sign in
            </Button>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="py-12 sm:py-16 lg:py-20 px-4 sm:px-6 border-b">
        <div className="max-w-6xl mx-auto">
          <div className="grid lg:grid-cols-2 gap-8 lg:gap-12 items-center">
            {/* Left: Text content */}
            <div className="text-center lg:text-left">
              <h1 className="text-3xl sm:text-4xl font-semibold tracking-tight mb-4">
                Visualize your dependency tree
              </h1>
              
              <p className="text-lg text-muted-foreground max-w-xl mx-auto lg:mx-0 mb-6">
                Transform complex dependency trees into clear tower visualizations. 
                Understand your project's structure at a glance.
              </p>

              <div className="flex flex-col sm:flex-row items-center justify-center lg:justify-start gap-3 mb-8">
                <Button onClick={onLogin} className="gap-2">
                  <Github className="h-4 w-4" />
                  Get started
                  <ArrowRight className="h-4 w-4" />
                </Button>
                <Button variant="outline" onClick={() => {
                  document.getElementById('explore')?.scrollIntoView({ behavior: 'smooth' });
                }} className="gap-2">
                  <Compass className="h-4 w-4" />
                  Browse towers
                </Button>
              </div>

              {/* Stats */}
              {stats && (
                <div className="grid grid-cols-3 gap-4 pt-6 mt-2 border-t max-w-md mx-auto lg:mx-0">
                  <div className="text-center lg:text-left">
                    <div className="text-3xl font-bold tabular-nums tracking-tight">{stats.total_renders.toLocaleString()}</div>
                    <div className="text-sm text-muted-foreground">packages processed</div>
                  </div>
                  <div className="text-center lg:text-left">
                    <div className="text-3xl font-bold tabular-nums tracking-tight">{stats.total_dependencies.toLocaleString()}</div>
                    <div className="text-sm text-muted-foreground">dependencies analyzed</div>
                  </div>
                  <div className="text-center lg:text-left">
                    <div className="text-3xl font-bold tabular-nums tracking-tight">{stats.total_users.toLocaleString()}</div>
                    <div className="text-sm text-muted-foreground">users exploring</div>
                  </div>
                </div>
              )}
            </div>

            {/* Right: Example tower visualization */}
            <div className="relative rounded-lg border bg-card overflow-hidden max-w-sm lg:max-w-md mx-auto lg:mx-0">
              <div className="absolute top-2 left-2 flex items-center gap-1.5 px-2 py-1 bg-background/80 backdrop-blur-sm rounded text-xs text-muted-foreground border z-10">
                <LanguageIcon language="python" className="h-3.5 w-3.5" />
                <span className="font-mono">fastapi</span>
              </div>
              <div className="p-3 flex items-center justify-center max-h-[280px] overflow-hidden">
                <img 
                  src={fastapiTower} 
                  alt="FastAPI dependency tower visualization" 
                  className="w-full h-auto max-h-[260px] object-contain"
                />
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Integrations Section */}
      {integrations?.languages && (
        <section className="py-12 px-4 sm:px-6 border-b">
          <div className="max-w-4xl mx-auto">
            <div className="text-center mb-8">
              <h2 className="text-xl font-semibold mb-2">Works with your stack</h2>
              <p className="text-muted-foreground text-sm">
                Support for major package ecosystems
              </p>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
              {integrations.languages.map((lang) => (
                <div
                  key={lang.name}
                  className="p-4 rounded-lg border bg-card hover:bg-accent/50 transition-colors"
                >
                  <div className="flex items-center gap-2 mb-2">
                    <LanguageIcon 
                      language={lang.name as 'python' | 'javascript' | 'rust' | 'go' | 'ruby' | 'php' | 'java'} 
                      className="h-5 w-5" 
                    />
                    <span className="font-medium capitalize">{lang.name}</span>
                  </div>
                  <p className="text-xs text-muted-foreground mb-2">
                    {REGISTRY_NAMES[lang.registry.name] || lang.registry.name}
                  </p>
                  <div className="flex flex-wrap gap-1">
                    {lang.manifests.slice(0, 2).map((m) => (
                      <span 
                        key={m.filename}
                        className="text-xs px-1.5 py-0.5 rounded bg-muted font-mono"
                      >
                        {m.filename}
                      </span>
                    ))}
                    {lang.manifests.length > 2 && (
                      <span className="text-xs text-muted-foreground">
                        +{lang.manifests.length - 2}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      )}

      {/* Explore Section */}
      <section id="explore" className="py-12 px-4 sm:px-6">
        <div className="max-w-6xl mx-auto">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
            <div>
              <h2 className="text-xl font-semibold">Community towers</h2>
              <p className="text-sm text-muted-foreground">
                Browse visualizations from the community
              </p>
            </div>

            {/* Filter controls */}
            <div className="flex items-center gap-2">
              {/* Language filter */}
              <div className="flex gap-0.5 p-0.5 bg-muted rounded-md">
                <button
                  onClick={() => setSelectedLanguage('all')}
                  className={cn(
                    'px-2.5 py-1 text-xs font-medium rounded transition-colors',
                    selectedLanguage === 'all'
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  All
                </button>
                {LANGUAGES.map((lang) => (
                  <button
                    key={lang.value}
                    onClick={() => setSelectedLanguage(lang.value)}
                    className={cn(
                      'p-1.5 rounded transition-colors',
                      selectedLanguage === lang.value
                        ? 'bg-background shadow-sm'
                        : 'text-muted-foreground hover:text-foreground'
                    )}
                    title={lang.label}
                  >
                    <LanguageIcon language={lang.value} className="h-3.5 w-3.5" />
                  </button>
                ))}
              </div>

              {/* Sort toggle */}
              <div className="flex gap-0.5 p-0.5 bg-muted rounded-md">
                <button
                  onClick={() => setSortBy('popular')}
                  className={cn(
                    'px-2.5 py-1 text-xs font-medium rounded transition-colors flex items-center gap-1',
                    sortBy === 'popular'
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  <TrendingUp className="h-3 w-3" />
                  Popular
                </button>
                <button
                  onClick={() => setSortBy('recent')}
                  className={cn(
                    'px-2.5 py-1 text-xs font-medium rounded transition-colors flex items-center gap-1',
                    sortBy === 'recent'
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  <Clock className="h-3 w-3" />
                  Recent
                </button>
              </div>
            </div>
          </div>

          {/* Tower grid */}
          {isExploring && !entries.length ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {[...Array(8)].map((_, i) => (
                <Skeleton key={i} className="aspect-[4/3] rounded-lg" />
              ))}
            </div>
          ) : entries.length === 0 ? (
            <div className="text-center py-16">
              <Compass className="w-10 h-10 mx-auto text-muted-foreground mb-3" />
              <p className="font-medium">No towers yet</p>
              <p className="text-sm text-muted-foreground">Be the first to create one!</p>
            </div>
          ) : (
            <>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {entries.slice(0, 8).map((entry) => (
                  <TowerCard
                    key={`${entry.source.language}-${entry.source.package}`}
                    entry={entry}
                    onClick={() => handleSelect(entry)}
                    isAuthenticated={false}
                  />
                ))}
              </div>

              {(hasNextPage || entries.length > 8) && (
                <div className="flex justify-center mt-8">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => fetchNextPage()}
                    disabled={isFetchingNextPage}
                  >
                    {isFetchingNextPage ? (
                      <>
                        <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        Loading...
                      </>
                    ) : (
                      'Load more'
                    )}
                  </Button>
                </div>
              )}
            </>
          )}
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-16 px-4 sm:px-6 border-t">
        <div className="max-w-xl mx-auto text-center">
          <h2 className="text-xl font-semibold mb-3">Ready to get started?</h2>
          <p className="text-muted-foreground mb-6">
            Sign in with GitHub to create your own visualizations 
            and save your favorites.
          </p>
          <Button onClick={onLogin} className="gap-2">
            <Github className="h-4 w-4" />
            Sign in with GitHub
          </Button>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-6 px-4 sm:px-6 border-t">
        <div className="max-w-6xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <Layers className="h-3.5 w-3.5" />
            <span>Stacktower</span>
            <span className="mx-1">·</span>
            <span>Apache-2.0</span>
          </div>
          <div className="flex items-center gap-4">
            <a
              href="https://github.com/matzehuels/stacktower"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 hover:text-foreground transition-colors"
            >
              <Github className="h-3.5 w-3.5" />
              GitHub
            </a>
            <a
              href="https://x.com/stacktower"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1 hover:text-foreground transition-colors"
            >
              <XIcon className="h-3.5 w-3.5" />
              @stacktower
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}
