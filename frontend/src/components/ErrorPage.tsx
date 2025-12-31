/**
 * Error page with a collapsed tower SVG.
 * Used for failed jobs and general errors.
 */

import { Button } from '@/components/ui/button';
import { CollapsedTowerIcon } from '@/components/icons';
import { ArrowLeft } from 'lucide-react';

interface ErrorPageProps {
  title?: string;
  message?: string;
  suggestion?: string;
  onBack?: () => void;
}

export function ErrorPage({ 
  title = "Something went wrong", 
  message = "We couldn't build your visualization.",
  suggestion,
  onBack,
}: ErrorPageProps) {
  return (
    <div className="flex-1 flex items-center justify-center p-6">
      <div className="max-w-md w-full text-center">
        {/* Collapsed tower illustration */}
        <CollapsedTowerIcon className="w-40 h-28 mx-auto mb-6 text-muted-foreground" />
        
        {/* Error message */}
        <h1 className="text-lg font-semibold mb-1">{title}</h1>
        <p className="text-sm text-muted-foreground mb-2">{message}</p>
        
        {/* Suggestion */}
        {suggestion && (
          <p className="text-xs text-muted-foreground bg-muted/50 rounded-lg px-3 py-2 mb-6 text-left">
            💡 {suggestion}
          </p>
        )}
        
        {/* Back button */}
        {onBack && (
          <Button variant="outline" size="sm" onClick={onBack} className="gap-1.5">
            <ArrowLeft className="h-3.5 w-3.5" />
            Go back
          </Button>
        )}
      </div>
    </div>
  );
}
