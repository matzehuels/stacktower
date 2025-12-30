/**
 * Landing page shown to unauthenticated users.
 * Clean, minimal design with example tower and stats.
 */

import { Github, Layers, Package, Users } from 'lucide-react';
import { StacktowerLogoLarge } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { usePublicStats } from '@/hooks/queries';

// Import the showcase image as an asset
import fastApiTower from '@/assets/fastapi.svg';

interface LoginScreenProps {
  onLogin: () => void;
}

function formatNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toString();
}

export function LoginScreen({ onLogin }: LoginScreenProps) {
  const { data: stats } = usePublicStats();

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-950 dark:to-slate-900 flex flex-col">
      {/* Header */}
      <header className="flex items-center justify-between px-6 py-4">
        <div className="flex items-center gap-3">
          <StacktowerLogoLarge />
          <span className="text-xl font-semibold text-foreground">Stacktower</span>
        </div>
        <Button onClick={onLogin} variant="outline" size="sm">
          <Github className="mr-2 h-4 w-4" />
          Sign in
        </Button>
      </header>

      {/* Main content */}
      <main className="flex-1 flex flex-col lg:flex-row items-center justify-center gap-12 px-6 py-12">
        {/* Left: Hero text */}
        <div className="max-w-md text-center lg:text-left">
          <h1 className="text-4xl lg:text-5xl font-bold text-foreground mb-4 tracking-tight">
            Visualize your dependencies
          </h1>
          <p className="text-lg text-muted-foreground mb-8">
            See the full picture of your package dependencies. 
            Beautiful tower visualizations for Python, JavaScript, Rust, and Go.
          </p>
          
          <Button onClick={onLogin} size="lg" className="mb-8">
            <Github className="mr-2 h-5 w-5" />
            Get started with GitHub
          </Button>

          {/* Stats */}
          {stats && (stats.total_renders > 0 || stats.total_dependencies > 0) && (
            <div className="flex items-center justify-center lg:justify-start gap-8 text-sm text-muted-foreground">
              <div className="flex items-center gap-2">
                <Layers className="h-4 w-4" />
                <span><strong className="text-foreground">{formatNumber(stats.total_renders)}</strong> repos processed</span>
              </div>
              <div className="flex items-center gap-2">
                <Package className="h-4 w-4" />
                <span><strong className="text-foreground">{formatNumber(stats.total_dependencies)}</strong> deps scanned</span>
              </div>
              {stats.total_users > 0 && (
                <div className="flex items-center gap-2">
                  <Users className="h-4 w-4" />
                  <span><strong className="text-foreground">{formatNumber(stats.total_users)}</strong> users</span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Right: Example tower */}
        <div className="relative">
          <div className="w-80 h-96 lg:w-96 lg:h-[480px] rounded-xl border border-border/50 bg-white dark:bg-slate-900 shadow-xl overflow-hidden">
            <img 
              src={fastApiTower} 
              alt="FastAPI dependency tower"
              className="w-full h-full object-contain p-4"
            />
          </div>
          <div className="absolute -bottom-3 left-1/2 -translate-x-1/2 bg-background border border-border rounded-full px-4 py-1.5 shadow-lg">
            <span className="text-sm font-medium text-muted-foreground">fastapi</span>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="px-6 py-4 text-center text-sm text-muted-foreground">
        <p>Open source · Built for developers</p>
      </footer>
    </div>
  );
}
