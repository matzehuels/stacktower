import { useReports, useReport } from '../hooks/useApi';
import { AnalysisReportView } from './AnalysisReport';
import { useState } from 'react';

// Helper to safely format dates from various formats
function formatDate(dateValue: string | number | Date | undefined): string {
  if (!dateValue) return '-';
  try {
    const date = new Date(dateValue);
    if (isNaN(date.getTime())) return '-';
    return date.toLocaleDateString('en-US', { 
      month: 'short', 
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  } catch {
    return '-';
  }
}

function formatDuration(ns?: number): string {
  if (!ns) return '-';
  const ms = ns / 1_000_000;
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export function ReportHistory() {
  const { reports, total, isLoading, error, refresh } = useReports(20);
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const { report: selectedReport, isLoading: reportLoading, error: reportError } = useReport(selectedJobId || undefined);

  // Handle click on a report
  const handleReportClick = (report: typeof reports[0]) => {
    if (report.status === 'completed') {
      setSelectedJobId(report.job_id);
    }
  };

  // Show loading state when fetching selected report
  if (selectedJobId && reportLoading) {
    return (
      <div className="h-full flex flex-col">
        <div className="flex items-center gap-3 mb-4">
          <button onClick={() => setSelectedJobId(null)} className="btn btn-ghost">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 19l-7-7 7-7" />
            </svg>
            Back
          </button>
        </div>
        <div className="flex-1 flex items-center justify-center">
          <div className="w-6 h-6 border-2 border-[var(--color-primary)] rounded-full border-t-transparent animate-spin" />
        </div>
      </div>
    );
  }

  // Show error state if report fetch failed
  if (selectedJobId && reportError) {
    return (
      <div className="h-full flex flex-col">
        <div className="flex items-center gap-3 mb-4">
          <button onClick={() => setSelectedJobId(null)} className="btn btn-ghost">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 19l-7-7 7-7" />
            </svg>
            Back
          </button>
        </div>
        <div className="panel">
          <div className="panel-body text-center py-8">
            <p className="text-xs text-[var(--color-error)] mb-4">{reportError}</p>
            <button onClick={() => setSelectedJobId(null)} className="btn btn-secondary">
              Back to List
            </button>
          </div>
        </div>
      </div>
    );
  }

  // Show selected report
  if (selectedJobId && selectedReport?.report) {
    return (
      <div className="h-full flex flex-col">
        <div className="flex items-center gap-3 mb-4">
          <button onClick={() => setSelectedJobId(null)} className="btn btn-ghost">
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 19l-7-7 7-7" />
            </svg>
            Back
          </button>
          <span className="text-xs text-[var(--color-text-muted)]">
            {selectedReport.repo}
          </span>
        </div>
        <div className="flex-1 min-h-0">
          <AnalysisReportView 
            report={selectedReport.report} 
            onReset={() => setSelectedJobId(null)} 
          />
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="w-6 h-6 border-2 border-[var(--color-primary)] rounded-full border-t-transparent animate-spin" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="panel">
        <div className="panel-body text-center py-8">
          <p className="text-xs text-[var(--color-error)]">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col animate-fade-in">
      {/* Header */}
      <div className="section-header">
        <div className="section-title">
          <svg className="w-4 h-4 text-[var(--color-text-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Analysis History</span>
          <span className="section-count">{total}</span>
        </div>
        <button onClick={() => refresh()} className="btn btn-ghost">
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </button>
      </div>

      {/* Reports table */}
      {reports.length === 0 ? (
        <div className="empty-state flex-1">
          <div className="empty-state-icon">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          </div>
          <div className="empty-state-title">No reports yet</div>
          <div className="empty-state-text">Run an AI analysis to see reports here</div>
        </div>
      ) : (
        <div className="panel flex-1 overflow-hidden flex flex-col">
          <div className="overflow-auto flex-1">
            <table className="table">
              <thead>
                <tr>
                  <th className="w-8"></th>
                  <th>Repository</th>
                  <th className="w-20 text-center">Score</th>
                  <th className="w-24 text-center">Issues</th>
                  <th className="w-20 text-right">Time</th>
                  <th className="w-32 text-right">Date</th>
                </tr>
              </thead>
              <tbody>
                {reports.map((report) => (
                  <tr 
                    key={report.job_id}
                    onClick={() => handleReportClick(report)}
                    className={report.status === 'completed' ? 'cursor-pointer' : 'opacity-60'}
                  >
                    {/* Status indicator */}
                    <td className="w-8">
                      <div className={`status-dot ${
                        report.status === 'completed' ? 'status-dot-success' :
                        report.status === 'failed' ? 'status-dot-error' : 'status-dot-neutral'
                      }`} />
                    </td>
                    
                    {/* Repository */}
                    <td>
                      <div className="font-medium text-[var(--color-text)]">
                        {report.repo}
                      </div>
                      <div className="text-[10px] text-[var(--color-text-muted)] mt-0.5">
                        {report.language} • {report.total_dependencies || 0} deps
                      </div>
                    </td>
                    
                    {/* Health Score */}
                    <td className="text-center">
                      {report.status === 'completed' && report.health_score !== undefined ? (
                        <span className={`health-score text-sm ${
                          report.health_score >= 80 ? 'good' :
                          report.health_score >= 60 ? 'warning' : 'bad'
                        }`}>
                          {report.health_score}
                        </span>
                      ) : report.status === 'failed' ? (
                        <span className="text-xs text-[var(--color-error)]">Error</span>
                      ) : (
                        <span className="text-xs text-[var(--color-text-muted)]">-</span>
                      )}
                    </td>
                    
                    {/* Issues */}
                    <td className="text-center">
                      {report.status === 'completed' ? (
                        <div className="flex items-center justify-center gap-2 text-[10px]">
                          {(report.critical_count || 0) > 0 && (
                            <span className="severity severity-critical">{report.critical_count}</span>
                          )}
                          {(report.high_count || 0) > 0 && (
                            <span className="severity severity-high">{report.high_count}</span>
                          )}
                          {!(report.critical_count || report.high_count) && (
                            <span className="text-[var(--color-success)]">✓</span>
                          )}
                        </div>
                      ) : (
                        <span className="text-xs text-[var(--color-text-muted)]">-</span>
                      )}
                    </td>
                    
                    {/* Duration */}
                    <td className="text-right text-xs text-[var(--color-text-muted)] font-mono">
                      {formatDuration(report.duration)}
                    </td>
                    
                    {/* Date */}
                    <td className="text-right text-xs text-[var(--color-text-muted)]">
                      {formatDate(report.created_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
