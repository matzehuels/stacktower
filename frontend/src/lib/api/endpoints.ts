/**
 * API endpoint functions.
 * 
 * Typed functions for each API endpoint. These are used by React Query hooks
 * but can also be called directly when needed.
 * 
 * Adding a new endpoint:
 * 1. Add the function here with proper types
 * 2. Create/update the React Query hook in hooks/queries/
 * 3. Export from hooks/queries/index.ts
 */

import { api } from './client';
import {
  transformRenderResponse,
  transformJobStatusResponse,
  transformHistoryResponse,
  type RawRenderResponse,
  type RawJobResponse,
  type RawHistoryResponse,
} from './transforms';
import type {
  GitHubUser,
  GitHubRepo,
  ManifestFile,
  RenderRequest,
  RepoAnalyzeRequest,
  JobResponse,
  HistoryResponse,
  GraphData,
} from '@/types/api';

// =============================================================================
// Auth Endpoints
// =============================================================================

/**
 * Get the currently authenticated user.
 * Returns null if not authenticated.
 */
export async function getCurrentUser(): Promise<GitHubUser | null> {
  try {
    return await api.get<GitHubUser>('/auth/me');
  } catch {
    return null;
  }
}

/**
 * Log out the current user.
 */
export async function logout(): Promise<void> {
  await api.post('/auth/logout');
}

/**
 * Get the GitHub OAuth login URL.
 */
export function getLoginUrl(): string {
  // This uses full page redirect, not API call
  return `/api/v1/auth/github`;
}

// =============================================================================
// Repository Endpoints
// =============================================================================

/**
 * List the authenticated user's GitHub repositories.
 */
export async function listRepos(): Promise<GitHubRepo[]> {
  const data = await api.get<GitHubRepo[] | null>('/repos');
  return data || [];
}

/**
 * Get manifest files for a repository.
 */
export async function getManifests(
  owner: string,
  repo: string
): Promise<ManifestFile[]> {
  const data = await api.get<ManifestFile[] | null>(
    `/repos/${owner}/${repo}/manifests`
  );
  return data || [];
}

/**
 * Analyze a repository's dependencies.
 */
export async function analyzeRepo(
  owner: string,
  repo: string,
  request: RepoAnalyzeRequest
): Promise<JobResponse> {
  return api.post<JobResponse>(
    `/repos/${owner}/${repo}/analyze`,
    request
  );
}

// =============================================================================
// Render Endpoints
// =============================================================================

/**
 * Submit a render request for a package.
 * Uses canonical transform to normalize the response.
 */
export async function submitRender(request: RenderRequest): Promise<JobResponse> {
  const raw = await api.post<RawRenderResponse>('/render', request);
  
  return transformRenderResponse(raw, {
    language: request.language,
    package: request.package,
    viz_type: request.viz_type,
  });
}

/**
 * Get a render by ID (includes full layout data).
 */
export async function getRender(renderId: string): Promise<JobResponse> {
  const raw = await api.get<RawRenderResponse>(`/render/${renderId}`);
  // Transform to JobResponse format
  return transformRenderResponse(raw, {
    language: raw.result?.source?.language || '',
    package: raw.result?.source?.package || '',
    viz_type: raw.result?.viz_type as any,
  });
}

/**
 * Delete a render from history.
 */
export async function deleteRender(renderId: string): Promise<void> {
  await api.delete(`/render/${renderId}`);
}

/**
 * Delete multiple renders (e.g., all viz types for a package).
 */
export async function deleteRenders(renderIds: string[]): Promise<void> {
  await Promise.all(renderIds.map(id => api.delete(`/render/${id}`)));
}

// =============================================================================
// Job Endpoints
// =============================================================================

/**
 * Get the status of a job.
 * Uses canonical transform to normalize the response.
 */
export async function getJob(jobId: string): Promise<JobResponse> {
  const raw = await api.get<RawJobResponse>(`/jobs/${jobId}`);
  return transformJobStatusResponse(raw);
}

/**
 * Transform raw API job response to normalized JobResponse.
 * Re-exported from transforms for backward compatibility.
 */
export { transformJobStatusResponse as transformJobResponse } from './transforms';

// =============================================================================
// History Endpoints
// =============================================================================

/**
 * Get render history for the current user.
 * Uses canonical transform to normalize the response.
 */
export async function getHistory(
  limit: number,
  offset: number
): Promise<HistoryResponse> {
  const raw = await api.get<RawHistoryResponse>(`/history?limit=${limit}&offset=${offset}`);
  return transformHistoryResponse(raw);
}

// =============================================================================
// Public Stats (no auth required)
// =============================================================================

import type { PublicStats, PackageSuggestion, IntegrationsResponse } from '@/types/api';

/**
 * Get public platform statistics for the landing page.
 * This endpoint does not require authentication.
 */
export async function getPublicStats(): Promise<PublicStats> {
  return api.get<PublicStats>('/stats');
}

// =============================================================================
// Supported Integrations (no auth required)
// =============================================================================

/**
 * Get supported integrations (languages, registries, manifest files).
 * This endpoint does not require authentication.
 */
export async function getIntegrations(): Promise<IntegrationsResponse> {
  return api.get<IntegrationsResponse>('/integrations');
}

// =============================================================================
// Package Suggestions (no auth required - for autocomplete)
// =============================================================================

interface PackageSuggestionsResponse {
  suggestions: PackageSuggestion[];
}

/**
 * Get package suggestions for autocomplete.
 * Returns packages from global render history that match the query.
 */
export async function getPackageSuggestions(
  language: string,
  query: string
): Promise<PackageSuggestion[]> {
  const params = new URLSearchParams();
  if (language) params.set('language', language);
  if (query) params.set('q', query);
  
  const response = await api.get<PackageSuggestionsResponse>(
    `/packages/suggestions?${params.toString()}`
  );
  return response.suggestions || [];
}

// =============================================================================
// Explore
// =============================================================================

import type { ExploreResponse, LibraryResponse } from '@/types/api';

/**
 * Get public towers for discovery.
 * @param sortBy - "popular" (default) or "recent"
 */
export async function getExplore(
  language?: string,
  sortBy: 'popular' | 'recent' = 'popular',
  limit = 20,
  offset = 0
): Promise<ExploreResponse> {
  const params = new URLSearchParams();
  if (language) params.set('language', language);
  params.set('sort_by', sortBy);
  params.set('limit', String(limit));
  params.set('offset', String(offset));
  
  return api.get<ExploreResponse>(`/explore?${params.toString()}`);
}

// =============================================================================
// User Library
// =============================================================================

/**
 * Get the user's library (saved packages + private repos).
 */
export async function getLibrary(limit = 20, offset = 0): Promise<LibraryResponse> {
  return api.get<LibraryResponse>(`/library?limit=${limit}&offset=${offset}`);
}

/**
 * Save a package to the user's library.
 */
export async function saveToLibrary(language: string, pkg: string): Promise<void> {
  await api.put(`/library/${language}/${pkg}`);
}

/**
 * Remove a package from the user's library.
 */
export async function removeFromLibrary(language: string, pkg: string): Promise<void> {
  await api.delete(`/library/${language}/${pkg}`);
}

// =============================================================================
// Graph Data
// =============================================================================

/**
 * Fetch and transform graph data (dependency graph JSON).
 */
export async function getGraphData(graphPath: string): Promise<GraphData> {
  const url = graphPath.startsWith('/api/')
    ? graphPath
    : `/artifacts/${graphPath}`;

  const json = await api.get<RawGraphData>(url);

  // Transform tower JSON format to GraphData format
  return {
    nodes: (json.blocks || json.nodes || []).map((block) => ({
      id: block.id || block.label || '',
      row: block.row,
      // Handle both formats: kind as string OR auxiliary/subdivider as booleans
      kind: block.kind as 'subdivider' | 'auxiliary' | undefined
        || (block.auxiliary ? 'auxiliary' : undefined)
        || (block.subdivider ? 'subdivider' : undefined),
      brittle: block.brittle,
      meta: block.meta || {
        description: block.description,
        repo_url: block.url,
        repo_stars: block.stars,
        version: block.version,
      },
    })),
    edges: json.edges || [],
  };
}

interface RawGraphData {
  blocks?: RawGraphNode[];
  nodes?: RawGraphNode[];
  edges?: { from: string; to: string }[];
}

interface RawGraphNode {
  id?: string;
  label?: string;
  row?: number;
  kind?: string;  // "subdivider" | "auxiliary" - string format
  auxiliary?: boolean;  // Boolean format (legacy)
  subdivider?: boolean;  // Boolean format (legacy)
  brittle?: boolean;  // Potentially unmaintained/at-risk package
  meta?: Record<string, unknown>;
  description?: string;
  url?: string;
  stars?: number;
  version?: string;
}

