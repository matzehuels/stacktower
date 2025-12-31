/**
 * Central state management hook for App.tsx.
 * 
 * Manages application-level state including:
 * - Tab navigation
 * - Job selection and visualization
 * - URL parameter handling (auth callbacks, shared renders)
 * - Library status tracking
 * 
 * @example
 * const { state, actions } = useAppState(user, checkAuth);
 * 
 * <Sidebar activeTab={state.activeTab} onNavigate={actions.navigate} />
 * <Library onSelect={actions.selectJob} />
 */

import { useState, useEffect, useCallback, useReducer } from 'react';
import { toast } from 'sonner';
import { getRender } from '@/lib/api';
import type { Tab } from '@/types/navigation';
import type { JobResponse, VizType, GitHubUser } from '@/types/api';

// =============================================================================
// State Types
// =============================================================================

interface AppState {
  activeTab: Tab;
  selectedJob: JobResponse | null;
  selectedInLibrary?: boolean;
}

type AppAction =
  | { type: 'SET_TAB'; payload: Tab }
  | { type: 'SET_JOB'; payload: { job: JobResponse; inLibrary?: boolean } }
  | { type: 'CLEAR_SELECTION' }
  | { type: 'RESET' };

// =============================================================================
// Reducer
// =============================================================================

function appReducer(state: AppState, action: AppAction): AppState {
  switch (action.type) {
    case 'SET_TAB':
      return {
        ...state,
        activeTab: action.payload,
        // Clear job selection when navigating to a new tab
        selectedJob: null,
        selectedInLibrary: undefined,
      };
    
    case 'SET_JOB':
      return {
        ...state,
        selectedJob: action.payload.job,
        selectedInLibrary: action.payload.inLibrary,
      };
    
    case 'CLEAR_SELECTION':
      return {
        ...state,
        selectedJob: null,
        selectedInLibrary: undefined,
      };
    
    case 'RESET':
      return {
        activeTab: 'explore',
        selectedJob: null,
        selectedInLibrary: undefined,
      };
    
    default:
      return state;
  }
}

// =============================================================================
// Hook
// =============================================================================

interface UseAppStateOptions {
  user: GitHubUser | null | undefined;
  checkAuth: () => void;
}

export function useAppState({ user, checkAuth }: UseAppStateOptions) {
  const [state, dispatch] = useReducer(appReducer, {
    activeTab: 'explore',
    selectedJob: null,
    selectedInLibrary: undefined,
  });

  // Handle URL parameters (auth callback and shared renders)
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    
    // Handle auth callback
    if (params.get('auth') === 'success') {
      checkAuth();
      window.history.replaceState({}, '', window.location.pathname);
    }
    
    // Handle shared render URL (e.g., ?render=abc123)
    const sharedRenderId = params.get('render');
    if (sharedRenderId && !state.selectedJob) {
      // Load the shared render
      getRender(sharedRenderId)
        .then((job) => {
          dispatch({ type: 'SET_JOB', payload: { job } });
          // Clean URL after loading
          window.history.replaceState({}, '', window.location.pathname);
        })
        .catch((error) => {
          console.error('Failed to load shared render:', error);
          toast.error('Failed to load shared visualization', {
            description: 'The link may be invalid or expired.'
          });
          // Clean URL even on error
          window.history.replaceState({}, '', window.location.pathname);
        });
    }
  }, [checkAuth, state.selectedJob]);

  // Actions
  const navigate = useCallback((tab: Tab) => {
    // Check if user needs to be authenticated for this tab
    if (!user && tab !== 'explore') {
      toast.info('Sign in to access this feature', {
        action: {
          label: 'Sign in with GitHub',
          onClick: () => {
            window.location.href = '/api/v1/auth/github';
          },
        },
      });
      return;
    }
    dispatch({ type: 'SET_TAB', payload: tab });
  }, [user]);

  const selectJob = useCallback((job: JobResponse, inLibrary?: boolean) => {
    dispatch({ type: 'SET_JOB', payload: { job, inLibrary } });
  }, []);

  const clearSelection = useCallback(() => {
    dispatch({ type: 'CLEAR_SELECTION' });
  }, []);

  const reset = useCallback(() => {
    dispatch({ type: 'RESET' });
  }, []);

  const updateJob = useCallback((updatedJob: JobResponse) => {
    dispatch({ type: 'SET_JOB', payload: { 
      job: updatedJob, 
      inLibrary: state.selectedInLibrary 
    } });
  }, [state.selectedInLibrary]);

  return {
    state,
    actions: {
      navigate,
      selectJob,
      clearSelection,
      reset,
      updateJob,
    },
  };
}

