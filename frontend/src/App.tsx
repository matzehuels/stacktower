/**
 * Main application component.
 * 
 * Supports both authenticated and public (explore-only) modes.
 */

import { useCallback } from 'react';
import { Routes, Route } from 'react-router-dom';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Sidebar, PackagesView } from '@/components/layout';
import { VisualizationResult } from '@/components/VisualizationResult';
import { RepoSelector } from '@/components/RepoSelector';
import { Library } from '@/components/Library';
import { TowerExplorer } from '@/components/TowerExplorer';
import { LandingPage } from '@/components/LandingPage';
import { NotFound } from '@/components/NotFound';
import { useCurrentUser, useLogout, useLogin, useRenderMutation } from '@/hooks/queries';
import { useAppState } from '@/hooks/useAppState';
import type { JobResponse, VizType } from '@/types/api';

function App() {
  const { data: user, isLoading: authLoading, refetch: checkAuth } = useCurrentUser();
  const { mutate: logoutMutate } = useLogout();
  const { login } = useLogin();
  const { job: renderJob, isLoading, render, reset: resetRender } = useRenderMutation();
  
  // Centralized app state management
  const { state, actions } = useAppState({ user, checkAuth });
  const { activeTab, selectedJob, selectedInLibrary } = state;

  const displayJob = renderJob || selectedJob;

  const handleReset = useCallback(() => {
    resetRender();
    actions.clearSelection();
  }, [resetRender, actions]);

  const handleLogout = useCallback(() => {
    logoutMutate();
    actions.navigate('explore');
  }, [logoutMutate, actions]);

  const handleNavigate = useCallback((tab: typeof activeTab) => {
    actions.navigate(tab);
    resetRender();
  }, [actions, resetRender]);

  const handleJobUpdate = useCallback((updatedJob: JobResponse) => {
    if (!renderJob) {
      actions.updateJob(updatedJob);
    }
  }, [renderJob, actions]);

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
      <Routes>
        <Route path="/" element={
          <LandingPage 
            onSelect={actions.selectJob} 
            onLogin={login} 
          />
        } />
        <Route path="*" element={<NotFound />} />
      </Routes>
    );
  }

  // Main app for authenticated users
  return (
    <Routes>
      <Route path="/" element={
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
              <PackagesView onSubmit={render} isLoading={isLoading} />
            )}
            {activeTab === 'repos' && (
              <RepoSelector onJobCreated={actions.selectJob} />
            )}
            {activeTab === 'library' && (
              <Library onSelect={actions.selectJob} />
            )}
            {activeTab === 'explore' && (
              <TowerExplorer 
                onSelect={actions.selectJob} 
                onRender={render}
                isAuthenticated={true} 
                onLogin={login} 
              />
            )}
          </main>
        </div>
      } />
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}

export default App;
