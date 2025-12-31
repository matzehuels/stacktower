/**
 * Hook for polling job status until completion.
 * 
 * Automatically polls the API while a job is in progress,
 * then stops when the job completes (success or error).
 * Cleans up polling on unmount.
 */

import { useEffect, useRef } from 'react';
import { getJob } from '@/lib/api';
import { JOB_POLL_INTERVAL } from '@/config/constants';
import type { JobResponse } from '@/types/api';

export function useJobPolling(
  job: JobResponse,
  onJobUpdate: ((job: JobResponse) => void) | undefined
) {
  const pollingRef = useRef<number | null>(null);

  useEffect(() => {
    // Only poll if job is in progress
    if (job.status !== 'pending' && job.status !== 'processing') {
      if (pollingRef.current !== null) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
      return;
    }

    const pollJob = async () => {
      try {
        const updated = await getJob(job.job_id);
        if (updated.status !== job.status || updated.result !== job.result) {
          onJobUpdate?.(updated);
        }

        // Stop polling once complete
        if (updated.status === 'completed' || updated.status === 'failed') {
          if (pollingRef.current !== null) {
            clearInterval(pollingRef.current);
            pollingRef.current = null;
          }
        }
      } catch (error) {
        console.error('Failed to poll job:', error);
      }
    };

    // Start polling
    pollingRef.current = window.setInterval(pollJob, JOB_POLL_INTERVAL);

    // Cleanup on unmount or when job completes
    return () => {
      if (pollingRef.current !== null) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    };
  }, [job.job_id, job.status, job.result, onJobUpdate]);
}

