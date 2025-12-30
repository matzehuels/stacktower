/**
 * React Query hook for public platform statistics.
 */

import { useQuery } from '@tanstack/react-query';
import { getPublicStats } from '@/lib/api';
import { queryKeys } from './keys';

/**
 * Fetch public platform statistics for the landing page.
 * This does not require authentication.
 */
export function usePublicStats() {
  return useQuery({
    queryKey: queryKeys.stats(),
    queryFn: getPublicStats,
    staleTime: 60 * 1000, // 1 minute - stats don't need to be super fresh
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
}

