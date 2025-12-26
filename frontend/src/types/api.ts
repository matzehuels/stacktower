export type Language = 
  | 'python' 
  | 'javascript' 
  | 'rust' 
  | 'go' 
  | 'ruby' 
  | 'php' 
  | 'java';

export type OutputFormat = 'svg' | 'png' | 'pdf' | 'json';

export type VizType = 'tower' | 'nodelink';

export type JobStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled';

export interface VisualizeRequest {
  language: Language;
  package: string;
  formats?: OutputFormat[];
  viz_type?: VizType;
  max_depth?: number;
  max_nodes?: number;
}

// GitHub integration types
export interface GitHubUser {
  id: number;
  login: string;
  name: string;
  avatar_url: string;
  email?: string;
}

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

// Graph data types (from graph.json)
export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface GraphNode {
  id: string;
  row?: number;
  kind?: 'subdivider' | 'auxiliary';
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
  graph_path?: string;
  layout_path?: string;
  svg?: string;
  png?: string;
  pdf?: string;
  json?: string;
  nodes?: number;
  edges?: number;
  blocks?: number;
  viz_type?: VizType;
}

export const LANGUAGES: { value: Language; label: string; placeholder: string }[] = [
  { value: 'python', label: 'Python (PyPI)', placeholder: 'e.g., flask, requests, django' },
  { value: 'javascript', label: 'JavaScript (npm)', placeholder: 'e.g., react, express, lodash' },
  { value: 'rust', label: 'Rust (crates.io)', placeholder: 'e.g., serde, tokio, actix-web' },
  { value: 'go', label: 'Go (proxy.golang.org)', placeholder: 'e.g., gin, fiber, echo' },
  { value: 'ruby', label: 'Ruby (RubyGems)', placeholder: 'e.g., rails, sinatra, devise' },
  { value: 'php', label: 'PHP (Packagist)', placeholder: 'e.g., laravel/framework, symfony/http-foundation' },
  { value: 'java', label: 'Java (Maven)', placeholder: 'e.g., org.springframework:spring-core' },
];

// =============================================================================
// Streaming / Agent Events
// =============================================================================

export type AgentEventType = 
  | 'thinking'
  | 'tool_call'
  | 'tool_result'
  | 'progress'
  | 'complete'
  | 'error';

export interface AgentEvent {
  type: AgentEventType;
  timestamp: string;
  message?: string;
  data?: Record<string, unknown>;
  tool_name?: string;
  tool_input?: Record<string, unknown>;
}

// =============================================================================
// Analysis Report - Updated to match new backend schema
// =============================================================================

export interface AnalysisReport {
  repo: string;
  language: string;
  analyzed_at: string;
  duration: number;
  summary: AnalysisSummary;
  direct_dependency_narrative: string;
  dependencies: DependencyAnalysis[];
  recommendations: Recommendation[];
  errors?: string[];
}

export interface AnalysisSummary {
  total_dependencies: number;
  direct_dependencies: number;
  dev_dependencies: number;
  healthy: number;
  caution: number;
  warning: number;
  critical: number;
  unused: number;
  unmaintained: number;
  deprecated: number;
  vulnerable: number;
  replaceable: number;
  health_score: number;
  executive_summary: string;
}

export interface DependencyAnalysis {
  name: string;
  version?: string;
  type: 'direct' | 'dev' | 'peer' | 'transitive';
  brought_by?: string[];
  transitive_issues?: TransitiveIssue[];
  context: DependencyContext;
  health: DependencyHealth;
  assessment: 'healthy' | 'caution' | 'warning' | 'critical';
  issues?: string[];
}

export interface TransitiveIssue {
  package: string;
  issue: string;
  impact?: string;
}

export interface DependencyContext {
  purpose: string;
  category: string;
  criticality: 'essential' | 'important' | 'convenient' | 'minimal' | 'unused';
  coupling: 'isolated' | 'moderate' | 'deep' | 'pervasive';
  usages?: Usage[];
  usage_pattern: string;
  file_count: number;
  import_count?: number;
  replaceability?: string;
  replacement_notes?: string;
}

export interface Usage {
  file: string;
  line?: number;
  url?: string;  // Full GitHub URL to file/line
  snippet?: string;
  purpose?: string;
}

export interface DependencyHealth {
  score: number;
  repo_url?: string;
  stars?: number;
  forks?: number;
  last_commit?: string;
  last_release?: string;
  commit_frequency?: string;
  is_archived?: boolean;
  is_deprecated?: boolean;
  license?: string;
  known_vulnerabilities?: string[];
  concerns?: string[];
}

export interface Recommendation {
  package: string;
  action: 'remove' | 'replace' | 'update' | 'monitor' | 'keep' | 'build_own' | 'migrate' | 'secure' | 'refactor';
  priority: 'critical' | 'high' | 'medium' | 'low';
  title: string;
  reason: string;
  triggered_by?: string;
  triggered_by_reason?: string;
  dependency_chain?: string[];
  impact?: string;
  files_affected?: number;
  effort: 'trivial' | 'low' | 'medium' | 'high' | 'major';
  time_estimate?: string;
  alternatives?: Alternative[];
  migration_steps?: string[];
  code_example?: string;
  diy_guidance?: string;
}

export interface Alternative {
  name: string;
  description?: string;
  reason: string;
  repo_url?: string;
  registry_url?: string;
  repo_stars?: number;
  repo_last_commit?: string;
  repo_last_release?: string;
  repo_archived?: boolean;
  license?: string;
}

// Legacy type aliases for backward compatibility
export type DependencyAssessment = DependencyAnalysis;

// Analysis job result
export interface AnalysisJobResult extends JobResult {
  report_path?: string;
  total_dependencies?: number;
  health_score?: number;
  unused?: number;
  outdated?: number;
  deprecated?: number;
  risky?: number;
  replaceable?: number;
  recommendation_count?: number;
  critical_count?: number;
  high_count?: number;
}

// Report summary for list views
export interface ReportSummary {
  job_id: string;
  repo: string;
  language: string;
  status: 'pending' | 'completed' | 'failed';
  created_at: string;
  health_score?: number;
  total_dependencies?: number;
  recommendation_count?: number;
  critical_count?: number;
  high_count?: number;
  duration?: number;
  error?: string;
}

export interface ReportDocument extends ReportSummary {
  report?: AnalysisReport;
}

export interface AnalyzeRequest {
  repo: string;
  language: Language;
  graph_path?: string;
  manifest_path?: string;
  include_dev_deps?: boolean;
  max_depth?: number;
}
