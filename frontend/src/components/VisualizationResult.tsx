import { useState, useCallback } from 'react';
import { useArtifact, getArtifactUrl, useGraphData, useAnalyze } from '../hooks/useApi';
import { DependencyList } from './DependencyList';
import { AnalysisReportView } from './AnalysisReport';
import { useAgent } from '../context/AgentContext';
import type { JobResponse, AgentEvent } from '../types/api';

interface Props {
  job: JobResponse;
  onReset: () => void;
}

export function VisualizationResult({ job, onReset }: Props) {
  const svgPath = job.result?.svg;
  const graphPath = job.result?.graph_path;
  const { data: svgData, isLoading: svgLoading } = useArtifact(job.job_id, svgPath);
  const { data: graphData, isLoading: graphLoading } = useGraphData(graphPath);
  const [showDeps, setShowDeps] = useState(false);
  const [showAnalysis, setShowAnalysis] = useState(false);
  
  // Agent context for streaming events to sidebar
  const { addEvent, setConnected, clear: clearAgentEvents, setVisible: setAgentVisible } = useAgent();
  
  // Handlers for the useAnalyze hook
  const handleAgentEvent = useCallback((event: AgentEvent) => {
    addEvent(event);
  }, [addEvent]);
  
  const handleConnectedChange = useCallback((connected: boolean) => {
    setConnected(connected);
  }, [setConnected]);
  
  // Analysis state with event forwarding to context
  const { 
    report: analyzeReport,
    isLoading: analyzeLoading, 
    error: analyzeError, 
    analyze, 
    reset: resetAnalyze 
  } = useAnalyze(handleAgentEvent, handleConnectedChange);

  const formatDuration = (ns?: number) => {
    if (!ns) return '-';
    const ms = ns / 1_000_000;
    if (ms < 1000) return `${Math.round(ms)}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  // Extract owner/repo from a GitHub URL
  const extractOwnerRepo = (url: string): string | null => {
    const match = url.match(/github\.com\/([^/]+\/[^/]+)/);
    if (match) return match[1].replace(/\.git$/, '').replace(/\/$/, '');
    return null;
  };

  // Extract package/repo name from job payload and graph data
  const getRepoFromGraph = (): string => {
    // First try job payload repo (for analyze jobs) - only if it's in owner/repo format
    const payloadRepo = (job as any).payload?.repo;
    if (payloadRepo && payloadRepo.includes('/')) return payloadRepo;
    
    // Get the package name from job payload (for visualize/parse jobs)
    const packageName = (job as any).payload?.package || payloadRepo;
    
    if (graphData?.nodes?.length) {
      // Find the node matching the package name and get its repo_url
      if (packageName) {
        const packageNode = graphData.nodes.find(n => n.id === packageName);
        if (packageNode?.meta?.repo_url) {
          const ownerRepo = extractOwnerRepo(packageNode.meta.repo_url);
          if (ownerRepo) return ownerRepo;
        }
      }
      
      // Try to find node with most stars (likely the main package being analyzed)
      const sortedByStars = [...graphData.nodes]
        .filter(n => n.meta?.repo_stars && n.meta?.repo_url)
        .sort((a, b) => (b.meta?.repo_stars || 0) - (a.meta?.repo_stars || 0));
      
      if (sortedByStars[0]?.meta?.repo_url) {
        const ownerRepo = extractOwnerRepo(sortedByStars[0].meta.repo_url);
        if (ownerRepo) return ownerRepo;
      }
    }
    
    // Fallback to package name (links won't work but at least we have a name)
    if (packageName) return packageName;
    
    // Last resort fallback
    return `analysis-${job.job_id.slice(0, 8)}`;
  };

  // Trigger AI analysis using existing graph data
  const handleAnalyze = () => {
    if (!graphPath) return;
    
    const repo = getRepoFromGraph();
    const language = (job as any).payload?.language || 'python';
    
    // Clear previous events and show agent sidebar
    clearAgentEvents();
    setAgentVisible(true);
    setShowAnalysis(true);
    
    analyze({
      repo,
      language,
      graph_path: graphPath,
    });
  };

  if (job.status === 'pending' || job.status === 'processing') {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 mx-auto mb-3 border-2 border-[var(--color-primary)] rounded-full border-t-transparent animate-spin" />
          <p className="text-sm text-[var(--color-text)]">
            {job.status === 'pending' ? 'Queued' : 'Analyzing'}
          </p>
          <p className="text-xs text-[var(--color-text-muted)] font-mono mt-1">
            {job.job_id.slice(0, 8)}
          </p>
        </div>
      </div>
    );
  }

  if (job.status === 'failed') {
    return (
      <div className="panel">
        <div className="panel-body">
          <div className="flex items-start gap-3">
            <div className="status-dot status-dot-error mt-1.5" />
            <div className="flex-1 min-w-0">
              <h3 className="text-sm font-medium text-[var(--color-text)]">Analysis Failed</h3>
              <p className="text-xs text-[var(--color-error)] font-mono mt-1 break-all">{job.error}</p>
              <button onClick={onReset} className="btn btn-secondary mt-3 text-xs">
                Try Again
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Show analysis report if we have one
  if (showAnalysis && analyzeReport) {
    return (
      <div className="h-full">
        <AnalysisReportView 
          report={analyzeReport} 
          onReset={() => {
            setShowAnalysis(false);
            resetAnalyze();
          }} 
        />
      </div>
    );
  }

  // Show analysis in progress
  if (showAnalysis && analyzeLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 mx-auto mb-3 border-2 border-[var(--color-primary)] rounded-full border-t-transparent animate-spin" />
          <p className="text-sm text-[var(--color-text)]">Analyzing with AI</p>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">
            Watch the Agent panel →
          </p>
          <button 
            onClick={() => {
              setShowAnalysis(false);
              resetAnalyze();
            }}
            className="btn btn-ghost text-xs mt-3"
          >
            Cancel
          </button>
        </div>
      </div>
    );
  }

  if (job.status === 'completed') {
    return (
      <div className="h-full flex flex-col gap-3 animate-fade-in">
        {/* Stats and actions bar */}
        <div className="flex items-center justify-between">
          {/* Stats */}
          <div className="flex items-center gap-6">
            <div>
              <span className="text-lg font-semibold text-[var(--color-text)] tabular-nums">
                {job.result?.nodes || job.result?.blocks || 0}
              </span>
              <span className="text-xs text-[var(--color-text-muted)] ml-1.5">deps</span>
            </div>
            <div>
              <span className="text-lg font-semibold text-[var(--color-text)] tabular-nums">
                {job.result?.edges || 0}
              </span>
              <span className="text-xs text-[var(--color-text-muted)] ml-1.5">edges</span>
            </div>
            <div>
              <span className="text-lg font-semibold text-[var(--color-text)] tabular-nums">
                {formatDuration(job.duration)}
              </span>
              <span className="text-xs text-[var(--color-text-muted)] ml-1.5">time</span>
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2">
            {graphPath && (
              <>
                <button
                  onClick={handleAnalyze}
                  disabled={analyzeLoading}
                  className="btn btn-primary"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                  </svg>
                  <span>AI Analyze</span>
                </button>
                <button
                  onClick={() => setShowDeps(!showDeps)}
                  className={`btn ${showDeps ? 'btn-secondary' : 'btn-ghost'}`}
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                  </svg>
                  <span>List</span>
                </button>
              </>
            )}
            <button onClick={onReset} className="btn btn-ghost">
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" />
              </svg>
              <span>New</span>
            </button>
          </div>
        </div>

        {/* Error display */}
        {analyzeError && (
          <div className="p-2.5 bg-[var(--color-error-light)] border border-[var(--color-error)]/20 rounded text-xs text-[var(--color-error)]">
            {analyzeError}
          </div>
        )}

        {/* Main content */}
        <div className="flex-1 flex gap-3 min-h-0">
          {/* Visualization panel */}
          <div className={`panel flex-1 flex flex-col min-w-0`}>
            <div className="panel-header flex items-center justify-between">
              <span>Tower</span>
              <div className="flex items-center gap-1">
                {job.result?.svg && (
                  <a href={getArtifactUrl(job.result.svg)} download className="btn btn-ghost py-1 px-2 text-[10px]">
                    SVG
                  </a>
                )}
                {job.result?.png && (
                  <a href={getArtifactUrl(job.result.png)} download className="btn btn-ghost py-1 px-2 text-[10px]">
                    PNG
                  </a>
                )}
                {job.result?.pdf && (
                  <a href={getArtifactUrl(job.result.pdf)} download className="btn btn-ghost py-1 px-2 text-[10px]">
                    PDF
                  </a>
                )}
                {graphPath && (
                  <a href={getArtifactUrl(graphPath)} download className="btn btn-ghost py-1 px-2 text-[10px]">
                    JSON
                  </a>
                )}
              </div>
            </div>
            <div className="flex-1 overflow-auto p-4 bg-[var(--color-bg)]">
              {svgLoading ? (
                <div className="h-full flex items-center justify-center">
                  <div className="w-6 h-6 border-2 border-[var(--color-primary)] rounded-full border-t-transparent animate-spin" />
                </div>
              ) : svgData ? (
                <div 
                  className="w-full"
                  dangerouslySetInnerHTML={{ __html: svgData }} 
                />
              ) : (
                <div className="h-full flex items-center justify-center text-xs text-[var(--color-text-muted)]">
                  Failed to load
                </div>
              )}
            </div>
          </div>

          {/* Dependencies sidebar */}
          {showDeps && graphPath && (
            <div className="w-80 panel flex flex-col">
              <div className="panel-header flex items-center justify-between">
                <span>Dependencies</span>
                <button 
                  onClick={() => setShowDeps(false)}
                  className="btn btn-ghost p-1 -mr-1"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <div className="flex-1 overflow-hidden">
                {graphLoading ? (
                  <div className="h-full flex items-center justify-center">
                    <div className="w-5 h-5 border-2 border-[var(--color-primary)] rounded-full border-t-transparent animate-spin" />
                  </div>
                ) : graphData ? (
                  <DependencyList data={graphData} />
                ) : (
                  <div className="h-full flex items-center justify-center text-xs text-[var(--color-text-muted)]">
                    Failed to load
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }

  return null;
}
