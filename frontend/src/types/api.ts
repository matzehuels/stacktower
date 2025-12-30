/**
 * API type definitions.
 * 
 * These types mirror the backend API response shapes.
 * For constants like LANGUAGES, see config/constants.ts.
 */

// =============================================================================
// Core Types (re-exported from constants for convenience)
// =============================================================================

// Import from constants - single source of truth
import type { Language, VizType, JobStatus } from '@/config/constants';
export type { Language, VizType, JobStatus };

export type OutputFormat = 'svg' | 'png' | 'pdf' | 'json';

// =============================================================================
// GitHub Auth Types
// =============================================================================

export interface GitHubUser {
  id: number;
  login: string;
  name: string;
  avatar_url: string;
  email?: string;
}

// =============================================================================
// GitHub Repo Types
// =============================================================================

export interface GitHubRepo {
  id: number;
  name: string;
  full_name: string;
  description: string;
  private: boolean;
  default_branch: string;
  language: string;
  updated_at: string;
}

export interface ManifestFile {
  path: string;
  language: Language;
  name: string;
}

export interface RepoAnalyzeRequest {
  manifest_path: string;
  formats?: OutputFormat[];
}

// =============================================================================
// Render Request/Response Types
// =============================================================================

export interface RenderRequest {
  language: Language;
  package: string;
  formats?: OutputFormat[];
  viz_type?: VizType;
  max_depth?: number;
  max_nodes?: number;
  merge?: boolean;
}

export interface RenderResponse {
  status: 'pending' | 'completed';
  render_id?: string;
  job_id?: string;
  cached?: boolean;
  result?: RenderResult;
  error?: string;
}

export interface RenderResult {
  artifacts: Record<string, string>; // format -> artifact URL
  node_count: number;
  edge_count: number;
  viz_type: VizType;
  source: RenderSourceInfo;
}

export interface RenderSourceInfo {
  type: 'package' | 'manifest';
  language: string;
  package?: string;
  repo?: string;
}

// =============================================================================
// Job Types
// =============================================================================

export interface JobResponse {
  job_id: string;
  status: JobStatus;
  type?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  duration?: number;
  result?: JobResult;
  error?: string;
}

export interface JobResult {
  // Current visualization artifacts
  svg?: string;
  png?: string;
  pdf?: string;
  graph_path?: string;  // Shared JSON graph URL
  // Metadata
  nodes?: number;
  edges?: number;
  blocks?: number;
  // Viz type info
  viz_type?: VizType;              // Current viz type being displayed
  available_viz_types?: VizType[]; // Which viz types are cached (for quick switching)
  // Source info for re-rendering other viz types
  source?: {
    language?: string;
    package?: string;
    manifest?: string;
  };
  // Related render IDs (for deleting all viz types together)
  related_render_ids?: string[];
  // Full layout data (includes nebraska, blocks, edges, etc.)
  layout?: LayoutData;
}

// =============================================================================
// Layout Data Types (from render result)
// =============================================================================

export interface LayoutData {
  width?: number;
  height?: number;
  margin_x?: number;
  margin_y?: number;
  style?: string;
  seed?: number;
  randomize?: boolean;
  viz_type?: string;
  merged?: boolean;
  blocks?: LayoutBlock[];
  edges?: LayoutEdge[];
  rows?: Record<string, string[]>;
  nebraska?: NebraskaRanking[];
}

export interface LayoutBlock {
  id: string;
  label: string;
  x: number;
  y: number;
  width: number;
  height: number;
  url?: string;
  meta?: NodeMetadata;
}

export interface LayoutEdge {
  from: string;
  to: string;
}

// =============================================================================
// Graph Data Types (for dependency panel)
// =============================================================================

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface GraphNode {
  id: string;
  row?: number;
  kind?: 'subdivider' | 'auxiliary';
  brittle?: boolean;  // Potentially unmaintained/at-risk package
  meta?: NodeMetadata;
}

export interface NodeMetadata {
  version?: string;
  summary?: string;
  description?: string;
  repo_url?: string;
  repo_stars?: number;
  repo_owner?: string;
  repo_maintainers?: string[];
  repo_last_commit?: string;
  repo_archived?: boolean;
  homepage?: string;
  license?: string;
  [key: string]: unknown;
}

export interface GraphEdge {
  from: string;
  to: string;
}

export interface NebraskaRanking {
  maintainer: string;
  score: number;
  packages: NebraskaPackage[];
}

export interface NebraskaPackage {
  package: string;
  role: string; // "owner", "lead", or "maintainer"
  url?: string;
  depth?: number;
}

// =============================================================================
// History Types
// =============================================================================

// VizTypeRender represents a single visualization type's artifacts
export interface VizTypeRender {
  viz_type: VizType;
  artifacts: Record<string, string> | null; // svg, png, pdf URLs (null if not fetched yet)
}

// HistoryItem represents one package in user's history
// Each item can have multiple viz type renders available
export interface HistoryItem {
  id: string;
  source: RenderSourceInfo;
  node_count: number;
  edge_count: number;
  graph_url: string;              // Shared JSON graph URL
  renders: VizTypeRender[];       // Available viz types with their artifacts
  created_at: string;
}

export interface HistoryResponse {
  renders: HistoryItem[];
  total: number;
  limit: number;
  offset: number;
}

// =============================================================================
// Public Stats Types (landing page)
// =============================================================================

export interface PublicStats {
  total_renders: number;
  total_dependencies: number;
  total_users: number;
}

// =============================================================================
// Supported Integrations
// =============================================================================

export interface IntegrationsResponse {
  languages: LanguageInfo[];
}

export interface LanguageInfo {
  name: string;
  registry: RegistryInfo;
  manifests: ManifestInfoAPI[];
}

export interface RegistryInfo {
  name: string;
  aliases?: string[];
}

export interface ManifestInfoAPI {
  filename: string;
  type: string;
}

// =============================================================================
// Package Suggestions (autocomplete)
// =============================================================================

export interface PackageSuggestion {
  package: string;
  language: string;
  popularity: number;
}

// =============================================================================
// Explore / Discovery
// =============================================================================

export type ExploreSortBy = 'recent' | 'popular';

export interface ExploreVizType {
  viz_type: VizType;
  render_id: string;
  graph_id?: string;
  artifact_svg?: string;
  artifact_png?: string;
  artifact_pdf?: string;
}

export interface ExploreEntry {
  source: RenderSourceInfo;
  node_count: number;
  edge_count: number;
  created_at: string;
  viz_types: ExploreVizType[];
  popularity: number;  // Users with this in library
  in_library: boolean; // Whether current user has this saved
}

export interface ExploreResponse {
  entries: ExploreEntry[];
  total: number;
  limit: number;
  offset: number;
}

// =============================================================================
// User Library
// =============================================================================

export interface LibraryItem {
  language: string;
  package: string;
  saved_at: string;
  viz_types: ExploreVizType[];
  node_count: number;
  edge_count: number;
}

export interface RepoItem {
  id: string;
  source: RenderSourceInfo;
  node_count: number;
  edge_count: number;
  graph_url: string;
  renders: VizTypeRender[];
  created_at: string;
}

export interface LibraryResponse {
  packages: LibraryItem[];
  repos: RepoItem[];
  total: number;
  limit: number;
  offset: number;
}
