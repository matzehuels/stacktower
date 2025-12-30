/**
 * Repository selector for analyzing GitHub repos.
 */

import { useState, useEffect } from 'react';
import { Search, ChevronRight, ChevronLeft, Folder, File, Zap } from 'lucide-react';
import type { GitHubRepo, ManifestFile, JobResponse } from '@/types/api';
import { useRepos, useManifests, useAnalyzeRepoMutation } from '@/hooks/queries';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

interface Props {
  onJobCreated: (job: JobResponse) => void;
}

export function RepoSelector({ onJobCreated }: Props) {
  const { data: repos = [], isLoading: reposLoading, error: reposError } = useRepos();
  const [selectedRepo, setSelectedRepo] = useState<GitHubRepo | null>(null);
  const [selectedManifest, setSelectedManifest] = useState<ManifestFile | null>(null);
  const [search, setSearch] = useState('');

  // Parse owner/repo from selectedRepo
  const [owner, repoName] = selectedRepo?.full_name.split('/') ?? ['', ''];
  
  const { data: manifests = [], isLoading: manifestsLoading, error: manifestsError } = useManifests(owner, repoName);
  const { job, isLoading: analyzing, error: analyzeError, analyze, reset: resetAnalyze } = useAnalyzeRepoMutation();

  // When job is created, pass it up
  useEffect(() => {
    if (job) {
      onJobCreated(job);
    }
  }, [job, onJobCreated]);

  const filteredRepos = repos.filter(repo =>
    repo.name.toLowerCase().includes(search.toLowerCase()) ||
    repo.full_name.toLowerCase().includes(search.toLowerCase())
  );

  const handleAnalyze = () => {
    if (!selectedRepo || !selectedManifest) return;
    const [o, r] = selectedRepo.full_name.split('/');
    analyze(o, r, {
      manifest_path: selectedManifest.path,
      formats: ['svg', 'png', 'pdf']
    });
  };

  const handleBack = () => {
    setSelectedRepo(null);
    setSelectedManifest(null);
    resetAnalyze();
  };

  return (
    <div className="flex-1 flex h-full min-h-0">
      {/* Repo list */}
      <div className="w-80 border-r border-border bg-card flex flex-col min-h-0">
        <div className="p-4 border-b border-border">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search repositories..."
              className="pl-9"
            />
          </div>
        </div>

        {reposError && (
          <div className="p-4 text-sm text-destructive">{reposError.message}</div>
        )}

        {reposLoading ? (
          <div className="p-4 space-y-3">
            {[...Array(5)].map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        ) : (
          <div className="flex-1 overflow-y-auto">
            {filteredRepos.length === 0 ? (
              <div className="p-8 text-center text-muted-foreground text-sm">
                {search ? 'No repositories match your search' : 'No repositories found'}
              </div>
            ) : (
              <div className="divide-y divide-border">
                {filteredRepos.map(repo => (
                  <button
                    key={repo.id}
                    onClick={() => {
                      setSelectedRepo(repo);
                      setSelectedManifest(null);
                    }}
                    className={cn(
                      'w-full p-4 text-left hover:bg-muted transition-colors',
                      selectedRepo?.id === repo.id && 'bg-primary/10'
                    )}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-foreground truncate">{repo.name}</span>
                          {repo.private && (
                            <span className="px-1.5 py-0.5 text-[10px] bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 rounded">Private</span>
                          )}
                        </div>
                        {repo.description && (
                          <p className="text-xs text-muted-foreground truncate mt-0.5">{repo.description}</p>
                        )}
                        {repo.language && (
                          <div className="flex items-center gap-1 mt-1 text-xs text-muted-foreground">
                            <span className="w-2 h-2 rounded-full bg-primary" />
                            {repo.language}
                          </div>
                        )}
                      </div>
                      <ChevronRight className="w-4 h-4 text-muted-foreground" />
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Manifest selection */}
      <div className="flex-1 flex flex-col bg-background">
        {!selectedRepo ? (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 bg-muted rounded-xl flex items-center justify-center">
                <Folder className="w-8 h-8 text-muted-foreground" />
              </div>
              <p className="text-foreground font-medium">Select a repository</p>
              <p className="text-sm text-muted-foreground mt-1">Choose from your GitHub repos on the left</p>
            </div>
          </div>
        ) : (
          <>
            {/* Repo header */}
            <div className="h-14 px-6 flex items-center justify-between border-b border-border bg-card">
              <div className="flex items-center gap-3">
                <Button variant="ghost" size="icon" onClick={handleBack}>
                  <ChevronLeft className="w-5 h-5" />
                </Button>
                <span className="font-mono text-foreground">{selectedRepo.full_name}</span>
              </div>
            </div>

            {/* Error */}
            {(manifestsError || analyzeError) && (
              <div className="p-4 m-4 bg-destructive/10 border border-destructive/20 rounded-lg text-sm text-destructive">
                {manifestsError?.message || analyzeError}
              </div>
            )}

            {/* Loading manifests */}
            {manifestsLoading ? (
              <div className="flex-1 flex items-center justify-center">
                <div className="text-center">
                  <div className="w-6 h-6 mx-auto mb-3 border-2 border-primary rounded-full border-t-transparent animate-spin" />
                  <p className="text-sm text-muted-foreground">Detecting manifest files...</p>
                </div>
              </div>
            ) : manifests.length > 0 ? (
              <div className="flex-1 p-6 overflow-y-auto">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-4">
                  Select manifest to analyze
                </p>
                <div className="space-y-2">
                  {manifests.map(manifest => (
                    <Card
                      key={manifest.path}
                      className={cn(
                        'p-4 cursor-pointer transition-all',
                        selectedManifest?.path === manifest.path
                          ? 'border-primary bg-primary/10'
                          : 'hover:border-muted-foreground'
                      )}
                      onClick={() => setSelectedManifest(manifest)}
                    >
                      <div className="flex items-center gap-3">
                        <File className="w-5 h-5 text-muted-foreground" />
                        <div>
                          <p className="font-mono text-sm text-foreground">{manifest.name}</p>
                          <p className="text-xs text-muted-foreground capitalize">{manifest.language}</p>
                        </div>
                      </div>
                    </Card>
                  ))}
                </div>

                {selectedManifest && (
                  <Button
                    onClick={handleAnalyze}
                    disabled={analyzing}
                    className="w-full mt-6"
                    size="lg"
                  >
                    {analyzing ? (
                      <>
                        <span className="w-4 h-4 border-2 border-current rounded-full border-t-transparent animate-spin mr-2" />
                        Analyzing...
                      </>
                    ) : (
                      <>
                        <Zap className="w-4 h-4 mr-2" />
                        Generate Tower
                      </>
                    )}
                  </Button>
                )}
              </div>
            ) : (
              <div className="flex-1 flex items-center justify-center">
                <div className="text-center p-8">
                  <div className="w-16 h-16 mx-auto mb-4 bg-muted rounded-xl flex items-center justify-center">
                    <File className="w-8 h-8 text-muted-foreground" />
                  </div>
                  <p className="font-medium text-foreground">No manifest files found</p>
                  <p className="text-sm text-muted-foreground mt-1">
                    This repository doesn't have supported dependency files
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
