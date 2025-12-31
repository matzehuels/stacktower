/**
 * Shared GitHub login button with loading state.
 */

import { useState, useCallback } from 'react';
import { Github, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface GitHubLoginButtonProps {
  login: () => void;
  /** Compact mode (icon only, for collapsed sidebar) */
  compact?: boolean;
  /** Size variant */
  size?: 'sm' | 'default';
  className?: string;
}

export function GitHubLoginButton({ 
  login, 
  compact = false, 
  size = 'default',
  className 
}: GitHubLoginButtonProps) {
  const [isLoggingIn, setIsLoggingIn] = useState(false);

  const handleLogin = useCallback(() => {
    setIsLoggingIn(true);
    // Small delay to show the loading state before redirect
    setTimeout(() => {
      login();
    }, 150);
  }, [login]);

  return (
    <Button
      onClick={handleLogin}
      disabled={isLoggingIn}
      size={size}
      className={cn(
        'relative transition-all duration-200',
        'hover:scale-[1.02] active:scale-[0.98]',
        'hover:shadow-md active:shadow-sm',
        compact && 'px-0',
        className
      )}
    >
      <span className={cn(
        'flex items-center gap-2 transition-opacity duration-200',
        isLoggingIn && 'opacity-0'
      )}>
        <Github className="h-4 w-4" />
        {!compact && <span>Sign in with GitHub</span>}
      </span>
      
      {isLoggingIn && (
        <span className="absolute inset-0 flex items-center justify-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          {!compact && <span>Redirecting...</span>}
        </span>
      )}
    </Button>
  );
}
