/**
 * TowerExplorer component for discovering and browsing towers.
 * Clean, GitHub-inspired grid layout.
 */

import { useState, useMemo } from 'react';
import { Compass, Loader2, TrendingUp, Clock } from 'lucide-react';
import { useExplore } from '@/hooks/queries';
import { getRender } from '@/lib/api';
import { toast } from 'sonner';
import { TowerCard } from '@/components/TowerCard';
import { GitHubLoginButton } from '@/components/GitHubLoginButton';
import { PackageSearchBar } from '@/components/PackageSearchBar';
import { LanguageIcon } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { LANGUAGES } from '@/config/constants';
import type { Language } from '@/config/constants';
import type { JobResponse, ExploreEntry, ExploreSortBy, RenderRequest } from '@/types/api';

interface Props {
  onSelect: (job: JobResponse, inLibrary?: boolean) => void;
  onRender?: (request: RenderRequest) => void;
  isAuthenticated?: boolean;
  onLogin?: () => void;
}

export function TowerExplorer({ onSelect, onRender, isAuthenticated = true, onLogin }: Props) {
  const [selectedLanguage, setSelectedLanguage] = useState<Language | 'all'>('all');
  const [sortBy, setSortBy] = useState<ExploreSortBy>('popular');
  
  const languageFilter = selectedLanguage === 'all' ? undefined : selectedLanguage;
  
  const {
    data: exploreData,
    isLoading: isExploring,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
    error: exploreError,
  } = useExplore({ language: languageFilter, sortBy });

  const entries = useMemo(() => {
    if (!exploreData?.pages) return [];
    return exploreData.pages.flatMap((page) => page.entries);
  }, [exploreData?.pages]);

  // Handle package selection from search bar
  const handleSearchSelect = (language: Language, packageName: string) => {
    // First check if this package exists in our explore entries
    const existingEntry = entries.find(
      (e) => e.source.language === language && e.source.package === packageName
    );
    
    if (existingEntry) {
      // Package exists, select it directly
      handleSelect(existingEntry);
    } else if (onRender && isAuthenticated) {
      // Package doesn't exist in explore, trigger a render
      onRender({
        language,
        package: packageName,
        formats: ['svg', 'png', 'pdf', 'json'],
        viz_type: 'tower',
        merge: true,
      });
    } else if (onLogin) {
      // Not authenticated, prompt to login
      onLogin();
    }
  };

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
      onSelect(jobResponse, entry.in_library);
    } catch (error) {
      console.error('Failed to fetch render:', error);
      toast.error('Failed to load visualization');
    }
  };

  const totalCount = exploreData?.pages[0]?.total ?? 0;

  return (
    <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
      {/* Login banner for unauthenticated users */}
      {!isAuthenticated && onLogin && (
        <div className="px-4 py-2.5 bg-muted/50 border-b">
          <div className="flex items-center justify-between gap-4 max-w-screen-xl mx-auto">
            <p className="text-sm text-muted-foreground">
              <span className="font-medium text-foreground">Sign in</span> to save towers and create your own visualizations
            </p>
            <GitHubLoginButton login={onLogin} size="sm" className="shrink-0" />
          </div>
        </div>
      )}

      {/* Header */}
      <div className="px-4 sm:px-6 py-4 border-b bg-background">
        <div className="flex items-center gap-2 mb-3">
          <Compass className="w-4 h-4 text-muted-foreground" />
          <h2 className="text-sm font-medium">Explore</h2>
          {totalCount > 0 && (
            <span className="text-xs text-muted-foreground tabular-nums">
              {totalCount.toLocaleString()} packages
            </span>
          )}
        </div>

        {/* Filter bar */}
        <div className="flex flex-col sm:flex-row gap-2">
          {/* Language filter */}
          <div className="flex gap-0.5 p-0.5 bg-muted rounded-md overflow-x-auto shrink-0">
            <button
              onClick={() => setSelectedLanguage('all')}
              className={cn(
                'px-2.5 py-1 text-xs font-medium rounded transition-colors whitespace-nowrap',
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
                  'p-1.5 rounded transition-colors whitespace-nowrap',
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
          <div className="flex gap-0.5 p-0.5 bg-muted rounded-md shrink-0">
            <button
              onClick={() => setSortBy('popular')}
              className={cn(
                'px-2.5 py-1 text-xs font-medium rounded transition-colors whitespace-nowrap flex items-center gap-1',
                sortBy === 'popular'
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
              title="Sort by popularity"
            >
              <TrendingUp className="h-3 w-3" />
              Popular
            </button>
            <button
              onClick={() => setSortBy('recent')}
              className={cn(
                'px-2.5 py-1 text-xs font-medium rounded transition-colors whitespace-nowrap flex items-center gap-1',
                sortBy === 'recent'
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
              title="Sort by most recent"
            >
              <Clock className="h-3 w-3" />
              Recent
            </button>
          </div>

          {/* Search with autocomplete */}
          <PackageSearchBar
            onSelect={handleSearchSelect}
            language={selectedLanguage}
            placeholder="Search packages..."
            className="flex-1 min-w-0"
          />
        </div>
      </div>

      {/* Grid content */}
      <div className="flex-1 overflow-y-auto p-4 sm:p-6">
        {isExploring && !entries.length ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {[...Array(12)].map((_, i) => (
              <Skeleton key={i} className="aspect-[4/3] rounded-lg" />
            ))}
          </div>
        ) : exploreError ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <p className="text-sm text-destructive mb-3">{exploreError.message}</p>
              <Button variant="outline" size="sm" onClick={() => fetchNextPage()}>
                Try again
              </Button>
            </div>
          </div>
        ) : entries.length === 0 ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <Compass className="w-8 h-8 mx-auto text-muted-foreground mb-3" />
              <p className="font-medium">No towers found</p>
              <p className="text-sm text-muted-foreground mt-1">
                Be the first to create one!
              </p>
            </div>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {entries.map((entry) => (
                <TowerCard
                  key={`${entry.source.language}-${entry.source.package}`}
                  entry={entry}
                  onClick={() => handleSelect(entry)}
                  isAuthenticated={isAuthenticated}
                />
              ))}
            </div>

            {hasNextPage && (
              <div className="flex justify-center mt-6">
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
    </div>
  );
}
