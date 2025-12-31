/**
 * Navigation types and constants.
 */

// =============================================================================
// Tab Navigation
// =============================================================================

export const TABS = ['packages', 'repos', 'library', 'explore'] as const;

export type Tab = typeof TABS[number];

// Helper to check if a string is a valid tab
export function isValidTab(value: unknown): value is Tab {
  return typeof value === 'string' && TABS.includes(value as Tab);
}

