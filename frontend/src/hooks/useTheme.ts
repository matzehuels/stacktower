/**
 * Hook for accessing theme context.
 * 
 * Usage:
 *   const { theme, setTheme, resolvedTheme } = useTheme();
 */

import { useContext } from 'react';
import { ThemeProviderContext, type ThemeProviderState } from '@/providers/ThemeProvider';

export function useTheme(): ThemeProviderState {
  const context = useContext(ThemeProviderContext);
  
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  
  return context;
}

