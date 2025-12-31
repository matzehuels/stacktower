/**
 * Hook for sharing visualizations via URL.
 * 
 * Provides functionality to copy a shareable link to clipboard
 * and track the copied state for UI feedback.
 * 
 * @example
 * const { share, justCopied } = useShareLink();
 * 
 * <Button onClick={() => share(jobId)}>
 *   {justCopied ? 'Copied!' : 'Share'}
 * </Button>
 */

import { useState, useCallback } from 'react';
import { toast } from 'sonner';

export function useShareLink() {
  const [justCopied, setJustCopied] = useState(false);

  const share = useCallback(async (jobId: string) => {
    if (!jobId || jobId === 'cached' || jobId === '__cached__') {
      toast.error('Cannot share this visualization', {
        description: 'Only saved visualizations can be shared.'
      });
      return;
    }

    const shareUrl = `${window.location.origin}?render=${jobId}`;
    
    try {
      await navigator.clipboard.writeText(shareUrl);
      setJustCopied(true);
      toast.success('Share link copied!', {
        description: 'Anyone with this link can view this visualization.'
      });
      
      // Reset the "just copied" state after 2 seconds
      setTimeout(() => setJustCopied(false), 2000);
    } catch {
      toast.error('Failed to copy link', {
        description: 'Please try again or copy the URL manually.'
      });
    }
  }, []);

  return { share, justCopied };
}

