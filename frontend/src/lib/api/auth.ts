import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut, apiDelete } from "../api-client";
import type {
  User,
  LoginRequest,
  RegisterRequest,
  CreateUserRequest,
  UpdateUserRequest,
  UsersListResponse,
} from "../types";

// Auth endpoints
export function useLogin() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (credentials: LoginRequest) =>
      apiPost<{ user: User }>("/api/auth/login", credentials),
    onSuccess: (data) => {
      console.log("Login successful");
      // Tokens are now in httpOnly cookies, just cache the user
      queryClient.setQueryData(["auth", "me"], data.user);
    },
  });
}

export function useRegister() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: RegisterRequest) =>
      apiPost<{ user: User }>("/api/auth/register", data),
    onSuccess: (data) => {
      console.log("Registration successful");
      // Tokens are now in httpOnly cookies, just cache the user
      queryClient.setQueryData(["auth", "me"], data.user);
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      try {
        await apiPost("/api/auth/logout", {});
      } catch {
        // Ignore errors, we're logging out anyway
      }
    },
    onSuccess: () => {
      // Cookies are cleared by the server, just clear client cache
      queryClient.setQueryData(["auth", "me"], null);
      queryClient.clear();
    },
  });
}

export function useRefreshToken() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      return apiPost<{ message: string }>("/api/auth/refresh", {});
    },
    onSuccess: () => {
      console.log("Tokens refreshed successfully");
      // Invalidate current user to refetch with new token
      queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    },
    onError: () => {
      // If refresh fails, log out the user
      queryClient.setQueryData(["auth", "me"], null);
      queryClient.clear();
    },
  });
}

export function useCurrentUser() {
  return useQuery<User | null>({
    queryKey: ["auth", "me"],
    queryFn: async () => {
      try {
        return await apiGet<User>("/api/auth/me");
      } catch {
        // If not authenticated, return null
        return null;
      }
    },
    staleTime: 1000 * 60 * 5, // 5 minutes
    retry: false,
  });
}

// User management endpoints (admin only)
export function useUsers() {
  return useQuery<UsersListResponse>({
    queryKey: ["users"],
    queryFn: () => apiGet<UsersListResponse>("/api/users"),
  });
}

export function useUser(id: string | number) {
  return useQuery<User>({
    queryKey: ["users", id],
    queryFn: () => apiGet<User>(`/api/users/${id}`),
    enabled: !!id,
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateUserRequest) => apiPost<User>("/api/users", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
    },
  });
}

export function useUpdateUser(id: string | number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateUserRequest) =>
      apiPut<User>(`/api/users/${id}`, data),
    onSuccess: (data) => {
      queryClient.setQueryData(["users", id], data);
      queryClient.invalidateQueries({ queryKey: ["users"] });
    },
  });
}

export function useDeleteUser(id: string | number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => apiDelete(`/api/users/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
    },
  });
}
