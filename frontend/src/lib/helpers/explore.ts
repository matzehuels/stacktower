/**
 * Helper functions for explore/discovery features.
 */

import { toast } from 'sonner';
import { getRender } from '@/lib/api';
import type { ExploreEntry, JobResponse } from '@/types/api';

/**
 * Select and load an explore entry (package visualization).
 * 
 * Fetches the full render data for a package, preferring tower visualization.
 * Adds related render IDs for viz type switching.
 * 
 * @param entry - The explore entry to select
 * @param onSelect - Callback to handle the loaded job
 * 
 * @example
 * const handleSelect = (entry: ExploreEntry) => {
 *   selectExploreEntry(entry, (job, inLibrary) => {
 *     setSelectedJob(job);
 *   });
 * };
 */
export async function selectExploreEntry(
  entry: ExploreEntry,
  onSelect: (job: JobResponse, inLibrary?: boolean) => void
): Promise<void> {
  // Prefer tower visualization, fallback to first available
  const towerViz = entry.viz_types.find((v) => v.viz_type === 'tower');
  const selectedViz = towerViz || entry.viz_types[0];
  
  if (!selectedViz) {
    toast.error('No visualization available');
    return;
  }

  try {
    // Fetch the full render data (includes layout with Nebraska rankings)
    const jobResponse = await getRender(selectedViz.render_id);
    
    // Add related render IDs for viz type switching
    const allRenderIds = entry.viz_types.map((v) => v.render_id).filter(Boolean);
    if (jobResponse.result) {
      jobResponse.result.related_render_ids = allRenderIds;
    }
    
    onSelect(jobResponse, entry.in_library);
  } catch (error) {
    console.error('Failed to fetch render:', error);
    toast.error('Failed to load visualization');
  }
}

