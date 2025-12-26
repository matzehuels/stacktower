import { useState, useCallback, useEffect, useRef } from 'react';
import type { JobResponse, VisualizeRequest, GitHubUser, GitHubRepo, ManifestFile, RepoAnalyzeRequest, GraphData, ReportSummary, ReportDocument } from '../types/api';

const API_BASE = '/api/v1';

// Graph Data Hook - fetches parsed dependency graph
export function useGraphData(graphPath: string | undefined) {
  const [data, setData] = useState<GraphData | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!graphPath) {
      setData(null);
      return;
    }

    const fetchGraph = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const res = await fetch(`${API_BASE}/artifacts/${graphPath}`);
        if (!res.ok) throw new Error('Failed to fetch graph data');
        const json: GraphData = await res.json();
        setData(json);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch graph');
      } finally {
        setIsLoading(false);
      }
    };

    fetchGraph();
  }, [graphPath]);

  return { data, isLoading, error };
}

// GitHub Auth Hook
export function useAuth() {
  const [user, setUser] = useState<GitHubUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const checkAuth = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/auth/me`, { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setUser(data);
      } else {
        setUser(null);
      }
    } catch {
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const login = useCallback(() => {
    window.location.href = `${API_BASE}/auth/github`;
  }, []);

  const logout = useCallback(async () => {
    try {
      await fetch(`${API_BASE}/auth/logout`, { 
        method: 'POST',
        credentials: 'include'
      });
      setUser(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to logout');
    }
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  return { user, isLoading, error, login, logout, checkAuth };
}

// GitHub Repos Hook
export function useRepos() {
  const [repos, setRepos] = useState<GitHubRepo[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRepos = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const res = await fetch(`${API_BASE}/repos`, { credentials: 'include' });
      if (!res.ok) {
        if (res.status === 401) {
          throw new Error('Not authenticated');
        }
        throw new Error('Failed to fetch repos');
      }
      const data = await res.json();
      setRepos(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch repos');
      setRepos([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { repos, isLoading, error, fetchRepos };
}

// Manifests Hook
export function useManifests() {
  const [manifests, setManifests] = useState<ManifestFile[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchManifests = useCallback(async (owner: string, repo: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const res = await fetch(`${API_BASE}/repos/${owner}/${repo}/manifests`, { 
        credentials: 'include' 
      });
      if (!res.ok) throw new Error('Failed to fetch manifests');
      const data = await res.json();
      setManifests(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch manifests');
      setManifests([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const reset = useCallback(() => {
    setManifests([]);
    setError(null);
  }, []);

  return { manifests, isLoading, error, fetchManifests, reset };
}

// Analyze Repo Hook
export function useAnalyzeRepo() {
  const [job, setJob] = useState<JobResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollingRef = useRef<number | null>(null);

  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  }, []);

  const pollJob = useCallback(async (jobId: string) => {
    try {
      const res = await fetch(`${API_BASE}/jobs/${jobId}`);
      if (!res.ok) throw new Error('Failed to fetch job status');
      const data: JobResponse = await res.json();
      setJob(data);

      if (data.status === 'completed' || data.status === 'failed' || data.status === 'cancelled') {
        stopPolling();
        setIsLoading(false);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to poll job');
      stopPolling();
      setIsLoading(false);
    }
  }, [stopPolling]);

  const analyze = useCallback(async (
    owner: string, 
    repo: string, 
    request: RepoAnalyzeRequest
  ) => {
    setIsLoading(true);
    setError(null);
    setJob(null);
    stopPolling();

    try {
      const res = await fetch(`${API_BASE}/repos/${owner}/${repo}/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(request),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to analyze repo');
      }

      const data: JobResponse = await res.json();
      setJob(data);

      // Start polling
      pollingRef.current = window.setInterval(() => pollJob(data.job_id), 1000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to analyze repo');
      setIsLoading(false);
    }
  }, [pollJob, stopPolling]);

  const reset = useCallback(() => {
    stopPolling();
    setJob(null);
    setError(null);
    setIsLoading(false);
  }, [stopPolling]);

  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  return { job, isLoading, error, analyze, reset };
}

export function useVisualize() {
  const [job, setJob] = useState<JobResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollingRef = useRef<number | null>(null);

  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  }, []);

  const pollJob = useCallback(async (jobId: string) => {
    try {
      const res = await fetch(`${API_BASE}/jobs/${jobId}`);
      if (!res.ok) throw new Error('Failed to fetch job status');
      const data: JobResponse = await res.json();
      setJob(data);

      if (data.status === 'completed' || data.status === 'failed' || data.status === 'cancelled') {
        stopPolling();
        setIsLoading(false);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to poll job');
      stopPolling();
      setIsLoading(false);
    }
  }, [stopPolling]);

  const submit = useCallback(async (request: VisualizeRequest) => {
    setIsLoading(true);
    setError(null);
    setJob(null);
    stopPolling();

    try {
      const res = await fetch(`${API_BASE}/visualize`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to submit job');
      }

      const data: JobResponse = await res.json();
      setJob(data);

      // Start polling
      pollingRef.current = window.setInterval(() => pollJob(data.job_id), 1000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit job');
      setIsLoading(false);
    }
  }, [pollJob, stopPolling]);

  const reset = useCallback(() => {
    stopPolling();
    setJob(null);
    setError(null);
    setIsLoading(false);
  }, [stopPolling]);

  // Cleanup on unmount
  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  return { job, isLoading, error, submit, reset };
}

export function useArtifact(jobId: string | undefined, artifact: string | undefined) {
  const [data, setData] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!jobId || !artifact) {
      setData(null);
      return;
    }

    const fetchArtifact = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const res = await fetch(`${API_BASE}/artifacts/${artifact}`);
        if (!res.ok) throw new Error('Failed to fetch artifact');
        
        const contentType = res.headers.get('content-type') || '';
        if (contentType.includes('image/svg') || artifact.endsWith('.svg')) {
          const text = await res.text();
          setData(text);
        } else {
          const blob = await res.blob();
          setData(URL.createObjectURL(blob));
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch artifact');
      } finally {
        setIsLoading(false);
      }
    };

    fetchArtifact();
  }, [jobId, artifact]);

  return { data, isLoading, error };
}

export function getArtifactUrl(artifact: string): string {
  return `${API_BASE}/artifacts/${artifact}`;
}

// =============================================================================
// Streaming / Agent Analysis
// =============================================================================

import type { AgentEvent, AnalyzeRequest, AnalysisReport } from '../types/api';

// Hook for streaming agent events
export function useAgentStream(jobId: string | null) {
  const [events, setEvents] = useState<AgentEvent[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [finalStatus, setFinalStatus] = useState<JobResponse | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!jobId) {
      setEvents([]);
      setFinalStatus(null);
      return;
    }

    // Create EventSource for SSE
    const es = new EventSource(`${API_BASE}/stream/${jobId}`);
    eventSourceRef.current = es;
    setIsConnected(true);

    // Handle different event types
    const handleEvent = (e: MessageEvent) => {
      try {
        const event: AgentEvent = JSON.parse(e.data);
        setEvents(prev => [...prev, event]);
      } catch {
        console.warn('Failed to parse event:', e.data);
      }
    };

    es.addEventListener('thinking', handleEvent);
    es.addEventListener('tool_call', handleEvent);
    es.addEventListener('tool_result', handleEvent);
    es.addEventListener('progress', handleEvent);
    es.addEventListener('complete', handleEvent);
    es.addEventListener('error', handleEvent);

    // Handle final status
    es.addEventListener('status', (e: MessageEvent) => {
      try {
        const status = JSON.parse(e.data);
        setFinalStatus(status);
      } catch {
        console.warn('Failed to parse status:', e.data);
      }
    });

    // Handle connection close
    es.onerror = () => {
      setIsConnected(false);
      es.close();
    };

    return () => {
      es.close();
      setIsConnected(false);
    };
  }, [jobId]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      setIsConnected(false);
    }
  }, []);

  return { events, isConnected, finalStatus, disconnect };
}

// Hook for running an analysis job with streaming
// This version can optionally use an external event handler (from context)
export function useAnalyze(onEvent?: (event: AgentEvent) => void, onConnectedChange?: (connected: boolean) => void) {
  const [job, setJob] = useState<JobResponse | null>(null);
  const [events, setEvents] = useState<AgentEvent[]>([]);
  const [report, setReport] = useState<AnalysisReport | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  const analyze = useCallback(async (request: AnalyzeRequest) => {
    setIsLoading(true);
    setError(null);
    setEvents([]);
    setReport(null);
    onConnectedChange?.(true);

    // Close any existing stream
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    try {
      // Submit analysis job
      const res = await fetch(`${API_BASE}/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });

      if (!res.ok) {
        const errorData = await res.json();
        throw new Error(errorData.error || 'Failed to start analysis');
      }

      const jobData: JobResponse = await res.json();
      setJob(jobData);

      // Start streaming events
      console.log('[useAnalyze] Connecting to event stream:', `${API_BASE}/stream/${jobData.job_id}`);
      const es = new EventSource(`${API_BASE}/stream/${jobData.job_id}`);
      eventSourceRef.current = es;

      const handleEvent = (e: MessageEvent) => {
        console.log('[useAnalyze] Received event:', e.type, e.data?.substring(0, 100));
        try {
          const event: AgentEvent = JSON.parse(e.data);
          setEvents(prev => [...prev, event]);
          // Also push to external handler (context)
          onEvent?.(event);
          
          // Check for terminal events
          if (event.type === 'complete' || event.type === 'error') {
            console.log('[useAnalyze] Terminal event received:', event.type);
            onConnectedChange?.(false);
          }
        } catch (err) {
          console.error('[useAnalyze] Failed to parse event:', err);
        }
      };

      // Listen for named events
      es.addEventListener('thinking', handleEvent);
      es.addEventListener('tool_call', handleEvent);
      es.addEventListener('tool_result', handleEvent);
      es.addEventListener('progress', handleEvent);
      es.addEventListener('complete', handleEvent);
      es.addEventListener('error', handleEvent);

      // Also listen for generic message as fallback
      es.onmessage = (e: MessageEvent) => {
        console.log('[useAnalyze] Generic message:', e.data?.substring(0, 100));
        handleEvent(e);
      };

      es.addEventListener('status', async (e: MessageEvent) => {
        console.log('[useAnalyze] Status event:', e.data);
        try {
          const status = JSON.parse(e.data);
          setJob(prev => prev ? { ...prev, ...status } : status);
          
          // If complete, fetch the report from the reports endpoint
          if (status.status === 'completed') {
            console.log('[useAnalyze] Fetching report for job:', status.job_id);
            const reportRes = await fetch(`${API_BASE}/reports/${status.job_id}`);
            if (reportRes.ok) {
              const reportDoc = await reportRes.json();
              console.log('[useAnalyze] Report loaded successfully');
              setReport(reportDoc.report);
            } else {
              // Fallback to old artifact path method
              if (status.result?.report_path) {
                console.log('[useAnalyze] Falling back to artifact path:', status.result.report_path);
                const fallbackRes = await fetch(`${API_BASE}/artifacts/${status.result.report_path}`);
                if (fallbackRes.ok) {
                  const reportData = await fallbackRes.json();
                  setReport(reportData);
                }
              }
            }
          }
          
          setIsLoading(false);
          onConnectedChange?.(false);
          es.close();
        } catch (err) {
          console.error('[useAnalyze] Failed to parse status:', err);
        }
      });

      es.onopen = () => {
        console.log('[useAnalyze] EventSource connected');
      };

      es.onerror = (err) => {
        console.error('[useAnalyze] EventSource error:', err);
        setIsLoading(false);
        onConnectedChange?.(false);
        es.close();
      };

    } catch (err) {
      setError(err instanceof Error ? err.message : 'Analysis failed');
      setIsLoading(false);
      onConnectedChange?.(false);
    }
  }, [onEvent, onConnectedChange]);

  const reset = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }
    setJob(null);
    setEvents([]);
    setReport(null);
    setError(null);
    setIsLoading(false);
    onConnectedChange?.(false);
  }, [onConnectedChange]);

  return { job, events, report, isLoading, error, analyze, reset };
}

// Hook for listing reports
export function useReports(limit = 20) {
  const [reports, setReports] = useState<ReportSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchReports = useCallback(async (offset = 0) => {
    setIsLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/reports?limit=${limit}&offset=${offset}`);
      if (!res.ok) throw new Error('Failed to fetch reports');
      const data = await res.json();
      setReports(data.reports || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load reports');
    } finally {
      setIsLoading(false);
    }
  }, [limit]);

  useEffect(() => {
    fetchReports();
  }, [fetchReports]);

  return { reports, total, isLoading, error, refresh: fetchReports };
}

// Hook for fetching a single report
export function useReport(jobId: string | undefined) {
  const [report, setReport] = useState<ReportDocument | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!jobId) return;

    const fetchReport = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const res = await fetch(`${API_BASE}/reports/${jobId}`);
        if (!res.ok) throw new Error('Report not found');
        const data = await res.json();
        setReport(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load report');
      } finally {
        setIsLoading(false);
      }
    };

    fetchReport();
  }, [jobId]);

  return { report, isLoading, error };
}
