/**
 * Job mutations for render and analyze operations.
 * 
 * These mutations handle submitting jobs and polling for their completion.
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState, useCallback, useEffect, useRef } from 'react';
import { submitRender, analyzeRepo, getJob } from '@/lib/api';
import { JOB_POLL_INTERVAL, TERMINAL_JOB_STATUSES } from '@/config/constants';
import type { JobStatus } from '@/config/constants';
import { queryKeys } from './keys';
import type { JobResponse, RepoAnalyzeRequest } from '@/types/api';
import { parseError } from '@/lib/helpers/errors';
import { showError } from '@/lib/helpers/toast';

// =============================================================================
// Shared Polling Logic
// =============================================================================

function useJobPollingState() {
  const queryClient = useQueryClient();
  const [job, setJob] = useState<JobResponse | null>(null);
  const pollingRef = useRef<number | null>(null);
  // Keep track of initial source info (from the render request)
  const sourceRef = useRef<{ language?: string; package?: string } | null>(null);

  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  }, []);

  const pollJob = useCallback(async (jobId: string) => {
    try {
      const jobData = await getJob(jobId);
      
      // Preserve source info from initial request if polled result doesn't have it
      if (sourceRef.current && jobData.result && !jobData.result.source) {
        jobData.result.source = sourceRef.current;
      }
      
      setJob(jobData);

      if (TERMINAL_JOB_STATUSES.includes(jobData.status as JobStatus)) {
        stopPolling();
        if (jobData.status === 'completed') {
          queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
          queryClient.invalidateQueries({ queryKey: queryKeys.suggestions.all });
          queryClient.invalidateQueries({ queryKey: queryKeys.explore.all });
        }
      }
    } catch {
      stopPolling();
    }
  }, [stopPolling, queryClient]);

  const startPolling = useCallback((jobId: string) => {
    pollingRef.current = window.setInterval(
      () => pollJob(jobId),
      JOB_POLL_INTERVAL
    );
  }, [pollJob]);

  // Cleanup on unmount
  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  const setJobWithSource = useCallback((newJob: JobResponse | null) => {
    // Store source info for later polling
    if (newJob?.result?.source) {
      sourceRef.current = newJob.result.source;
    }
    setJob(newJob);
  }, []);

  return { job, setJob: setJobWithSource, startPolling, stopPolling, queryClient };
}

// =============================================================================
// Hooks
// =============================================================================

/**
 * Hook for rendering a package with automatic job polling.
 */
export function useRenderMutation() {
  const { job, setJob, startPolling, stopPolling, queryClient } = useJobPollingState();

  const mutation = useMutation({
    mutationFn: submitRender,
    onSuccess: (data) => {
      setJob(data);

      // If pending, start polling
      if (data.status === 'pending' || data.status === 'processing') {
        startPolling(data.job_id);
      }

      // If already completed (cached), invalidate queries
      if (data.status === 'completed') {
        queryClient.invalidateQueries({ queryKey: queryKeys.library.all });
        queryClient.invalidateQueries({ queryKey: queryKeys.suggestions.all });
        queryClient.invalidateQueries({ queryKey: queryKeys.explore.all });
      }
    },
    onError: (error) => {
      stopPolling();
      const parsed = parseError(error, 'package');
      showError(parsed.title, parsed.message + (parsed.suggestion ? `\n\n${parsed.suggestion}` : ''));
    },
  });

  const reset = useCallback(() => {
    stopPolling();
    setJob(null);
    mutation.reset();
  }, [stopPolling, setJob, mutation]);

  return {
    job,
    isLoading: mutation.isPending || (job?.status === 'pending' || job?.status === 'processing'),
    error: mutation.error?.message || null,
    render: mutation.mutate,
    reset,
  };
}

/**
 * Hook for analyzing a repository with automatic job polling.
 */
export function useAnalyzeRepoMutation() {
  const { job, setJob, startPolling, stopPolling } = useJobPollingState();

  const mutation = useMutation({
    mutationFn: ({ owner, repo, request }: { owner: string; repo: string; request: RepoAnalyzeRequest }) =>
      analyzeRepo(owner, repo, request),
    onSuccess: (data) => {
      setJob(data);

      if (data.status === 'pending' || data.status === 'processing') {
        startPolling(data.job_id);
      }
    },
    onError: (error) => {
      stopPolling();
      const parsed = parseError(error, 'repository');
      showError(parsed.title, parsed.message + (parsed.suggestion ? `\n\n${parsed.suggestion}` : ''));
    },
  });

  const analyze = useCallback((owner: string, repo: string, request: RepoAnalyzeRequest) => {
    mutation.mutate({ owner, repo, request });
  }, [mutation]);

  const reset = useCallback(() => {
    stopPolling();
    setJob(null);
    mutation.reset();
  }, [stopPolling, setJob, mutation]);

  return {
    job,
    isLoading: mutation.isPending || (job?.status === 'pending' || job?.status === 'processing'),
    error: mutation.error?.message || null,
    analyze,
    reset,
  };
}
