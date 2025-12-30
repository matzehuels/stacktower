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
import { AlertTriangle, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';

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
        error={this.state.error}
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

function DefaultErrorFallback({ error, onReset, onReload }: DefaultErrorFallbackProps) {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6">
      <Card className="max-w-md w-full">
        <CardHeader className="text-center">
        {/* Error icon */}
          <div className="w-16 h-16 mx-auto mb-4 bg-destructive/10 rounded-full flex items-center justify-center">
            <AlertTriangle className="w-8 h-8 text-destructive" />
        </div>
          <CardTitle>Something went wrong</CardTitle>
          <CardDescription>
          An unexpected error occurred. Please try again or refresh the page.
          </CardDescription>
        </CardHeader>

        <CardContent>
        {/* Error details (collapsed by default) */}
        {error && (
            <details className="text-left">
              <summary className="text-sm text-muted-foreground cursor-pointer hover:text-foreground transition-colors">
              Error details
            </summary>
              <pre className="mt-2 p-3 bg-muted rounded-lg text-xs text-destructive font-mono overflow-auto max-h-32">
              {error.message}
              {error.stack && '\n\n' + error.stack}
            </pre>
          </details>
        )}
        </CardContent>

        <CardFooter className="flex gap-3 justify-center">
          <Button variant="outline" onClick={onReset}>
            Try Again
          </Button>
          <Button onClick={onReload}>
            <RefreshCw className="w-4 h-4 mr-2" />
            Refresh Page
          </Button>
        </CardFooter>
      </Card>
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
export function ErrorFallback({ error, resetErrorBoundary }: ErrorFallbackProps) {
  return (
    <DefaultErrorFallback 
      error={error} 
      onReset={resetErrorBoundary} 
      onReload={() => window.location.reload()} 
    />
  );
}
