/**
 * Main application component.
 * 
 * Supports both authenticated and public (explore-only) modes.
 */

import { useState, useEffect, useCallback } from 'react';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Sidebar, PackagesView } from '@/components/layout';
import type { Tab } from '@/components/layout';
import { VisualizationResult } from '@/components/VisualizationResult';
import { RepoSelector } from '@/components/RepoSelector';
import { Library } from '@/components/Library';
import { TowerExplorer } from '@/components/TowerExplorer';
import { LandingPage } from '@/components/LandingPage';
import { useCurrentUser, useLogout, useLogin, useRenderMutation } from '@/hooks/queries';
import type { JobResponse, VizType } from '@/types/api';

function App() {
  const { data: user, isLoading: authLoading, refetch: checkAuth } = useCurrentUser();
  const { mutate: logoutMutate } = useLogout();
  const { login } = useLogin();
  const { job: renderJob, isLoading, error, render, reset: resetRender } = useRenderMutation();
  
  // Start on explore for unauthenticated users, packages for authenticated
  const [activeTab, setActiveTab] = useState<Tab>('explore');
  const [selectedJob, setSelectedJob] = useState<JobResponse | null>(null);
  const [selectedInLibrary, setSelectedInLibrary] = useState<boolean | undefined>(undefined);

  // Set initial tab based on auth state
  useEffect(() => {
    if (user && activeTab === 'explore') {
      // If user just logged in, stay on explore (they were browsing)
      // Only switch to packages if they haven't navigated yet
    }
  }, [user, activeTab]);

  const displayJob = renderJob || selectedJob;

  // Check for auth callback and shared render
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    
    // Handle auth callback
    if (params.get('auth') === 'success') {
      checkAuth();
      window.history.replaceState({}, '', window.location.pathname);
    }
    
    // Handle shared render URL (e.g., ?render=abc123)
    const sharedRenderId = params.get('render');
    if (sharedRenderId && !selectedJob && !renderJob) {
      // Load the shared render
      import('@/lib/api').then(({ getRender }) => {
        getRender(sharedRenderId)
          .then((job) => {
            setSelectedJob(job);
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
      });
    }
  }, [checkAuth, selectedJob, renderJob]);

  const handleReset = useCallback(() => {
    resetRender();
    setSelectedJob(null);
  }, [resetRender]);

  const handleLogout = useCallback(() => {
    logoutMutate();
    setActiveTab('explore'); // Go back to explore after logout
  }, [logoutMutate]);

  const handleRepoJob = useCallback((repoJob: JobResponse) => {
    setSelectedJob(repoJob);
  }, []);

  const handleSelect = useCallback((job: JobResponse, inLibrary?: boolean) => {
    setSelectedJob(job);
    setSelectedInLibrary(inLibrary);
  }, []);

  const handleNavigate = useCallback((tab: Tab) => {
    // Check if user needs to be authenticated for this tab
    if (!user && tab !== 'explore') {
      toast.info('Sign in to access this feature', {
        action: {
          label: 'Sign in with GitHub',
          onClick: login,
        },
      });
      return;
    }
    setActiveTab(tab);
    setSelectedJob(null);
    resetRender();
  }, [user, login, resetRender]);

  const handleJobUpdate = useCallback((updatedJob: JobResponse) => {
    if (!renderJob) {
      setSelectedJob(updatedJob);
    }
  }, [renderJob]);

  const handleVizTypeChange = useCallback((newVizType: VizType) => {
    const source = displayJob?.result?.source;
    if (!source?.language || !source?.package) {
      toast.error('Cannot switch visualization type', { 
        description: 'Source information not available' 
      });
      return;
    }
    
    toast.info(`Switching to ${newVizType === 'tower' ? 'Tower' : 'Node Link'}...`);
    
    render({
      language: source.language as 'python' | 'javascript' | 'go' | 'rust',
      package: source.package,
      viz_type: newVizType,
      formats: ['svg', 'png', 'pdf', 'json'],
    });
  }, [displayJob, render]);

  // Loading state
  if (authLoading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="w-10 h-10 animate-spin text-primary" />
          <p className="text-muted-foreground text-sm">Loading...</p>
        </div>
      </div>
    );
  }

  // Show visualization result if we have a job (for both authenticated and unauthenticated users)
  if (displayJob && (displayJob.status === 'completed' || displayJob.status === 'pending' || displayJob.status === 'processing')) {
    // For unauthenticated users viewing a tower, show minimal UI
    if (!user) {
      return (
        <div className="h-screen bg-background flex overflow-hidden">
          <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
            <VisualizationResult 
              job={displayJob} 
              onReset={handleReset} 
              onJobUpdate={handleJobUpdate} 
              onDelete={handleReset}
              inLibrary={false}
              isAuthenticated={false}
            />
          </main>
        </div>
      );
    }
    
    return (
      <div className="h-screen bg-background flex overflow-hidden">
        <Sidebar 
          user={user} 
          logout={handleLogout} 
          login={login}
          activeTab={activeTab} 
          onNavigate={handleNavigate} 
        />
        <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
          <VisualizationResult 
            job={displayJob} 
            onReset={handleReset} 
            onJobUpdate={handleJobUpdate} 
            onDelete={handleReset}
            onVizTypeChange={handleVizTypeChange}
            inLibrary={selectedInLibrary}
            isAuthenticated={true}
          />
        </main>
      </div>
    );
  }

  // Landing page for unauthenticated users
  if (!user) {
    return (
      <LandingPage 
        onSelect={(job) => setSelectedJob(job)} 
        onLogin={login} 
      />
    );
  }

  // Main app for authenticated users
  return (
    <div className="h-screen bg-background flex overflow-hidden">
      <Sidebar 
        user={user} 
        logout={handleLogout} 
        login={login}
        activeTab={activeTab} 
        onNavigate={handleNavigate} 
      />

      <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {activeTab === 'packages' && (
          <PackagesView onSubmit={render} isLoading={isLoading} error={error} />
        )}
        {activeTab === 'repos' && (
          <RepoSelector onJobCreated={handleRepoJob} />
        )}
        {activeTab === 'library' && (
          <Library onSelect={handleSelect} />
        )}
        {activeTab === 'explore' && (
          <TowerExplorer 
            onSelect={handleSelect} 
            onRender={render}
            isAuthenticated={true} 
            onLogin={login} 
          />
        )}
      </main>
    </div>
  );
}

export default App;
