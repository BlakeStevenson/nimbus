export type MediaKind =
  | "movie"
  | "tv_series"
  | "tv_season"
  | "tv_episode"
  | "music_artist"
  | "music_album"
  | "music_track"
  | "book"
  | string; // allow custom kinds for plugins

export interface MediaItem {
  id: number | string;
  kind: MediaKind;
  title: string;
  sort_title: string;
  year?: number | null;
  parent_id?: number | string | null;
  external_ids?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface MediaListResponse {
  items: MediaItem[];
  total: number;
}

export interface ConfigValue {
  key: string;
  value: any;
  updated_at: string;
}

// For future plugins
export interface PluginUiManifest {
  id: string;
  displayName: string;
  navItems: Array<{
    label: string;
    path: string;
    group?: string;
    icon?: string;
  }>;
  routes: Array<{
    path: string;
    bundleUrl: string;
  }>;
}

export interface Plugin {
  id: string;
  name: string;
  version: string;
  enabled: boolean;
  capabilities?: string[];
  description?: string;
}

export interface MediaFilters {
  kind?: string;
  q?: string;
  parentId?: string | number;
  limit?: number;
  offset?: number;
}

export interface UpdateMediaPayload {
  title?: string;
  year?: number | null;
  metadata?: Record<string, unknown>;
}

// Authentication types
export interface User {
  id: number | string;
  username: string;
  email?: string;
  is_admin: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email?: string;
  password: string;
}

export interface TokenData {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  token_type: string;
}

export interface AuthResponse {
  user: User;
  tokens: TokenData;
}

export interface CreateUserRequest {
  username: string;
  email?: string;
  password: string;
  is_admin?: boolean;
}

export interface UpdateUserRequest {
  email?: string;
  password?: string;
  is_admin?: boolean;
}

export interface UsersListResponse {
  users: User[];
  total: number;
}
