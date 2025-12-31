/**
 * Error handling utilities for user-friendly error messages.
 * 
 * Parses different error types and provides specific, actionable feedback.
 */

import { ApiError } from '../api/client';

export interface ParsedError {
  title: string;
  message: string;
  suggestion?: string;
  retryable: boolean;
}

/**
 * Parse an error into a user-friendly format with title, message, and suggestions.
 */
export function parseError(error: unknown, context?: string): ParsedError {
  // Handle ApiError from HTTP requests
  if (error instanceof ApiError) {
    return parseApiError(error, context);
  }

  // Handle Error objects
  if (error instanceof Error) {
    return {
      title: 'An error occurred',
      message: error.message,
      retryable: true,
    };
  }

  // Handle string errors
  if (typeof error === 'string') {
    return parseErrorMessage(error, context);
  }

  // Unknown error type
  return {
    title: 'An unexpected error occurred',
    message: 'Something went wrong. Please try again.',
    retryable: true,
  };
}

/**
 * Parse ApiError based on HTTP status and error message.
 */
function parseApiError(error: ApiError, context?: string): ParsedError {
  const { status, message } = error;

  // 404 - Not Found
  if (status === 404) {
    return {
      title: 'Package not found',
      message: `The ${context || 'package'} you requested could not be found.`,
      suggestion: 'Please check the package name and try again. For Go modules, make sure to use the full module path (e.g., github.com/user/repo).',
      retryable: false,
    };
  }

  // 400 - Bad Request (validation errors)
  if (status === 400) {
    return {
      title: 'Invalid request',
      message: message || 'The request contains invalid data.',
      suggestion: 'Please check your input and try again.',
      retryable: false,
    };
  }

  // 401 - Unauthorized
  if (status === 401) {
    return {
      title: 'Authentication required',
      message: 'You need to sign in to access this feature.',
      retryable: false,
    };
  }

  // 403 - Forbidden
  if (status === 403) {
    return {
      title: 'Access denied',
      message: 'You don\'t have permission to access this resource.',
      retryable: false,
    };
  }

  // 429 - Rate Limited
  if (status === 429) {
    return {
      title: 'Too many requests',
      message: 'You\'ve made too many requests. Please wait a moment before trying again.',
      suggestion: 'Try again in a few minutes.',
      retryable: true,
    };
  }

  // 500+ - Server errors
  if (status >= 500) {
    return {
      title: 'Server error',
      message: 'Our servers encountered an error processing your request.',
      suggestion: 'Please try again in a few moments. If the problem persists, contact support.',
      retryable: true,
    };
  }

  // Network/timeout errors (status 0 or network error message)
  if (status === 0 || message.toLowerCase().includes('network') || message.toLowerCase().includes('timeout')) {
    return {
      title: 'Network error',
      message: 'Could not connect to the server.',
      suggestion: 'Please check your internet connection and try again.',
      retryable: true,
    };
  }

  // Generic API error
  return {
    title: 'Request failed',
    message: message || 'The request could not be completed.',
    retryable: true,
  };
}

/**
 * Parse error message string for known patterns.
 */
function parseErrorMessage(message: string, context?: string): ParsedError {
  const lowerMessage = message.toLowerCase();

  // Not found errors
  if (lowerMessage.includes('not found')) {
    return {
      title: 'Package not found',
      message: `The ${context || 'package'} could not be found.`,
      suggestion: 'Please verify the package name and registry. For Go, use the full module path (e.g., github.com/user/repo).',
      retryable: false,
    };
  }

  // Network/timeout errors
  if (lowerMessage.includes('network') || lowerMessage.includes('timeout') || lowerMessage.includes('fetch')) {
    return {
      title: 'Network error',
      message: 'Could not connect to the server.',
      suggestion: 'Check your internet connection and try again.',
      retryable: true,
    };
  }

  // Dependency resolution errors
  if (lowerMessage.includes('resolve') || lowerMessage.includes('dependency')) {
    return {
      title: 'Dependency resolution failed',
      message: 'Could not resolve all package dependencies.',
      suggestion: 'The package or one of its dependencies might not exist or is unavailable.',
      retryable: true,
    };
  }

  // Go module specific errors
  if (lowerMessage.includes('go module') || lowerMessage.includes('proxy.golang.org')) {
    return {
      title: 'Go module not found',
      message: 'The Go module could not be found.',
      suggestion: 'Make sure you\'re using the full module path (e.g., github.com/spf13/cobra, not just "cobra").',
      retryable: false,
    };
  }

  // Parse errors
  if (lowerMessage.includes('parse') || lowerMessage.includes('invalid')) {
    return {
      title: 'Invalid format',
      message: 'The package data could not be parsed.',
      suggestion: 'The package may have an invalid or unsupported manifest format.',
      retryable: false,
    };
  }

  // Generic error
  return {
    title: 'An error occurred',
    message,
    retryable: true,
  };
}

/**
 * Get a concise error message suitable for toast notifications.
 */
export function getErrorSummary(error: unknown): string {
  const parsed = parseError(error);
  return parsed.message;
}

/**
 * Check if an error is retryable (user should try again).
 */
export function isRetryable(error: unknown): boolean {
  const parsed = parseError(error);
  return parsed.retryable;
}

