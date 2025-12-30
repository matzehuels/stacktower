/**
 * Repository queries.
 */

import { useQuery } from '@tanstack/react-query';
import { listRepos, getManifests } from '@/lib/api';
import { queryKeys } from './keys';
import type { Language } from '@/types/api';

/**
 * Query for fetching user's GitHub repositories.
 */
export function useRepos() {
  return useQuery({
    queryKey: queryKeys.repos.list(),
    queryFn: listRepos,
    staleTime: 60 * 1000, // Repos don't change often, cache for 1 minute
  });
}

/**
 * Query for fetching manifest files in a repository.
 */
export function useManifests(owner: string, repo: string) {
  return useQuery({
    queryKey: queryKeys.repos.manifests(owner, repo),
    queryFn: () => getManifests(owner, repo),
    enabled: Boolean(owner && repo), // Only run when we have owner and repo
    staleTime: 2 * 60 * 1000, // Manifests are stable, cache for 2 minutes
  });
}

// Re-export the Language type for convenience
export type { Language };
