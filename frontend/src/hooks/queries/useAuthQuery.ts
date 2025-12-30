/**
 * Authentication queries and mutations.
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getCurrentUser, logout as logoutApi, getLoginUrl } from '@/lib/api';
import { queryKeys } from './keys';

/**
 * Query for fetching the current authenticated user.
 */
export function useCurrentUser() {
  return useQuery({
    queryKey: queryKeys.auth.me(),
    queryFn: getCurrentUser,
    staleTime: 5 * 60 * 1000, // User data is stable, cache for 5 minutes
  });
}

/**
 * Mutation for logging out.
 */
export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: logoutApi,
    onSuccess: () => {
      // Clear the user from cache
      queryClient.setQueryData(queryKeys.auth.me(), null);
      // Invalidate all queries since user context changed
      queryClient.invalidateQueries();
    },
  });
}

/**
 * Redirect to GitHub OAuth login.
 */
export function useLogin() {
  return {
    login: () => {
      window.location.href = getLoginUrl();
    },
  };
}
