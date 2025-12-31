/**
 * 404 Not Found page component with collapsed tower animation.
 */

import { Button } from '@/components/ui/button';
import { CollapsedTowerIcon } from '@/components/icons';
import { Home } from 'lucide-react';

export function NotFound() {
  return (
    <div className="h-screen bg-background flex items-center justify-center p-6">
      <div className="max-w-md w-full text-center">
        {/* Collapsed tower illustration */}
        <CollapsedTowerIcon className="w-40 h-28 mx-auto mb-6 text-muted-foreground" />
        
        {/* Error message */}
        <h1 className="text-2xl font-bold mb-2">404 - Page Not Found</h1>
        <p className="text-sm text-muted-foreground mb-6">
          The page you're looking for doesn't exist.
        </p>
        
        {/* Action */}
        <Button asChild size="sm" className="gap-1.5">
          <a href="/">
            <Home className="h-3.5 w-3.5" />
            Go to homepage
          </a>
        </Button>
      </div>
    </div>
  );
}

