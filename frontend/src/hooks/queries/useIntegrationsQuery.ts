/**
 * Query hook for supported integrations.
 */

import { useQuery } from '@tanstack/react-query';
import { getIntegrations } from '@/lib/api';
import { queryKeys } from './keys';

/**
 * Hook for fetching supported integrations (languages, registries, manifests).
 * This is public data that doesn't require authentication.
 */
export function useIntegrations() {
  return useQuery({
    queryKey: queryKeys.integrations,
    queryFn: getIntegrations,
    staleTime: 24 * 60 * 60 * 1000, // 24 hours - this data rarely changes
    gcTime: 24 * 60 * 60 * 1000,
  });
}

