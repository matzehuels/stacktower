import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import type { AnalysisReport, DependencyAnalysis, Recommendation, Alternative, Usage } from '../types/api';

interface AnalysisReportProps {
  report: AnalysisReport;
  onReset: () => void;
}

export function AnalysisReportView({ report, onReset }: AnalysisReportProps) {
  const [selectedDep, setSelectedDep] = useState<string | null>(null);
  const [view, setView] = useState<'overview' | 'narrative' | 'recommendations'>('overview');

  const selectedDepData = selectedDep 
    ? report.dependencies.find(d => d.name === selectedDep) 
    : null;

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-4">
          <div>
            <h2 className="text-lg font-semibold text-[var(--color-text)]">
              {report.repo}
            </h2>
            <p className="text-sm text-[var(--color-text-muted)]">
              {report.language} • {report.summary.total_dependencies} dependencies • {formatDuration(report.duration)}
            </p>
          </div>
          <HealthBadge score={report.summary.health_score} />
        </div>
        <button onClick={onReset} className="btn btn-ghost">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Main content - 2 column layout */}
      <div className="flex-1 flex min-h-0">
        {/* Left: Dependency List */}
        <div className="w-80 border-r border-[var(--color-border)] flex flex-col">
          <div className="p-3 border-b border-[var(--color-border)] bg-[var(--color-bg-elevated)]">
            <h3 className="text-sm font-semibold text-[var(--color-text)]">
              Direct Dependencies ({report.summary.direct_dependencies})
            </h3>
          </div>
          <div className="flex-1 overflow-y-auto">
            {report.dependencies.map(dep => (
              <DependencyListItem
                key={dep.name}
                dep={dep}
                isSelected={selectedDep === dep.name}
                onClick={() => setSelectedDep(dep.name === selectedDep ? null : dep.name)}
              />
            ))}
          </div>
        </div>

        {/* Right: Detail View */}
        <div className="flex-1 flex flex-col min-w-0">
          {/* View Tabs */}
          <div className="flex border-b border-[var(--color-border)]">
            {(['overview', 'narrative', 'recommendations'] as const).map(tab => (
              <button
                key={tab}
                onClick={() => { setView(tab); setSelectedDep(null); }}
                className={`px-4 py-3 text-sm font-medium transition-colors ${
                  view === tab && !selectedDep
                    ? 'border-b-2 border-[var(--color-primary)] text-[var(--color-primary)]'
                    : 'text-[var(--color-text-muted)] hover:text-[var(--color-text)]'
                }`}
              >
                {tab === 'overview' ? 'Overview' : tab === 'narrative' ? 'Usage Narrative' : 'Recommendations'}
                {tab === 'recommendations' && report.recommendations.length > 0 && (
                  <span className="ml-2 px-1.5 py-0.5 text-xs bg-[var(--color-primary-light)] text-[var(--color-primary)] rounded">
                    {report.recommendations.length}
                  </span>
                )}
              </button>
            ))}
          </div>

          {/* Content Area */}
          <div className="flex-1 overflow-y-auto p-6">
            {selectedDep && selectedDepData ? (
              <DependencyDetail 
                dep={selectedDepData} 
                recommendation={report.recommendations.find(r => r.package === selectedDep)}
                onClose={() => setSelectedDep(null)}
              />
            ) : view === 'overview' ? (
              <OverviewView report={report} />
            ) : view === 'narrative' ? (
              <NarrativeView narrative={report.direct_dependency_narrative} />
            ) : (
              <RecommendationsView recommendations={report.recommendations} />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Health Badge
// =============================================================================

function HealthBadge({ score }: { score: number }) {
  const getColor = () => {
    if (score >= 80) return 'bg-emerald-500';
    if (score >= 60) return 'bg-amber-500';
    return 'bg-red-500';
  };

  return (
    <div className={`px-3 py-1.5 rounded-full ${getColor()} text-white font-bold text-sm`}>
      {score}/100
    </div>
  );
}

// =============================================================================
// Dependency List Item
// =============================================================================

function DependencyListItem({ 
  dep, 
  isSelected, 
  onClick 
}: { 
  dep: DependencyAnalysis; 
  isSelected: boolean; 
  onClick: () => void;
}) {
  const getAssessmentColor = () => {
    switch (dep.assessment) {
      case 'healthy': return 'bg-emerald-500';
      case 'caution': return 'bg-amber-500';
      case 'warning': return 'bg-orange-500';
      case 'critical': return 'bg-red-500';
      default: return 'bg-gray-400';
    }
  };

  const getCriticalityIcon = () => {
    switch (dep.context.criticality) {
      case 'essential': return '🔥';
      case 'important': return '⭐';
      case 'convenient': return '✨';
      case 'minimal': return '·';
      case 'unused': return '⚪';
      default: return '';
    }
  };

  return (
    <button
      onClick={onClick}
      className={`w-full text-left p-3 border-b border-[var(--color-border)] transition-colors ${
        isSelected 
          ? 'bg-[var(--color-primary-light)]' 
          : 'hover:bg-[var(--color-bg-hover)]'
      }`}
    >
      <div className="flex items-center gap-2">
        <span className={`w-2 h-2 rounded-full ${getAssessmentColor()}`} />
        <span className="font-mono text-sm font-medium text-[var(--color-text)] truncate flex-1">
          {dep.name}
        </span>
        <span className="text-xs" title={dep.context.criticality}>
          {getCriticalityIcon()}
        </span>
      </div>
      <div className="flex items-center gap-2 mt-1">
        {dep.version && (
          <span className="text-xs text-[var(--color-text-muted)]">v{dep.version}</span>
        )}
        {dep.health?.stars && dep.health.stars > 0 && (
          <span className="text-xs text-[var(--color-text-muted)]">
            ⭐ {formatNumber(dep.health.stars)}
          </span>
        )}
        <span className="text-xs text-[var(--color-text-muted)] capitalize">
          {dep.context.coupling}
        </span>
      </div>
      <p className="text-xs text-[var(--color-text-muted)] mt-1 line-clamp-2">
        {dep.context.purpose}
      </p>
    </button>
  );
}

// =============================================================================
// Dependency Detail View
// =============================================================================

function DependencyDetail({ 
  dep, 
  recommendation, 
  onClose 
}: { 
  dep: DependencyAnalysis;
  recommendation?: Recommendation;
  onClose: () => void;
}) {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h3 className="text-xl font-bold text-[var(--color-text)]">{dep.name}</h3>
            {dep.version && (
              <span className="badge badge-neutral">v{dep.version}</span>
            )}
            <AssessmentBadge assessment={dep.assessment} />
          </div>
          <p className="text-[var(--color-text-secondary)] mt-1">{dep.context.purpose}</p>
        </div>
        <button onClick={onClose} className="btn btn-ghost btn-sm">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-4 gap-4">
        <StatBox label="Criticality" value={dep.context.criticality} />
        <StatBox label="Coupling" value={dep.context.coupling} />
        <StatBox label="Files" value={dep.context.file_count} />
        <StatBox label="Health" value={dep.health?.score || '—'} />
      </div>

      {/* Usage Pattern */}
      <div className="panel">
        <div className="panel-header">Usage Pattern</div>
        <div className="panel-body">
          <p className="text-sm text-[var(--color-text-secondary)]">
            {dep.context.usage_pattern}
          </p>
        </div>
      </div>

      {/* Code Usages */}
      {dep.context.usages && dep.context.usages.length > 0 && (
        <div className="panel">
          <div className="panel-header flex items-center gap-2">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
            </svg>
            Code References ({dep.context.usages.length})
          </div>
          <div className="panel-body space-y-4">
            {dep.context.usages.map((usage, i) => (
              <CodeSnippet key={i} usage={usage} />
            ))}
          </div>
        </div>
      )}

      {/* Health Info */}
      <div className="panel">
        <div className="panel-header">Health & Metadata</div>
        <div className="panel-body">
          <div className="grid grid-cols-2 gap-4 text-sm">
            {dep.health?.stars !== undefined && (
              <div>
                <span className="text-[var(--color-text-muted)]">Stars:</span>
                <span className="ml-2 font-medium">⭐ {formatNumber(dep.health.stars)}</span>
              </div>
            )}
            {dep.health?.license && (
              <div>
                <span className="text-[var(--color-text-muted)]">License:</span>
                <span className="ml-2 font-medium">{dep.health.license}</span>
              </div>
            )}
            {dep.health?.last_commit && (
              <div>
                <span className="text-[var(--color-text-muted)]">Last Commit:</span>
                <span className="ml-2 font-medium">{dep.health.last_commit}</span>
              </div>
            )}
            {dep.health?.repo_url && (
              <div className="col-span-2">
                <a 
                  href={dep.health.repo_url} 
                  target="_blank" 
                  rel="noopener noreferrer"
                  className="text-[var(--color-primary)] hover:underline text-sm"
                >
                  {dep.health.repo_url}
                </a>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Issues */}
      {dep.issues && dep.issues.length > 0 && (
        <div className="panel border-[var(--color-warning)]">
          <div className="panel-header text-[var(--color-warning)]">⚠️ Issues</div>
          <div className="panel-body">
            <ul className="space-y-1">
              {dep.issues.map((issue, i) => (
                <li key={i} className="text-sm text-[var(--color-text-secondary)]">• {issue}</li>
              ))}
            </ul>
          </div>
        </div>
      )}

      {/* Recommendation */}
      {recommendation && (
        <RecommendationCard rec={recommendation} />
      )}
    </div>
  );
}

// =============================================================================
// Overview View
// =============================================================================

function OverviewView({ report }: { report: AnalysisReport }) {
  return (
    <div className="space-y-6">
      {/* Executive Summary */}
      <div className="panel">
        <div className="panel-header">Executive Summary</div>
        <div className="panel-body">
          <p className="text-[var(--color-text-secondary)]">{report.summary.executive_summary}</p>
        </div>
      </div>

      {/* Health Breakdown */}
      <div className="grid grid-cols-4 gap-4">
        <StatCard value={report.summary.healthy} label="Healthy" color="success" />
        <StatCard value={report.summary.caution} label="Caution" color="warning" />
        <StatCard value={report.summary.warning} label="Warning" color="orange" />
        <StatCard value={report.summary.critical} label="Critical" color="error" />
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-3 gap-4">
        <div className="panel p-4 text-center">
          <div className="text-3xl font-bold text-[var(--color-text)]">{report.summary.direct_dependencies}</div>
          <div className="text-sm text-[var(--color-text-muted)]">Direct Dependencies</div>
        </div>
        <div className="panel p-4 text-center">
          <div className="text-3xl font-bold text-[var(--color-text)]">{report.summary.total_dependencies}</div>
          <div className="text-sm text-[var(--color-text-muted)]">Total (incl. transitive)</div>
        </div>
        <div className="panel p-4 text-center">
          <div className="text-3xl font-bold text-[var(--color-text)]">{report.recommendations.length}</div>
          <div className="text-sm text-[var(--color-text-muted)]">Recommendations</div>
        </div>
      </div>

      {/* Analysis Metadata */}
      <div className="panel">
        <div className="panel-header">Analysis Details</div>
        <div className="panel-body grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-[var(--color-text-muted)]">Analyzed:</span>
            <span className="ml-2">{new Date(report.analyzed_at).toLocaleString()}</span>
          </div>
          <div>
            <span className="text-[var(--color-text-muted)]">Duration:</span>
            <span className="ml-2">{formatDuration(report.duration)}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Narrative View
// =============================================================================

function NarrativeView({ narrative }: { narrative: string }) {
  return (
    <div className="prose prose-sm max-w-none">
      <ReactMarkdown
        components={{
          h2: ({ children }) => (
            <h2 className="text-lg font-bold text-[var(--color-text)] mt-6 mb-2 pb-2 border-b border-[var(--color-border)]">
              {children}
            </h2>
          ),
          p: ({ children }) => (
            <p className="text-[var(--color-text-secondary)] mb-4 leading-relaxed">{children}</p>
          ),
          code: ({ children }) => (
            <code className="px-1.5 py-0.5 bg-[var(--color-bg-elevated)] rounded text-[var(--color-primary)] font-mono text-sm">
              {children}
            </code>
          ),
        }}
      >
        {narrative}
      </ReactMarkdown>
    </div>
  );
}

// =============================================================================
// Recommendations View
// =============================================================================

function RecommendationsView({ recommendations }: { recommendations: Recommendation[] }) {
  if (recommendations.length === 0) {
    return (
      <div className="text-center py-12">
        <div className="text-4xl mb-4">🎉</div>
        <h3 className="text-lg font-semibold text-[var(--color-text)]">All Clear!</h3>
        <p className="text-[var(--color-text-muted)]">No recommendations - your dependencies look healthy.</p>
      </div>
    );
  }

  // Group by priority
  const grouped = recommendations.reduce((acc, rec) => {
    acc[rec.priority] = acc[rec.priority] || [];
    acc[rec.priority].push(rec);
    return acc;
  }, {} as Record<string, Recommendation[]>);

  const priorityOrder = ['critical', 'high', 'medium', 'low'];

  return (
    <div className="space-y-6">
      {priorityOrder.map(priority => {
        const recs = grouped[priority];
        if (!recs || recs.length === 0) return null;
        return (
          <div key={priority}>
            <h3 className="text-sm font-semibold text-[var(--color-text-muted)] uppercase tracking-wide mb-3">
              {priority} Priority ({recs.length})
            </h3>
            <div className="space-y-4">
              {recs.map((rec, i) => (
                <RecommendationCard key={i} rec={rec} />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// =============================================================================
// Recommendation Card
// =============================================================================

function RecommendationCard({ rec }: { rec: Recommendation }) {
  const [expanded, setExpanded] = useState(false);

  const getActionIcon = () => {
    switch (rec.action) {
      case 'keep': return '✅';
      case 'remove': return '🗑️';
      case 'replace': return '🔄';
      case 'update': return '⬆️';
      case 'monitor': return '👀';
      case 'migrate': return '🚀';
      case 'secure': return '🔒';
      case 'build_own': return '🛠️';
      case 'refactor': return '✍️';
      default: return '📋';
    }
  };

  const getPriorityColor = () => {
    switch (rec.priority) {
      case 'critical': return 'border-l-red-500';
      case 'high': return 'border-l-orange-500';
      case 'medium': return 'border-l-amber-500';
      case 'low': return 'border-l-blue-500';
      default: return 'border-l-gray-400';
    }
  };

  return (
    <div className={`panel border-l-4 ${getPriorityColor()}`}>
      <div className="p-4">
        <div className="flex items-start gap-3">
          <span className="text-xl">{getActionIcon()}</span>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="font-mono font-semibold text-[var(--color-text)]">{rec.package}</span>
              <span className="badge badge-neutral text-xs">{rec.action}</span>
              <span className="badge badge-neutral text-xs">{rec.effort}</span>
            </div>
            <h4 className="font-medium text-[var(--color-text)] mt-1">{rec.title}</h4>
            <p className="text-sm text-[var(--color-text-secondary)] mt-1">{rec.reason}</p>

            {/* Transitive issue info */}
            {rec.triggered_by && (
              <div className="mt-3 p-2 bg-[var(--color-warning-light)] rounded text-sm">
                <span className="font-medium">Triggered by:</span> {rec.triggered_by}
                {rec.triggered_by_reason && (
                  <span className="block text-xs mt-1">{rec.triggered_by_reason}</span>
                )}
                {rec.dependency_chain && rec.dependency_chain.length > 0 && (
                  <div className="mt-2">
                    {rec.dependency_chain.map((chain, i) => (
                      <span key={i} className="block font-mono text-xs">{chain}</span>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Alternatives */}
            {rec.alternatives && rec.alternatives.length > 0 && (
              <div className="mt-4">
                <button 
                  onClick={() => setExpanded(!expanded)}
                  className="text-sm font-medium text-[var(--color-primary)] hover:underline flex items-center gap-1"
                >
                  {expanded ? '▼' : '▶'} {rec.alternatives.length} Alternative{rec.alternatives.length > 1 ? 's' : ''}
                </button>
                {expanded && (
                  <div className="mt-3 space-y-3">
                    {rec.alternatives.map((alt, i) => (
                      <AlternativeCard key={i} alt={alt} />
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Migration Steps */}
            {rec.migration_steps && rec.migration_steps.length > 0 && (
              <div className="mt-4">
                <h5 className="text-sm font-medium text-[var(--color-text)] mb-2">Migration Steps:</h5>
                <ol className="list-decimal list-inside space-y-1">
                  {rec.migration_steps.map((step, i) => (
                    <li key={i} className="text-sm text-[var(--color-text-secondary)]">{step}</li>
                  ))}
                </ol>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Alternative Card
// =============================================================================

function AlternativeCard({ alt }: { alt: Alternative }) {
  return (
    <div className="p-3 bg-[var(--color-bg)] rounded border border-[var(--color-border)]">
      <div className="flex items-center gap-2 flex-wrap">
        <span className="font-mono font-semibold text-[var(--color-text)]">{alt.name}</span>
        {alt.repo_stars && (
          <span className="text-xs text-[var(--color-text-muted)]">⭐ {formatNumber(alt.repo_stars)}</span>
        )}
        {alt.license && (
          <span className="badge badge-neutral text-xs">{alt.license}</span>
        )}
        {alt.repo_archived && (
          <span className="badge badge-warning text-xs">Archived</span>
        )}
      </div>
      
      {alt.description && (
        <p className="text-sm text-[var(--color-text-secondary)] mt-1">{alt.description}</p>
      )}
      <p className="text-sm text-[var(--color-primary)] mt-1">{alt.reason}</p>
      
      {/* Links */}
      <div className="flex gap-3 mt-2">
        {alt.repo_url && (
          <a 
            href={alt.repo_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-[var(--color-text-muted)] hover:text-[var(--color-primary)] flex items-center gap-1"
          >
            <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
            </svg>
            GitHub
          </a>
        )}
        {alt.registry_url && (
          <a 
            href={alt.registry_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-[var(--color-text-muted)] hover:text-[var(--color-primary)] flex items-center gap-1"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            Registry
          </a>
        )}
      </div>

      {/* Stats comparison */}
      {(alt.repo_last_commit || alt.repo_last_release) && (
        <div className="flex gap-4 mt-2 text-xs text-[var(--color-text-muted)]">
          {alt.repo_last_commit && <span>Last commit: {alt.repo_last_commit}</span>}
          {alt.repo_last_release && <span>Last release: {alt.repo_last_release}</span>}
        </div>
      )}
    </div>
  );
}

// =============================================================================
// Code Snippet Component
// =============================================================================

function CodeSnippet({ usage }: { usage: Usage }) {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = () => {
    if (usage.snippet) {
      navigator.clipboard.writeText(usage.snippet);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  // Get file extension for syntax hint
  const getFileIcon = (file: string) => {
    if (file.endsWith('.py')) return '🐍';
    if (file.endsWith('.js') || file.endsWith('.ts') || file.endsWith('.tsx')) return '📜';
    if (file.endsWith('.go')) return '🔵';
    if (file.endsWith('.rs')) return '🦀';
    if (file.endsWith('.rb')) return '💎';
    if (file.endsWith('.php')) return '🐘';
    if (file.endsWith('.java')) return '☕';
    return '📄';
  };

  // Use URL directly from the usage (agent provides it)
  const githubUrl = usage.url || null;

  return (
    <div className="rounded-lg border border-[var(--color-border)] overflow-hidden bg-[#1e1e1e]">
      {/* File header */}
      <div className="flex items-center justify-between px-3 py-2 bg-[#2d2d2d] border-b border-[var(--color-border)]">
        <div className="flex items-center gap-2">
          <span>{getFileIcon(usage.file)}</span>
          {githubUrl ? (
            <a 
              href={githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="font-mono text-xs text-blue-400 hover:text-blue-300 hover:underline flex items-center gap-1"
              title="View on GitHub"
            >
              {usage.file}
              {usage.line && <span className="text-gray-500">:{usage.line}</span>}
              <svg className="w-3 h-3 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
              </svg>
            </a>
          ) : (
            <span className="font-mono text-xs text-gray-300">
              {usage.file}
              {usage.line && <span className="text-gray-500">:{usage.line}</span>}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {githubUrl && (
            <a
              href={githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-gray-400 hover:text-gray-200 transition-colors"
              title="View on GitHub"
            >
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                <path fillRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clipRule="evenodd" />
              </svg>
            </a>
          )}
          <button
            onClick={copyToClipboard}
            className="text-xs text-gray-400 hover:text-gray-200 transition-colors"
            title="Copy code"
          >
            {copied ? (
              <span className="text-emerald-400">✓</span>
            ) : (
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
            )}
          </button>
        </div>
      </div>
      
      {/* Code content */}
      {usage.snippet && (
        <div className="overflow-x-auto">
          <pre className="p-3 text-sm font-mono leading-relaxed text-gray-200 whitespace-pre">
            {usage.snippet.split('\n').map((line: string, i: number) => (
              <div key={i} className="flex">
                <span className="w-8 text-right pr-3 text-gray-600 select-none text-xs">
                  {(usage.line || 1) + i}
                </span>
                <span className="flex-1">{highlightCode(line)}</span>
              </div>
            ))}
          </pre>
        </div>
      )}
      
      {/* Purpose footer */}
      {usage.purpose && (
        <div className="px-3 py-2 bg-[#2d2d2d] border-t border-[var(--color-border)]">
          <p className="text-xs text-gray-400">
            <span className="text-emerald-400 font-medium">Purpose:</span> {usage.purpose}
          </p>
        </div>
      )}
    </div>
  );
}

// Simple syntax highlighting (keywords, strings, comments)
function highlightCode(line: string): React.ReactNode {
  // Very basic highlighting - can be enhanced
  const parts: React.ReactNode[] = [];
  let remaining = line;
  let key = 0;

  // Keywords
  const keywords = /\b(import|from|def|class|return|if|else|for|while|try|except|async|await|const|let|var|function|export|default|interface|type|struct|fn|pub|use|mod)\b/g;
  
  // Process the line
  const matches = [...remaining.matchAll(keywords)];
  let lastIndex = 0;

  for (const match of matches) {
    if (match.index !== undefined && match.index > lastIndex) {
      parts.push(<span key={key++}>{remaining.slice(lastIndex, match.index)}</span>);
    }
    parts.push(
      <span key={key++} className="text-purple-400 font-medium">
        {match[0]}
      </span>
    );
    lastIndex = (match.index || 0) + match[0].length;
  }

  if (lastIndex < remaining.length) {
    // Check for strings
    const rest = remaining.slice(lastIndex);
    const stringMatch = rest.match(/(['"`]).*?\1/);
    if (stringMatch) {
      const idx = rest.indexOf(stringMatch[0]);
      if (idx > 0) parts.push(<span key={key++}>{rest.slice(0, idx)}</span>);
      parts.push(
        <span key={key++} className="text-emerald-400">
          {stringMatch[0]}
        </span>
      );
      parts.push(<span key={key++}>{rest.slice(idx + stringMatch[0].length)}</span>);
    } else if (rest.includes('#') || rest.includes('//')) {
      // Comments
      const commentIdx = Math.min(
        rest.includes('#') ? rest.indexOf('#') : Infinity,
        rest.includes('//') ? rest.indexOf('//') : Infinity
      );
      if (commentIdx < rest.length) {
        parts.push(<span key={key++}>{rest.slice(0, commentIdx)}</span>);
        parts.push(
          <span key={key++} className="text-gray-500 italic">
            {rest.slice(commentIdx)}
          </span>
        );
      } else {
        parts.push(<span key={key++}>{rest}</span>);
      }
    } else {
      parts.push(<span key={key++}>{rest}</span>);
    }
  }

  return parts.length > 0 ? parts : line;
}

// =============================================================================
// Utility Components
// =============================================================================

function AssessmentBadge({ assessment }: { assessment: string }) {
  const getStyles = () => {
    switch (assessment) {
      case 'healthy': return 'bg-emerald-100 text-emerald-700';
      case 'caution': return 'bg-amber-100 text-amber-700';
      case 'warning': return 'bg-orange-100 text-orange-700';
      case 'critical': return 'bg-red-100 text-red-700';
      default: return 'bg-gray-100 text-gray-700';
    }
  };

  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${getStyles()}`}>
      {assessment}
    </span>
  );
}

function StatBox({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="panel p-3 text-center">
      <div className="text-lg font-bold text-[var(--color-text)] capitalize">{value}</div>
      <div className="text-xs text-[var(--color-text-muted)]">{label}</div>
    </div>
  );
}

function StatCard({ value, label, color }: { value: number; label: string; color: 'success' | 'warning' | 'orange' | 'error' }) {
  const getColor = () => {
    switch (color) {
      case 'success': return 'text-emerald-500';
      case 'warning': return 'text-amber-500';
      case 'orange': return 'text-orange-500';
      case 'error': return 'text-red-500';
    }
  };

  return (
    <div className="panel p-4 text-center">
      <div className={`text-2xl font-bold ${getColor()}`}>{value}</div>
      <div className="text-xs text-[var(--color-text-muted)]">{label}</div>
    </div>
  );
}

// =============================================================================
// Utilities
// =============================================================================

function formatDuration(ns: number): string {
  const seconds = ns / 1_000_000_000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const minutes = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${minutes}m ${secs}s`;
}

function formatNumber(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return n.toString();
}
