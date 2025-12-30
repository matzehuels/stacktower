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
        compact && 'px-0',
        className
      )}
    >
      {isLoggingIn ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <Github className="h-4 w-4" />
      )}
      {!compact && (
        <span>{isLoggingIn ? 'Redirecting...' : 'Sign in with GitHub'}</span>
      )}
    </Button>
  );
}
