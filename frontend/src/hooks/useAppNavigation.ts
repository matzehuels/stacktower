/**
 * Hook for managing app navigation and job selection state.
 * 
 * Centralizes navigation logic from App.tsx to reduce component complexity.
 * 
 * @example
 * const { activeTab, selectedJob, navigate, selectJob, clearSelection } = useAppNavigation();
 * 
 * <Sidebar onNavigate={navigate} activeTab={activeTab} />
 * <Library onSelect={selectJob} />
 */

import { useState, useCallback } from 'react';
import type { JobResponse } from '@/types/api';
import type { Tab } from '@/types/navigation';

interface AppNavigationState {
  /** Current active tab */
  activeTab: Tab;
  /** Currently selected job/visualization */
  selectedJob: JobResponse | null;
  /** Whether the selected job is in the user's library */
  selectedInLibrary?: boolean;
}

interface AppNavigationActions {
  /** Navigate to a different tab */
  navigate: (tab: Tab) => void;
  /** Select a job/visualization */
  selectJob: (job: JobResponse, inLibrary?: boolean) => void;
  /** Clear the current selection */
  clearSelection: () => void;
  /** Reset to initial state */
  reset: () => void;
}

type UseAppNavigationReturn = AppNavigationState & AppNavigationActions;

/**
 * Hook for managing application navigation and job selection.
 */
export function useAppNavigation(initialTab: Tab = 'explore'): UseAppNavigationReturn {
  const [activeTab, setActiveTab] = useState<Tab>(initialTab);
  const [selectedJob, setSelectedJob] = useState<JobResponse | null>(null);
  const [selectedInLibrary, setSelectedInLibrary] = useState<boolean | undefined>();

  const navigate = useCallback((tab: Tab) => {
    setActiveTab(tab);
    setSelectedJob(null);
    setSelectedInLibrary(undefined);
  }, []);

  const selectJob = useCallback((job: JobResponse, inLibrary?: boolean) => {
    setSelectedJob(job);
    setSelectedInLibrary(inLibrary);
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedJob(null);
    setSelectedInLibrary(undefined);
  }, []);

  const reset = useCallback(() => {
    setActiveTab(initialTab);
    setSelectedJob(null);
    setSelectedInLibrary(undefined);
  }, [initialTab]);

  return {
    activeTab,
    selectedJob,
    selectedInLibrary,
    navigate,
    selectJob,
    clearSelection,
    reset,
  };
}


