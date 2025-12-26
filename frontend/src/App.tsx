import { useState, useEffect } from 'react';
import { PlaygroundForm } from './components/PlaygroundForm';
import { VisualizationResult } from './components/VisualizationResult';
import { RepoSelector } from './components/RepoSelector';
import { ReportHistory } from './components/ReportHistory';
import { AgentSidebar } from './components/AgentSidebar';
import { AgentProvider, useAgent } from './context/AgentContext';
import { useVisualize, useAuth } from './hooks/useApi';

type Mode = 'package' | 'github' | 'history';

function AppContent() {
  const { job, isLoading, error, submit, reset } = useVisualize();
  const { user, isLoading: authLoading, login, logout, checkAuth } = useAuth();
  const [mode, setMode] = useState<Mode>('package');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const { isVisible: agentVisible, toggleVisible: toggleAgentPanel, events: agentEvents, isConnected } = useAgent();

  // Check for auth callback
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get('auth') === 'success') {
      setMode('github');
      checkAuth();
      window.history.replaceState({}, '', window.location.pathname);
    }
  }, [checkAuth]);

  return (
    <div className="flex h-screen bg-[var(--color-bg)]">
      {/* Left Sidebar */}
      <aside className={`${sidebarCollapsed ? 'w-14' : 'w-56'} bg-[var(--color-sidebar)] flex flex-col border-r border-[var(--color-border)] transition-all duration-200`}>
        {/* Logo */}
        <div className="h-12 flex items-center px-3 border-b border-[var(--color-border)]">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 bg-[var(--color-primary)] rounded flex items-center justify-center">
              <svg className="w-4 h-4 text-[#0f0f0f]" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                <rect x="3" y="14" width="6" height="7" rx="1" />
                <rect x="9" y="8" width="6" height="13" rx="1" />
                <rect x="15" y="3" width="6" height="18" rx="1" />
              </svg>
            </div>
            {!sidebarCollapsed && (
              <span className="text-[var(--color-text)] font-semibold text-sm">Stacktower</span>
            )}
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 py-2 overflow-y-auto">
          <div className="px-2 space-y-0.5">
            <NavItem
              active={mode === 'package'}
              onClick={() => setMode('package')}
              collapsed={sidebarCollapsed}
              icon={<PackageIcon />}
              label="Packages"
            />
            <NavItem
              active={mode === 'github'}
              onClick={() => setMode('github')}
              collapsed={sidebarCollapsed}
              icon={<GitHubIcon />}
              label="GitHub"
            />
            <NavItem
              active={mode === 'history'}
              onClick={() => setMode('history')}
              collapsed={sidebarCollapsed}
              icon={<HistoryIcon />}
              label="History"
            />
          </div>

          {/* Separator */}
          {!sidebarCollapsed && (
            <>
              <div className="my-3 mx-3 border-t border-[var(--color-border)]" />
              
              {/* Quick actions */}
              <div className="px-2">
                <p className="px-2.5 mb-1.5 text-[10px] font-medium text-[var(--color-text-muted)] uppercase tracking-wider">
                  Quick Start
                </p>
                <div className="space-y-0.5">
                  {[
                    { lang: 'python', pkg: 'flask', emoji: '🐍' },
                    { lang: 'javascript', pkg: 'express', emoji: '📦' },
                    { lang: 'rust', pkg: 'tokio', emoji: '🦀' },
                  ].map(({ lang, pkg, emoji }) => (
                    <button
                      key={pkg}
                      onClick={() => {
                        setMode('package');
                        submit({ language: lang as any, package: pkg, formats: ['svg', 'png', 'pdf'] });
                      }}
                      disabled={isLoading}
                      className="w-full flex items-center gap-2.5 px-2.5 py-1.5 rounded text-xs text-[var(--color-text-muted)] 
                                 hover:bg-[var(--color-sidebar-hover)] hover:text-[var(--color-text-secondary)]
                                 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                    >
                      <span>{emoji}</span>
                      <span className="font-mono">{pkg}</span>
                    </button>
                  ))}
                </div>
              </div>
            </>
          )}
        </nav>

        {/* User section */}
        <div className="border-t border-[var(--color-border)] p-2">
          {user ? (
            <div className={`flex items-center ${sidebarCollapsed ? 'justify-center' : 'gap-2.5 px-2'}`}>
              <img
                src={user.avatar_url}
                alt={user.login}
                className="avatar-sm"
              />
              {!sidebarCollapsed && (
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-medium text-[var(--color-text)] truncate">
                    {user.login}
                  </p>
                  <button
                    onClick={logout}
                    className="text-[10px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
                  >
                    Sign out
                  </button>
                </div>
              )}
            </div>
          ) : (
            <button
              onClick={login}
              disabled={authLoading}
              className={`w-full flex items-center gap-2 px-2.5 py-1.5 rounded text-xs font-medium 
                         bg-[var(--color-sidebar-hover)] text-[var(--color-text-secondary)]
                         hover:bg-[var(--color-sidebar-active)] transition-colors
                         ${sidebarCollapsed ? 'justify-center' : ''}`}
            >
              <GitHubIcon />
              {!sidebarCollapsed && <span>Sign in</span>}
            </button>
          )}
        </div>

        {/* Collapse toggle */}
        <button
          onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
          className="h-9 flex items-center justify-center border-t border-[var(--color-border)] text-[var(--color-text-muted)] 
                     hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-sidebar-hover)] transition-colors"
        >
          <svg className={`w-4 h-4 transition-transform ${sidebarCollapsed ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
          </svg>
        </button>
      </aside>

      {/* Main content */}
      <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Top bar */}
        <header className="h-12 bg-[var(--color-bg-elevated)] border-b border-[var(--color-border)] flex items-center justify-between px-4">
          <div className="flex items-center gap-3">
            <h1 className="text-sm font-medium text-[var(--color-text)]">
              {mode === 'package' ? 'Package Analysis' : mode === 'github' ? 'GitHub Repositories' : 'Report History'}
            </h1>
            {job && job.status === 'completed' && (
              <span className="badge badge-success">
                <span className="status-dot status-dot-success" />
                Complete
              </span>
            )}
            {isLoading && (
              <span className="badge badge-neutral">
                <svg className="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Processing
              </span>
            )}
          </div>
          
          <div className="flex items-center gap-2">
            {/* Agent panel toggle */}
            <button
              onClick={toggleAgentPanel}
              className={`btn ${agentVisible ? 'btn-primary' : 'btn-ghost'} relative`}
              title="Toggle AI Agent Panel"
            >
              <BrainIcon />
              <span className="hidden sm:inline">Agent</span>
              {(agentEvents.length > 0 || isConnected) && (
                <span className={`absolute -top-1 -right-1 w-4 h-4 text-[10px] font-medium rounded-full flex items-center justify-center ${
                  isConnected ? 'bg-[var(--color-success)] text-[#0f0f0f]' : 'bg-[var(--color-primary)] text-[#0f0f0f]'
                }`}>
                  {isConnected ? '•' : agentEvents.length > 9 ? '9+' : agentEvents.length}
                </span>
              )}
            </button>
            
            <a
              href="https://github.com/matzehuels/stacktower"
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-ghost"
            >
              <GitHubIcon />
              <span className="hidden sm:inline">Source</span>
            </a>
          </div>
        </header>

        {/* Content area */}
        <div className="flex-1 overflow-auto p-4">
          {/* Package Mode */}
          {mode === 'package' && (
            <div className="h-full flex flex-col">
              {!job && (
                <div className="panel mb-4 animate-fade-in">
                  <div className="panel-header">
                    <span>Analyze Package</span>
                  </div>
                  <div className="panel-body">
                    <PlaygroundForm onSubmit={submit} isLoading={isLoading} />
                    {error && (
                      <div className="mt-3 p-3 bg-[var(--color-error-light)] border border-[var(--color-error)]/20 rounded text-xs text-[var(--color-error)]">
                        {error}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {job && (
                <div className="flex-1 min-h-0 animate-fade-in">
                  <VisualizationResult job={job} onReset={reset} />
                </div>
              )}

              {!job && !isLoading && (
                <div className="flex-1 flex items-center justify-center">
                  <div className="empty-state">
                    <div className="empty-state-icon">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                        <rect x="3" y="14" width="6" height="7" rx="1" />
                        <rect x="9" y="8" width="6" height="13" rx="1" />
                        <rect x="15" y="3" width="6" height="18" rx="1" />
                      </svg>
                    </div>
                    <div className="empty-state-title">No analysis yet</div>
                    <div className="empty-state-text">
                      Enter a package name above to visualize dependencies
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* GitHub Mode */}
          {mode === 'github' && (
            <div className="h-full">
              {!user ? (
                <div className="h-full flex items-center justify-center">
                  <div className="empty-state">
                    <div className="empty-state-icon">
                      <GitHubIcon />
                    </div>
                    <div className="empty-state-title">Connect GitHub</div>
                    <div className="empty-state-text mb-4">
                      Analyze dependencies from your repositories
                    </div>
                    <button
                      onClick={login}
                      disabled={authLoading}
                      className="btn btn-primary"
                    >
                      <GitHubIcon />
                      Sign in with GitHub
                    </button>
                  </div>
                </div>
              ) : (
                <RepoSelector onBack={() => setMode('package')} />
              )}
            </div>
          )}

          {/* History Mode */}
          {mode === 'history' && (
            <div className="h-full">
              <ReportHistory />
            </div>
          )}
        </div>
      </main>

      {/* Right Agent Sidebar */}
      <AgentSidebar />
    </div>
  );
}

// Navigation item component
function NavItem({ 
  active, 
  onClick, 
  collapsed, 
  icon, 
  label 
}: { 
  active: boolean; 
  onClick: () => void; 
  collapsed: boolean; 
  icon: React.ReactNode; 
  label: string;
}) {
  return (
    <button
      onClick={onClick}
      className={`w-full flex items-center gap-2.5 px-2.5 py-2 rounded text-xs font-medium transition-colors ${
        active
          ? 'bg-[var(--color-sidebar-active)] text-[var(--color-text)]'
          : 'text-[var(--color-text-muted)] hover:bg-[var(--color-sidebar-hover)] hover:text-[var(--color-text-secondary)]'
      } ${collapsed ? 'justify-center' : ''}`}
    >
      <span className="w-4 h-4 flex-shrink-0">{icon}</span>
      {!collapsed && <span>{label}</span>}
    </button>
  );
}

// Icons
function PackageIcon() {
  return (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="1.5">
      <path strokeLinecap="round" strokeLinejoin="round" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
    </svg>
  );
}

function GitHubIcon() {
  return (
    <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
    </svg>
  );
}

function HistoryIcon() {
  return (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="1.5">
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function BrainIcon() {
  return (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth="1.5">
      <path strokeLinecap="round" strokeLinejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
    </svg>
  );
}

function App() {
  return (
    <AgentProvider>
      <AppContent />
    </AgentProvider>
  );
}

export default App;
