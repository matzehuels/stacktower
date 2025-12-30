/**
 * Artifact queries for fetching rendered outputs (SVG, PNG, etc.)
 */

import { useQuery } from '@tanstack/react-query';
import { fetchArtifact, getArtifactUrl, getGraphData } from '@/lib/api';
import { queryKeys } from './keys';

/**
 * Query for fetching an artifact (SVG, PNG, etc.)
 */
export function useArtifact(artifactPath: string | undefined) {
  return useQuery({
    queryKey: queryKeys.artifacts.detail(artifactPath || ''),
    queryFn: () => fetchArtifact(artifactPath!),
    enabled: Boolean(artifactPath),
    staleTime: Infinity, // Artifacts are immutable, cache forever
    gcTime: 10 * 60 * 1000, // Keep in cache for 10 minutes
  });
}

/**
 * Query for fetching graph data (for dependency panel).
 */
export function useGraphData(graphPath: string | undefined) {
  return useQuery({
    queryKey: queryKeys.graph.detail(graphPath || ''),
    queryFn: () => getGraphData(graphPath!),
    enabled: Boolean(graphPath),
    staleTime: Infinity, // Graph data is immutable
    gcTime: 10 * 60 * 1000,
  });
}

// Re-export utility
export { getArtifactUrl };
