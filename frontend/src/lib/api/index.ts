/**
 * API layer exports.
 * 
 * This module provides a clean interface for API interactions:
 * - `api` - Low-level HTTP client
 * - Endpoint functions - Typed functions for each API endpoint
 * - Artifact utilities - For fetching/building artifact URLs
 * - Transform functions - For normalizing API responses
 * 
 * Usage:
 *   import { api, getCurrentUser, getArtifactUrl } from '@/lib/api';
 */

// HTTP client
export { api, ApiError, fetchArtifact, getArtifactUrl } from './client';

// Endpoint functions
export {
  // Auth
  getCurrentUser,
  logout,
  getLoginUrl,
  // Repos
  listRepos,
  getManifests,
  analyzeRepo,
  // Render
  submitRender,
  getRender,
  deleteRender,
  deleteRenders,
  // Jobs
  getJob,
  transformJobResponse,
  // History (deprecated, use library)
  getHistory,
  // Graph
  getGraphData,
  // Public stats
  getPublicStats,
  // Supported integrations
  getIntegrations,
  // Package suggestions
  getPackageSuggestions,
  // Explore
  getExplore,
  // Library
  getLibrary,
  saveToLibrary,
  removeFromLibrary,
} from './endpoints';

// Transform utilities (for advanced usage)
export {
  CACHED_JOB_ID,
  normalizeVizType,
  normalizeJobStatus,
  transformRenderResponse,
  transformJobStatusResponse,
  transformHistoryResponse,
} from './transforms';

