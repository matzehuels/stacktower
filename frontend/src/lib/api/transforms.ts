/**
 * Canonical API response transformations.
 *
 * This module provides a single source of truth for transforming raw API
 * responses into normalized frontend types. All API endpoint functions should
 * use these transforms to ensure consistency.
 *
 * Design principles:
 * - One transform function per response type
 * - Explicit handling of optional/nullable fields
 * - No magic strings - use constants for sentinel values
 * - Type-safe conversions with validation
 */

import type {
  JobResponse,
  JobResult,
  VizType,
  RenderSourceInfo,
  HistoryItem,
  VizTypeRender,
} from '@/types/api';
import { VIZ_TYPE, JOB_STATUS } from '@/config/constants';

// Valid viz types for validation
const VALID_VIZ_TYPES = Object.values(VIZ_TYPE) as string[];

// Valid job statuses for validation
const VALID_JOB_STATUSES = Object.values(JOB_STATUS) as string[];

// =============================================================================
// Constants
// =============================================================================

/** Sentinel value for cached results that don't have a real job ID */
export const CACHED_JOB_ID = '__cached__';

// =============================================================================
// Raw API Response Types (internal - match backend exactly)
// =============================================================================

export interface RawRenderResponse {
  status: string;
  render_id?: string;
  job_id?: string;
  cached?: boolean;
  stale?: boolean;
  refreshing?: boolean;
  refresh_job_id?: string;
  result?: {
    artifacts?: Record<string, string>;
    node_count?: number;
    edge_count?: number;
    viz_type?: string;
    source?: {
      type?: string;
      language?: string;
      package?: string;
      repo?: string;
    };
    layout?: RawLayoutData;
  };
  error?: string;
}

// Layout data as returned from API
interface RawLayoutData {
  width?: number;
  height?: number;
  margin_x?: number;
  margin_y?: number;
  style?: string;
  seed?: number;
  randomize?: boolean;
  viz_type?: string;
  merged?: boolean;
  blocks?: Array<{
    id: string;
    label: string;
    x: number;
    y: number;
    width: number;
    height: number;
    url?: string;
    meta?: Record<string, unknown>;
  }>;
  edges?: Array<{ from: string; to: string }>;
  rows?: Record<string, string[]>;
  nebraska?: Array<{
    maintainer: string;
    score: number;
    packages: Array<{
      package: string;
      role: string;
      url?: string;
      depth?: number;
    }>;
  }>;
  [key: string]: unknown;
}

export interface RawJobResponse {
  job_id: string;
  status: string;
  type?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  duration?: number;
  error?: string;
  result?: {
    render_id?: string;
    graph_id?: string;
    graph_hash?: string;
    artifacts?: Record<string, string>;
    node_count?: number;
    edge_count?: number;
    nodes?: number;
    edges?: number;
    blocks?: number;
    viz_type?: string;
    graph_path?: string;
    svg?: string;
    png?: string;
    pdf?: string;
    json?: string;
    source?: {
      language?: string;
      package?: string;
      manifest?: string;
    };
    // Full layout data (includes nebraska, blocks, edges, etc.)
    layout?: RawLayoutData;
  };
}

export interface RawHistoryItem {
  id: string;
  source: {
    type: string;
    language: string;
    package?: string;
    repo?: string;
    manifest_filename?: string;
  };
  node_count: number;
  edge_count: number;
  graph_url: string;
  renders: Array<{
    viz_type: string;
    artifacts: Record<string, string> | null;
  }>;
  created_at: string;
}

export interface RawHistoryResponse {
  renders: RawHistoryItem[];
  total: number;
  limit: number;
  offset: number;
}

// =============================================================================
// Validation Helpers
// =============================================================================

/**
 * Validate and normalize a viz_type string to the VizType union.
 * Returns 'tower' as default for invalid/missing values.
 */
export function normalizeVizType(vizType: string | undefined): VizType {
  if (vizType && VALID_VIZ_TYPES.includes(vizType)) {
    return vizType as VizType;
  }
  return 'tower';
}

/**
 * Validate and normalize a job status string.
 * Returns 'pending' as default for invalid/missing values.
 */
export function normalizeJobStatus(status: string | undefined): JobResponse['status'] {
  if (status && VALID_JOB_STATUSES.includes(status)) {
    return status as JobResponse['status'];
  }
  return 'pending';
}

// =============================================================================
// Transform Functions
// =============================================================================

/**
 * Transform a raw render response into a normalized JobResponse.
 * Handles both cached (immediate) and async (pending) responses.
 */
export function transformRenderResponse(
  raw: RawRenderResponse,
  requestContext: { language: string; package: string; viz_type?: VizType }
): JobResponse {
  const sourceInfo = {
    language: raw.result?.source?.language || requestContext.language,
    package: raw.result?.source?.package || requestContext.package,
  };

  // Extract job/render ID with fallback for cached results
  const id = raw.job_id || raw.render_id || (raw.cached ? CACHED_JOB_ID : '');

  // Build result if we have artifacts or it's completed
  let result: JobResult | undefined;
  if (raw.result || raw.status === 'completed') {
    const artifacts = raw.result?.artifacts || {};
    result = {
      graph_path: artifacts.json,
      svg: artifacts.svg,
      png: artifacts.png,
      pdf: artifacts.pdf,
      nodes: raw.result?.node_count,
      edges: raw.result?.edge_count,
      viz_type: normalizeVizType(raw.result?.viz_type || requestContext.viz_type),
      source: sourceInfo,
      layout: raw.result?.layout,
    };
  }

  return {
    job_id: id,
    status: normalizeJobStatus(raw.status),
    created_at: new Date().toISOString(),
    result,
    error: raw.error,
  };
}

/**
 * Transform a raw job status response into a normalized JobResponse.
 * Handles the various artifact field locations in job results.
 */
export function transformJobStatusResponse(raw: RawJobResponse): JobResponse {
  const result = raw.result;
  const artifacts = result?.artifacts;

  // Prefer render_id over job_id when available (for completed renders)
  const id = result?.render_id || raw.job_id;

  // Build job result from raw data
  let jobResult: JobResult | undefined;
  if (result) {
    jobResult = {
      // Artifact URLs - check both direct fields and artifacts map
      graph_path: artifacts?.json || result.graph_path,
      svg: artifacts?.svg || result.svg,
      png: artifacts?.png || result.png,
      pdf: artifacts?.pdf || result.pdf,
      // Metadata - check both naming conventions
      nodes: result.node_count ?? result.nodes,
      edges: result.edge_count ?? result.edges,
      blocks: result.blocks,
      // Viz type with validation
      viz_type: normalizeVizType(result.viz_type),
      // Source info for re-rendering
      source: result.source,
      // Full layout data (includes nebraska, blocks, edges, etc.)
      layout: result.layout,
    };
  }

  return {
    job_id: id,
    status: normalizeJobStatus(raw.status),
    type: raw.type,
    created_at: raw.created_at,
    started_at: raw.started_at,
    completed_at: raw.completed_at,
    duration: raw.duration,
    error: raw.error,
    result: jobResult,
  };
}

/**
 * Transform a raw history item into a normalized HistoryItem.
 */
export function transformHistoryItem(raw: RawHistoryItem): HistoryItem {
  // Transform source info
  const source: RenderSourceInfo = {
    type: (raw.source.type as 'package' | 'manifest') || 'package',
    language: raw.source.language,
    package: raw.source.package,
    repo: raw.source.repo,
  };

  // Transform viz type renders
  const renders: VizTypeRender[] = raw.renders.map((r) => ({
    viz_type: normalizeVizType(r.viz_type),
    artifacts: r.artifacts,
  }));

  return {
    id: raw.id,
    source,
    node_count: raw.node_count,
    edge_count: raw.edge_count,
    graph_url: raw.graph_url,
    renders,
    created_at: raw.created_at,
  };
}

/**
 * Transform a raw history response into normalized HistoryResponse.
 */
export function transformHistoryResponse(raw: RawHistoryResponse) {
  return {
    renders: raw.renders.map(transformHistoryItem),
    total: raw.total,
    limit: raw.limit,
    offset: raw.offset,
  };
}

