/**
 * Library queries and mutations.
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getLibrary, saveToLibrary, removeFromLibrary } from '@/lib/api';
import { DEFAULT_HISTORY_LIMIT } from '@/config/constants';
import { queryKeys } from './keys';

/**
 * Query for fetching user's library (saved packages + private repos).
 */
export function useLibrary(limit = DEFAULT_HISTORY_LIMIT, offset = 0) {
  return useQuery({
    queryKey: queryKeys.library.list(limit, offset),
    queryFn: () => getLibrary(limit, offset),
    staleTime: 30 * 1000,
  });
}

/**
 * Mutation for saving a package to library.
 */
export function useSaveToLibrary() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ language, pkg }: { language: string; pkg: string }) =>
      saveToLibrary(language, pkg),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.explore.all });
    },
  });
}

/**
 * Mutation for removing a package from library.
 */
export function useRemoveFromLibrary() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ language, pkg }: { language: string; pkg: string }) =>
      removeFromLibrary(language, pkg),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.explore.all });
    },
  });
}

