/**
 * Application constants and configuration.
 * 
 * This file centralizes all magic strings, URLs, and configuration values
 * to make them easy to find, update, and maintain.
 */

import { env } from './env';

// =============================================================================
// API Configuration
// =============================================================================

/** Base URL for API calls. Proxied through Vite in development. */
export const API_BASE = env.API_BASE_URL;

/** Default polling interval for job status checks (ms) */
export const JOB_POLL_INTERVAL = 1000;

// =============================================================================
// Pagination
// =============================================================================

export const DEFAULT_HISTORY_LIMIT = 20;

// =============================================================================
// Output Formats
// =============================================================================

export const OUTPUT_FORMATS = {
  SVG: 'svg',
  PNG: 'png', 
  PDF: 'pdf',
  JSON: 'json',
} as const;

export const DEFAULT_FORMATS: string[] = ['svg', 'png', 'pdf', 'json'];

// =============================================================================
// Job Status
// =============================================================================

export const JOB_STATUS = {
  PENDING: 'pending',
  PROCESSING: 'processing',
  COMPLETED: 'completed',
  FAILED: 'failed',
  CANCELLED: 'cancelled',
} as const;

export type JobStatus = typeof JOB_STATUS[keyof typeof JOB_STATUS];

/** Statuses that indicate a job is still in progress */
export const ACTIVE_JOB_STATUSES: JobStatus[] = [
  JOB_STATUS.PENDING,
  JOB_STATUS.PROCESSING,
];

/** Statuses that indicate a job has finished (successfully or not) */
export const TERMINAL_JOB_STATUSES: JobStatus[] = [
  JOB_STATUS.COMPLETED,
  JOB_STATUS.FAILED,
  JOB_STATUS.CANCELLED,
];

// =============================================================================
// Visualization Types
// =============================================================================

export const VIZ_TYPE = {
  TOWER: 'tower',
  NODELINK: 'nodelink',
} as const;

export type VizType = typeof VIZ_TYPE[keyof typeof VIZ_TYPE];

// =============================================================================
// Languages / Registries
// =============================================================================

export const LANGUAGE = {
  PYTHON: 'python',
  JAVASCRIPT: 'javascript',
  RUST: 'rust',
  GO: 'go',
  RUBY: 'ruby',
  PHP: 'php',
  JAVA: 'java',
} as const;

export type Language = typeof LANGUAGE[keyof typeof LANGUAGE];

export interface LanguageConfig {
  value: Language;
  label: string;
  placeholder: string;
}

export const LANGUAGES: LanguageConfig[] = [
  { value: 'python', label: 'Python (PyPI)', placeholder: 'flask, requests, django' },
  { value: 'javascript', label: 'JavaScript (npm)', placeholder: 'react, express, lodash' },
  { value: 'rust', label: 'Rust (crates.io)', placeholder: 'serde, tokio, actix-web' },
  { value: 'go', label: 'Go (proxy.golang.org)', placeholder: 'github.com/gin-gonic/gin' },
  { value: 'ruby', label: 'Ruby (RubyGems)', placeholder: 'rails, sinatra, devise' },
  { value: 'php', label: 'PHP (Packagist)', placeholder: 'laravel/framework' },
  { value: 'java', label: 'Java (Maven)', placeholder: 'org.springframework:spring-core' },
];

// =============================================================================
// Quick Examples (for landing page)
// =============================================================================

export const QUICK_EXAMPLES = [
  { lang: LANGUAGE.PYTHON, pkg: 'flask' },
  { lang: LANGUAGE.JAVASCRIPT, pkg: 'express' },
  { lang: LANGUAGE.RUST, pkg: 'serde' },
  { lang: LANGUAGE.GO, pkg: 'github.com/gin-gonic/gin' },
] as const;

// =============================================================================
// External URLs
// =============================================================================

export const EXTERNAL_URLS = {
  GITHUB_REPO: 'https://github.com/matzehuels/stacktower',
} as const;

// =============================================================================
// Registry Display Names
// =============================================================================

export const REGISTRY_DISPLAY_NAMES: Record<string, string> = {
  pypi: 'PyPI',
  npm: 'npm',
  crates: 'crates.io',
  rubygems: 'RubyGems',
  packagist: 'Packagist',
  maven: 'Maven Central',
  go: 'Go Modules',
};

// =============================================================================
// Local Storage Keys
// =============================================================================

export const STORAGE_KEYS = {
  THEME: 'stacktower-theme',
  LAST_LANGUAGE: 'stacktower-last-language',
} as const;

