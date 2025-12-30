/**
 * React Query key factory.
 * 
 * Centralizes all query keys to ensure consistency and make invalidation easy.
 * 
 * Usage:
 *   useQuery({ queryKey: queryKeys.auth.me() })
 *   queryClient.invalidateQueries({ queryKey: queryKeys.repos.all })
 */

export const queryKeys = {
  // Auth
  auth: {
    all: ['auth'] as const,
    me: () => [...queryKeys.auth.all, 'me'] as const,
  },

  // Repositories
  repos: {
    all: ['repos'] as const,
    list: () => [...queryKeys.repos.all, 'list'] as const,
    manifests: (owner: string, repo: string) => 
      [...queryKeys.repos.all, owner, repo, 'manifests'] as const,
  },

  // Jobs
  jobs: {
    all: ['jobs'] as const,
    detail: (jobId: string) => [...queryKeys.jobs.all, jobId] as const,
    list: (filters?: { status?: string }) => 
      [...queryKeys.jobs.all, 'list', filters] as const,
  },

  // Library
  library: {
    all: ['library'] as const,
    list: (limit: number, offset: number) =>
      [...queryKeys.library.all, { limit, offset }] as const,
  },

  // Artifacts
  artifacts: {
    all: ['artifacts'] as const,
    detail: (artifactId: string) => 
      [...queryKeys.artifacts.all, artifactId] as const,
  },

  // Graph data
  graph: {
    all: ['graph'] as const,
    detail: (graphPath: string) => 
      [...queryKeys.graph.all, graphPath] as const,
  },

  // Public stats (landing page)
  stats: () => ['stats'] as const,

  // Supported integrations (landing page)
  integrations: ['integrations'] as const,

  // Package suggestions (autocomplete)
  suggestions: {
    all: ['suggestions'] as const,
    list: (language: string, query: string) =>
      [...queryKeys.suggestions.all, { language, query }] as const,
  },

  // Explore
  explore: {
    all: ['explore'] as const,
    list: (language?: string, sortBy?: 'popular' | 'recent') =>
      [...queryKeys.explore.all, { language, sortBy }] as const,
  },
} as const;

