/**
 * Toast notification helpers for consistent UX.
 * 
 * Provides standardized toast messages with error handling.
 */

import { toast } from 'sonner';

/**
 * Extract error message from various error types.
 */
function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === 'string') {
    return error;
  }
  return 'An unknown error occurred';
}

/**
 * Show a success toast.
 * 
 * @param title - Success message
 * 
 * @example
 * showSuccess('Saved to library');
 */
export function showSuccess(title: string): void {
  toast.success(title);
}

/**
 * Show an error toast with optional description.
 * 
 * @param title - Error title
 * @param error - Optional error object, message, or description string
 * 
 * @example
 * showError('Failed to save', err);
 * showError('Network error');
 * showError('Package not found', 'Check the package name');
 */
export function showError(title: string, error?: unknown): void {
  if (error) {
    const description = getErrorMessage(error);
    toast.error(title, { 
      description,
      duration: 5000, // Longer duration for errors with details
    });
  } else {
    toast.error(title);
  }
}

/**
 * Show an info toast.
 * 
 * @param title - Info message
 * @param description - Optional description
 * 
 * @example
 * showInfo('Processing...', 'This may take a moment');
 */
export function showInfo(title: string, description?: string): void {
  if (description) {
    toast.info(title, { description });
  } else {
    toast.info(title);
  }
}

/**
 * Show a loading toast (returns ID for dismissal).
 * 
 * @param title - Loading message
 * @returns Toast ID for dismissal
 * 
 * @example
 * const id = showLoading('Uploading...');
 * // Later: toast.dismiss(id);
 */
export function showLoading(title: string): string | number {
  return toast.loading(title);
}

/**
 * Show an error toast with a retry action.
 * 
 * @param title - Error title
 * @param onRetry - Retry callback
 * @param error - Optional error object
 * 
 * @example
 * showErrorWithRetry('Failed to load', () => refetch(), err);
 */
export function showErrorWithRetry(
  title: string,
  onRetry: () => void,
  error?: unknown
): void {
  toast.error(title, {
    description: error ? getErrorMessage(error) : undefined,
    action: {
      label: 'Retry',
      onClick: onRetry,
    },
  });
}

