/**
 * TowerExplorer component for discovering and browsing towers.
 * Clean, GitHub-inspired grid layout.
 */

import { useState, useMemo } from 'react';
import { Compass, Loader2, TrendingUp, Clock } from 'lucide-react';
import { useExplore } from '@/hooks/queries';
import { selectExploreEntry } from '@/lib/helpers/explore';
import { TowerCard } from '@/components/TowerCard';
import { GitHubLoginButton } from '@/components/GitHubLoginButton';
import { PackageSearchBar } from '@/components/PackageSearchBar';
import { LanguageFilter, SortToggle, Button, EmptyState, LoadingGrid } from '@/components/ui';
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

  const handleSelect = (entry: ExploreEntry) => {
    selectExploreEntry(entry, onSelect);
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
          <LanguageFilter 
            value={selectedLanguage}
            onChange={setSelectedLanguage}
            className="overflow-x-auto"
          />

          {/* Sort toggle */}
          <SortToggle
            value={sortBy}
            onChange={setSortBy}
            options={[
              { value: 'popular' as const, label: 'Popular', icon: <TrendingUp className="h-3 w-3" /> },
              { value: 'recent' as const, label: 'Recent', icon: <Clock className="h-3 w-3" /> },
            ]}
            className="shrink-0"
          />

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
          <LoadingGrid count={12} />
        ) : exploreError ? (
          <EmptyState
            title="Failed to load towers"
            description={exploreError.message}
            action={
              <Button variant="outline" size="sm" onClick={() => fetchNextPage()}>
                Try again
              </Button>
            }
          />
        ) : entries.length === 0 ? (
          <EmptyState
            icon={<Compass className="w-8 h-8" />}
            title="No towers found"
            description="Be the first to create one!"
          />
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
