class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public data?: any,
  ) {
    super(`API Error: ${status} ${statusText}`);
    this.name = "ApiError";
  }
}

let isRefreshing = false;
let refreshPromise: Promise<void> | null = null;

async function refreshTokens(): Promise<void> {
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }

  isRefreshing = true;
  refreshPromise = fetch("/api/auth/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({}),
  })
    .then((response) => {
      if (!response.ok) {
        throw new Error("Token refresh failed");
      }
      return response.json();
    })
    .then(() => {
      isRefreshing = false;
      refreshPromise = null;
    })
    .catch((error) => {
      isRefreshing = false;
      refreshPromise = null;
      throw error;
    });

  return refreshPromise;
}

async function handleResponse<T>(
  response: Response,
  originalRequest?: () => Promise<Response>,
): Promise<T> {
  if (!response.ok) {
    // Handle 401 Unauthorized - try to refresh token
    if (response.status === 401 && originalRequest) {
      try {
        await refreshTokens();
        // Retry the original request
        const retryResponse = await originalRequest();
        return handleResponse<T>(retryResponse);
      } catch {
        // Refresh failed, let the error propagate
      }
    }

    let errorData;
    try {
      errorData = await response.json();
    } catch {
      // Response might not be JSON
    }
    throw new ApiError(response.status, response.statusText, errorData);
  }

  // Handle empty responses (like 204 No Content)
  const contentType = response.headers.get("content-type");
  if (!contentType || !contentType.includes("application/json")) {
    return {} as T;
  }

  return response.json();
}

function buildUrl(path: string, params?: URLSearchParams): string {
  const base = path.startsWith("/") ? path : `/${path}`;
  if (params && params.toString()) {
    return `${base}?${params.toString()}`;
  }
  return base;
}

function getHeaders(): Record<string, string> {
  return {
    "Content-Type": "application/json",
  };
}

export async function apiGet<T>(
  path: string,
  params?: URLSearchParams,
): Promise<T> {
  const url = buildUrl(path, params);
  const makeRequest = () =>
    fetch(url, {
      method: "GET",
      headers: getHeaders(),
      credentials: "include", // Include cookies in requests
    });
  const response = await makeRequest();
  return handleResponse<T>(response, makeRequest);
}

export async function apiPost<T>(path: string, body: any): Promise<T> {
  const url = buildUrl(path);
  const makeRequest = () =>
    fetch(url, {
      method: "POST",
      headers: getHeaders(),
      credentials: "include", // Include cookies in requests
      body: JSON.stringify(body),
    });
  const response = await makeRequest();
  return handleResponse<T>(response, makeRequest);
}

export async function apiPut<T>(path: string, body: any): Promise<T> {
  const url = buildUrl(path);
  const makeRequest = () =>
    fetch(url, {
      method: "PUT",
      headers: getHeaders(),
      credentials: "include", // Include cookies in requests
      body: JSON.stringify(body),
    });
  const response = await makeRequest();
  return handleResponse<T>(response, makeRequest);
}

export async function apiDelete(path: string): Promise<void> {
  const url = buildUrl(path);
  const makeRequest = () =>
    fetch(url, {
      method: "DELETE",
      headers: getHeaders(),
      credentials: "include", // Include cookies in requests
    });
  const response = await makeRequest();
  await handleResponse<void>(response, makeRequest);
}

export { ApiError };
