/**
 * HTTP client for API requests.
 * 
 * Provides a thin wrapper around fetch with:
 * - Consistent error handling
 * - Automatic JSON parsing
 * - Credentials included by default
 * - Type-safe responses
 * 
 * Usage:
 *   const user = await api.get<GitHubUser>('/auth/me');
 *   await api.post('/render', { package: 'flask' });
 */

import { API_BASE } from '@/config/constants';

// =============================================================================
// Types
// =============================================================================

export class ApiError extends Error {
  status: number;
  data?: unknown;
  
  constructor(message: string, status: number, data?: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  /** Request body - will be JSON stringified */
  body?: unknown;
}

// =============================================================================
// Client Implementation
// =============================================================================

async function request<T>(
  endpoint: string,
  options: RequestOptions = {}
): Promise<T> {
  const { body, headers: customHeaders, ...rest } = options;

  const url = endpoint.startsWith('/api/')
    ? endpoint
    : `${API_BASE}${endpoint}`;

  const headers: HeadersInit = {
    ...customHeaders,
  };

  if (body) {
    (headers as Record<string, string>)['Content-Type'] = 'application/json';
  }

  const response = await fetch(url, {
    ...rest,
    headers,
    credentials: 'include',
    body: body ? JSON.stringify(body) : undefined,
  });

  // Handle non-JSON responses (e.g., 204 No Content)
  const contentType = response.headers.get('content-type');
  const isJson = contentType?.includes('application/json');
  
  if (!response.ok) {
    let errorMessage = `Request failed with status ${response.status}`;
    let errorData: unknown;

    if (isJson) {
      try {
        errorData = await response.json();
        errorMessage = (errorData as { error?: string })?.error || errorMessage;
      } catch {
        // Ignore JSON parse errors for error responses
      }
    }

    throw new ApiError(errorMessage, response.status, errorData);
  }

  // Return null for 204 No Content
  if (response.status === 204) {
    return null as T;
  }

  return isJson ? response.json() : (response.text() as unknown as T);
}

// =============================================================================
// Public API
// =============================================================================

export const api = {
  get: <T>(endpoint: string, options?: RequestOptions) =>
    request<T>(endpoint, { ...options, method: 'GET' }),

  post: <T>(endpoint: string, body?: unknown, options?: RequestOptions) =>
    request<T>(endpoint, { ...options, method: 'POST', body }),

  put: <T>(endpoint: string, body?: unknown, options?: RequestOptions) =>
    request<T>(endpoint, { ...options, method: 'PUT', body }),

  patch: <T>(endpoint: string, body?: unknown, options?: RequestOptions) =>
    request<T>(endpoint, { ...options, method: 'PATCH', body }),

  delete: <T>(endpoint: string, options?: RequestOptions) =>
    request<T>(endpoint, { ...options, method: 'DELETE' }),
};

/**
 * Fetch a binary artifact (images, PDFs, etc.)
 * Returns a blob URL or raw text for SVG
 */
export async function fetchArtifact(artifactPath: string): Promise<string> {
  const url = artifactPath.startsWith('/api/')
    ? artifactPath
    : `${API_BASE}/artifacts/${artifactPath}`;

  const response = await fetch(url, { credentials: 'include' });
  
  if (!response.ok) {
    throw new ApiError('Failed to fetch artifact', response.status);
  }

  const contentType = response.headers.get('content-type') || '';
  
  // Return SVG as text for inline rendering
  if (contentType.includes('image/svg') || artifactPath.endsWith('.svg')) {
    return response.text();
  }
  
  // Return blob URL for binary formats
  const blob = await response.blob();
  return URL.createObjectURL(blob);
}

/**
 * Build full artifact URL for downloads/image src.
 */
export function getArtifactUrl(artifactPath: string): string {
  return artifactPath.startsWith('/api/')
    ? artifactPath
    : `${API_BASE}/artifacts/${artifactPath}`;
}

