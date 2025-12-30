/**
 * Package suggestions query for autocomplete.
 */

import { useQuery } from '@tanstack/react-query';
import { getPackageSuggestions } from '@/lib/api';
import { queryKeys } from './keys';

/**
 * Hook for fetching package suggestions for autocomplete.
 * Debouncing should be handled by the caller.
 */
export function usePackageSuggestions(language: string, query: string) {
  return useQuery({
    queryKey: queryKeys.suggestions.list(language, query),
    queryFn: () => getPackageSuggestions(language, query),
    enabled: query.length >= 1 || language !== '', // Require at least 1 char or a language filter
    staleTime: 5 * 60 * 1000, // 5 minutes - suggestions don't change often
    gcTime: 10 * 60 * 1000,
  });
}

