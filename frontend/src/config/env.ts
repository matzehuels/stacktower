/**
 * Environment configuration.
 * 
 * All environment variables should be accessed through this module
 * to ensure type safety and provide sensible defaults.
 * 
 * In Vite, env vars must be prefixed with VITE_ to be exposed to the client.
 * See: https://vite.dev/guide/env-and-mode
 * 
 * Usage:
 *   import { env } from '@/config/env';
 *   console.log(env.API_BASE_URL);
 */

interface Env {
  /** Base URL for API calls. Defaults to '/api/v1' (proxied in dev). */
  API_BASE_URL: string;
  /** Whether we're in development mode */
  DEV: boolean;
  /** Whether we're in production mode */
  PROD: boolean;
  /** Current mode (development, production, etc.) */
  MODE: string;
}

export const env: Env = {
  API_BASE_URL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  DEV: import.meta.env.DEV,
  PROD: import.meta.env.PROD,
  MODE: import.meta.env.MODE,
};

/**
 * Validate that required env vars are set.
 * Call this early in app initialization.
 */
export function validateEnv(): void {
  // Currently all vars have defaults, so nothing to validate
  // Add required checks here if needed in the future
  if (env.DEV) {
    console.log('[env] Running in development mode');
    console.log('[env] API_BASE_URL:', env.API_BASE_URL);
  }
}

