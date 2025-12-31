/**
 * Error boundary component for graceful error handling.
 * 
 * Catches JavaScript errors anywhere in the child component tree,
 * logs them, and displays a fallback UI instead of crashing.
 * 
 * Usage:
 *   <ErrorBoundary>
 *     <App />
 *   </ErrorBoundary>
 */

import { Component, type ReactNode, type ErrorInfo } from 'react';
import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface Props {
  children: ReactNode;
  /** Custom fallback UI. If not provided, uses default error UI. */
  fallback?: ReactNode;
  /** Callback when an error is caught */
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // Log to console in development
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    
    // Call optional error callback
    this.props.onError?.(error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return <DefaultErrorFallback 
        error={null}
        onReset={this.handleReset}
        onReload={this.handleReload}
      />;
    }

    return this.props.children;
  }
}

// =============================================================================
// Default Error Fallback UI
// =============================================================================

interface DefaultErrorFallbackProps {
  error: Error | null;
  onReset: () => void;
  onReload: () => void;
}

// Collapsed/fallen tower SVG
function CollapsedTowerSVG({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 200 120"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      {/* Ground line */}
      <path
        d="M10 100 L190 100"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        opacity="0.3"
      />
      
      {/* Dust clouds */}
      <ellipse cx="60" cy="95" rx="15" ry="8" fill="currentColor" opacity="0.1" />
      <ellipse cx="100" cy="92" rx="20" ry="10" fill="currentColor" opacity="0.08" />
      <ellipse cx="145" cy="96" rx="12" ry="6" fill="currentColor" opacity="0.1" />
      
      {/* Fallen tower blocks - rotated and scattered */}
      <g transform="rotate(-75 45 85)">
        <rect x="30" y="70" width="30" height="20" rx="3" fill="currentColor" opacity="0.6" />
        <rect x="32" y="72" width="26" height="16" rx="2" fill="currentColor" opacity="0.2" />
      </g>
      
      <g transform="rotate(-60 80 80)">
        <rect x="65" y="65" width="28" height="18" rx="3" fill="currentColor" opacity="0.5" />
        <rect x="67" y="67" width="24" height="14" rx="2" fill="currentColor" opacity="0.15" />
      </g>
      
      <g transform="rotate(-85 115 90)">
        <rect x="100" y="75" width="26" height="16" rx="3" fill="currentColor" opacity="0.55" />
        <rect x="102" y="77" width="22" height="12" rx="2" fill="currentColor" opacity="0.18" />
      </g>
      
      <g transform="rotate(-45 150 75)">
        <rect x="138" y="62" width="24" height="14" rx="3" fill="currentColor" opacity="0.45" />
        <rect x="140" y="64" width="20" height="10" rx="2" fill="currentColor" opacity="0.12" />
      </g>
      
      <g transform="rotate(15 170 88)">
        <rect x="162" y="80" width="18" height="12" rx="2" fill="currentColor" opacity="0.4" />
        <rect x="164" y="82" width="14" height="8" rx="1" fill="currentColor" opacity="0.1" />
      </g>
      
      {/* Impact lines */}
      <path d="M55 98 L50 90" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.2" />
      <path d="M58 97 L56 92" stroke="currentColor" strokeWidth="1" strokeLinecap="round" opacity="0.15" />
      <path d="M95 96 L92 88" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" opacity="0.2" />
      <path d="M130 97 L128 91" stroke="currentColor" strokeWidth="1" strokeLinecap="round" opacity="0.15" />
    </svg>
  );
}

function DefaultErrorFallback({ onReload }: DefaultErrorFallbackProps) {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6">
      <div className="max-w-md w-full text-center">
        {/* Collapsed tower illustration */}
        <CollapsedTowerSVG className="w-48 h-32 mx-auto mb-8 text-muted-foreground" />
        
        {/* Friendly error message */}
        <h1 className="text-2xl font-semibold mb-2">Oops! Something broke</h1>
        <p className="text-sm text-muted-foreground mb-8">
          Don't worry, it happens to the best of us. Let's get you back on track.
        </p>
        
        {/* Action */}
        <Button onClick={onReload} size="lg" className="gap-2">
          <RefreshCw className="h-4 w-4" />
          Refresh page
        </Button>
        
        {/* Footer hint */}
        <p className="text-xs text-muted-foreground mt-6">
          If this keeps happening, try clearing your browser cache
        </p>
      </div>
    </div>
  );
}

// =============================================================================
// Functional Error Fallback (for use with hooks)
// =============================================================================

interface ErrorFallbackProps {
  error: Error;
  resetErrorBoundary: () => void;
}

/**
 * A simpler error fallback component that can be used with react-error-boundary
 * or similar libraries that provide error boundary hooks.
 */
export function ErrorFallback({ resetErrorBoundary }: ErrorFallbackProps) {
  return (
    <DefaultErrorFallback 
      error={null}
      onReset={resetErrorBoundary} 
      onReload={() => window.location.reload()} 
    />
  );
}
