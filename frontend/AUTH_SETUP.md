# Authentication Setup Guide

## Current Implementation

The frontend is configured for **JWT token-based authentication** with the token stored in localStorage.

## Debugging Token Storage

When you log in, check the browser console. You should see:
```
Login response: { user: {...}, token: "..." }
Token saved to localStorage
```

If you see `"No token in login response"`, your backend is not returning a token in the expected format.

## Backend Response Formats

The frontend expects one of these response formats from `/api/auth/login`:

### Option 1: JWT Token (Current Setup)
```json
{
  "user": {
    "id": 1,
    "username": "john",
    "email": "john@example.com",
    "is_admin": true,
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "xyz123...",
    "expires_at": "2024-01-01T01:00:00Z",
    "token_type": "Bearer"
  }
}
```

### Option 2: Session Cookie (Requires Code Changes)
If your backend uses session cookies instead of JWT tokens, you need to modify the frontend to use `credentials: 'include'` in fetch requests and remove token storage logic.

## Checking Token Storage

Open browser DevTools:
1. Go to **Application** tab (Chrome) or **Storage** tab (Firefox)
2. Look under **Local Storage** → `http://localhost:5173`
3. You should see `auth_token` with your JWT

Or run in console:
```javascript
localStorage.getItem('auth_token')
```

## Common Issues

### 1. Token Not in Response
**Problem:** Backend doesn't include `token` field  
**Solution:** Update your backend to return the token, or modify frontend for session-based auth

### 2. Different Token Field Name
**Problem:** Backend uses `access_token` instead of `token`  
**Solution:** Update `src/lib/types.ts`:
```typescript
export interface AuthResponse {
  user: User;
  access_token?: string;  // Change field name
}
```

Then update `src/lib/api/auth.ts`:
```typescript
if (data.access_token) {
  localStorage.setItem("auth_token", data.access_token);
}
```

### 3. Session-Based Auth (No Token)
If your backend uses cookies/sessions instead of tokens:

1. Update `src/lib/api-client.ts`:
```typescript
export async function apiGet<T>(
  path: string,
  params?: URLSearchParams,
): Promise<T> {
  const url = buildUrl(path, params);
  const response = await fetch(url, {
    method: "GET",
    headers: getAuthHeaders(),
    credentials: "include",  // Add this
  });
  return handleResponse<T>(response);
}
```

2. Remove token storage from auth hooks
3. Backend must set httpOnly cookies

### 4. CORS Issues with Cookies
If using cookies, your backend needs:
```go
// Allow credentials
AllowCredentials: true

// Specific origin (not wildcard)
AllowOrigins: []string{"http://localhost:5173"}
```

## Testing Authentication

1. Open browser DevTools Console
2. Try logging in
3. Check console output for "Login response"
4. Verify token is present in response
5. Check localStorage for `auth_token`
6. Try accessing a protected route

## Manual Token Test

You can manually set a token to test:
```javascript
localStorage.setItem('auth_token', 'your-test-token-here')
```

Then refresh the page. If the token is valid, you should be logged in.

## Backend Requirements

Your Go backend should implement:

### POST /api/auth/login
**Request:**
```json
{
  "username": "john",
  "password": "password123"
}
```

**Response (200 OK):**
```json
{
  "user": {
    "id": 1,
    "username": "john",
    "email": "john@example.com",
    "is_admin": true,
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "xyz123...",
    "expires_at": "2024-01-01T01:00:00Z",
    "token_type": "Bearer"
  }
}
```

### GET /api/auth/me
**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200 OK):**
```json
{
  "id": 1,
  "username": "john",
  "email": "john@example.com",
  "is_admin": true,
  "is_active": true,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Response (401 Unauthorized):** If token is invalid or missing
