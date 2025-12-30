/**
 * React Query hook for explore page data.
 */

import { useInfiniteQuery } from '@tanstack/react-query';
import { queryKeys } from './keys';
import { getExplore } from '@/lib/api';
import type { ExploreEntry, ExploreSortBy } from '@/types/api';

const PAGE_SIZE = 20;

export interface UseExploreOptions {
  language?: string;
  sortBy?: ExploreSortBy;
  enabled?: boolean;
}

export interface ExploreData {
  entries: ExploreEntry[];
  total: number;
}

/**
 * Infinite query hook for explore page.
 */
export function useExplore(options: UseExploreOptions = {}) {
  const { language, sortBy = 'popular', enabled = true } = options;

  return useInfiniteQuery({
    queryKey: queryKeys.explore.list(language, sortBy),
    queryFn: async ({ pageParam = 0 }) => {
      const response = await getExplore(language, sortBy, PAGE_SIZE, pageParam);
      return response;
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => {
      const totalLoaded = allPages.length * PAGE_SIZE;
      if (totalLoaded >= lastPage.total) {
        return undefined;
      }
      return totalLoaded;
    },
    enabled,
    staleTime: 30_000,
  });
}
