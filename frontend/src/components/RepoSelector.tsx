import { useState, useEffect } from 'react';
import type { GitHubRepo, ManifestFile } from '../types/api';
import { useRepos, useManifests, useAnalyzeRepo } from '../hooks/useApi';
import { VisualizationResult } from './VisualizationResult';

interface RepoSelectorProps {
  onBack?: () => void;
}

export function RepoSelector({ onBack: _onBack }: RepoSelectorProps) {
  const { repos, isLoading: reposLoading, error: reposError, fetchRepos } = useRepos();
  const { manifests, isLoading: manifestsLoading, error: manifestsError, fetchManifests, reset: resetManifests } = useManifests();
  const { job, isLoading: analyzing, error: analyzeError, analyze, reset: resetAnalyze } = useAnalyzeRepo();
  
  const [selectedRepo, setSelectedRepo] = useState<GitHubRepo | null>(null);
  const [selectedManifest, setSelectedManifest] = useState<ManifestFile | null>(null);
  const [search, setSearch] = useState('');

  useEffect(() => {
    fetchRepos();
  }, [fetchRepos]);

  useEffect(() => {
    if (selectedRepo) {
      const [owner, repo] = selectedRepo.full_name.split('/');
      fetchManifests(owner, repo);
    }
  }, [selectedRepo, fetchManifests]);

  const filteredRepos = repos.filter(repo => 
    repo.name.toLowerCase().includes(search.toLowerCase()) ||
    repo.full_name.toLowerCase().includes(search.toLowerCase())
  );

  const handleAnalyze = () => {
    if (!selectedRepo || !selectedManifest) return;
    
    const [owner, repo] = selectedRepo.full_name.split('/');
    analyze(owner, repo, {
      manifest_path: selectedManifest.path,
      formats: ['svg', 'png', 'pdf']
    });
  };

  const handleReset = () => {
    setSelectedRepo(null);
    setSelectedManifest(null);
    resetManifests();
    resetAnalyze();
  };

  // If we have a job result, show it
  if (job && (job.status === 'completed' || job.status === 'failed')) {
    return (
      <div className="h-full flex flex-col">
        {/* Back button */}
        <div className="flex items-center gap-3 mb-4">
          <button
            onClick={handleReset}
            className="btn btn-ghost"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            Back
          </button>
          <span className="text-[var(--color-text-muted)]">•</span>
          <span className="font-mono text-sm text-[var(--color-text)]">{selectedRepo?.full_name}</span>
        </div>
        <div className="flex-1 min-h-0">
          <VisualizationResult job={job} onReset={handleReset} />
        </div>
      </div>
    );
  }

  // Processing state
  if (job && (job.status === 'pending' || job.status === 'processing')) {
    return (
      <div className="h-full flex flex-col">
        <div className="flex items-center gap-3 mb-4">
          <span className="font-mono text-sm text-[var(--color-text)]">{selectedRepo?.full_name}</span>
        </div>
        <div className="flex-1 min-h-0">
          <VisualizationResult job={job} onReset={handleReset} />
        </div>
      </div>
    );
  }

  return (
    <div className="h-full flex gap-6">
      {/* Repo list */}
      <div className="w-96 flex-shrink-0 panel flex flex-col">
        <div className="panel-header">
          Select Repository
        </div>
        
        {/* Search */}
        <div className="p-3 border-b border-[var(--color-border)]">
          <div className="relative">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search repositories..."
              className="input pl-9"
            />
            <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
        </div>

        {/* Error */}
        {reposError && (
          <div className="p-3 m-3 bg-[var(--color-error-light)] rounded-lg text-sm text-[var(--color-error)]">
            {reposError}
          </div>
        )}

        {/* Loading */}
        {reposLoading && (
          <div className="flex-1 flex items-center justify-center">
            <div className="w-8 h-8 border-2 border-[var(--color-primary)] border-t-transparent rounded-full animate-spin" />
          </div>
        )}

        {/* Repo List */}
        {!reposLoading && (
          <div className="flex-1 overflow-y-auto">
            {filteredRepos.length === 0 ? (
              <div className="p-8 text-center text-[var(--color-text-muted)] text-sm">
                {search ? 'No repositories match your search' : 'No repositories found'}
              </div>
            ) : (
              <div className="divide-y divide-[var(--color-border)]">
                {filteredRepos.map(repo => (
                  <button
                    key={repo.id}
                    onClick={() => {
                      setSelectedRepo(repo);
                      setSelectedManifest(null);
                    }}
                    className={`w-full p-3 text-left hover:bg-[var(--color-bg-hover)] transition-colors ${
                      selectedRepo?.id === repo.id ? 'bg-[var(--color-primary-light)]' : ''
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-[var(--color-text)] truncate">{repo.name}</span>
                          {repo.private && (
                            <span className="badge badge-warning text-xs">Private</span>
                          )}
                        </div>
                        {repo.description && (
                          <p className="text-xs text-[var(--color-text-muted)] truncate mt-0.5">
                            {repo.description}
                          </p>
                        )}
                        <div className="flex items-center gap-3 mt-1 text-xs text-[var(--color-text-muted)]">
                          {repo.language && (
                            <span className="flex items-center gap-1">
                              <span className="w-2 h-2 rounded-full bg-[var(--color-primary)]" />
                              {repo.language}
                            </span>
                          )}
                        </div>
                      </div>
                      <svg className="w-4 h-4 text-[var(--color-text-muted)] flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5l7 7-7 7" />
                      </svg>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Manifest selection */}
      <div className="flex-1 panel flex flex-col">
        {!selectedRepo ? (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 bg-[var(--color-bg)] rounded-xl flex items-center justify-center">
                <svg className="w-8 h-8 text-[var(--color-text-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                </svg>
              </div>
              <p className="text-[var(--color-text-secondary)]">Select a repository to analyze</p>
            </div>
          </div>
        ) : (
          <>
            <div className="panel-header flex items-center justify-between">
              <div className="flex items-center gap-2">
                <svg className="w-4 h-4 text-[var(--color-text-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                </svg>
                <span className="font-mono">{selectedRepo.full_name}</span>
              </div>
              <button
                onClick={() => {
                  setSelectedRepo(null);
                  setSelectedManifest(null);
                  resetManifests();
                }}
                className="btn btn-ghost text-xs py-1 px-2"
              >
                Change
              </button>
            </div>

            {/* Error */}
            {(manifestsError || analyzeError) && (
              <div className="p-3 m-3 bg-[var(--color-error-light)] rounded-lg text-sm text-[var(--color-error)]">
                {manifestsError || analyzeError}
              </div>
            )}

            {/* Loading manifests */}
            {manifestsLoading && (
              <div className="flex-1 flex items-center justify-center">
                <div className="text-center">
                  <div className="w-6 h-6 mx-auto mb-2 border-2 border-[var(--color-primary)] border-t-transparent rounded-full animate-spin" />
                  <p className="text-sm text-[var(--color-text-muted)]">Detecting manifest files...</p>
                </div>
              </div>
            )}

            {/* Manifest list */}
            {!manifestsLoading && manifests.length > 0 && (
              <div className="flex-1 p-4 overflow-y-auto">
                <p className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wide mb-3">
                  Select manifest to analyze
                </p>
                <div className="space-y-2">
                  {manifests.map(manifest => (
                    <button
                      key={manifest.path}
                      onClick={() => setSelectedManifest(manifest)}
                      className={`w-full p-3 text-left rounded-lg border transition-all ${
                        selectedManifest?.path === manifest.path
                          ? 'border-[var(--color-primary)] bg-[var(--color-primary-light)]'
                          : 'border-[var(--color-border)] hover:border-[var(--color-text-muted)]'
                      }`}
                    >
                      <div className="flex items-center gap-3">
                        <svg className="w-5 h-5 text-[var(--color-text-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                        </svg>
                        <div>
                          <p className="font-mono text-sm text-[var(--color-text)]">{manifest.name}</p>
                          <p className="text-xs text-[var(--color-text-muted)] capitalize">{manifest.language}</p>
                        </div>
                      </div>
                    </button>
                  ))}
                </div>

                {/* Analyze button */}
                {selectedManifest && (
                  <button
                    onClick={handleAnalyze}
                    disabled={analyzing}
                    className="btn btn-primary w-full mt-6"
                  >
                    {analyzing ? (
                      <>
                        <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                        </svg>
                        Analyzing...
                      </>
                    ) : (
                      <>
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                        </svg>
                        Generate Tower
                      </>
                    )}
                  </button>
                )}
              </div>
            )}

            {/* No manifests found */}
            {!manifestsLoading && manifests.length === 0 && (
              <div className="flex-1 flex items-center justify-center">
                <div className="text-center p-8">
                  <div className="w-16 h-16 mx-auto mb-4 bg-[var(--color-bg)] rounded-xl flex items-center justify-center">
                    <svg className="w-8 h-8 text-[var(--color-text-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                  </div>
                  <p className="font-medium text-[var(--color-text)]">No manifest files found</p>
                  <p className="text-sm text-[var(--color-text-muted)] mt-1">
                    This repository doesn't have any supported dependency files in the root directory.
                  </p>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
