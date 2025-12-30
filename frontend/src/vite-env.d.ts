/// <reference types="vite/client" />

/**
 * Type declarations for Vite environment variables.
 * 
 * All VITE_* env vars should be declared here for type safety.
 * See: https://vite.dev/guide/env-and-mode#intellisense-for-typescript
 */
interface ImportMetaEnv {
  /** Base URL for API calls */
  readonly VITE_API_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

