// @ts-nocheck - TODO: Fix types for v2. See V2-619.
/**
 * Legacy API client for Tent of Trials.
 * Handles HTTP requests with automatic token refresh on 401.
 */

const API_BASE = process.env.REACT_APP_API_URL || '/api';

let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

function getAuthToken(): string | null {
  return localStorage.getItem('access_token');
}

function getRefreshToken(): string | null {
  return localStorage.getItem('refresh_token');
}

function setTokens(access: string, refresh?: string): void {
  localStorage.setItem('access_token', access);
  if (refresh) localStorage.setItem('refresh_token', refresh);
}

function clearTokens(): void {
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
}

async function refreshToken(): Promise<boolean> {
  if (isRefreshing && refreshPromise) return refreshPromise;
  
  const refresh = getRefreshToken();
  if (!refresh) return false;

  isRefreshing = true;
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refresh }),
      });
      if (!res.ok) { clearTokens(); return false; }
      const data = await res.json();
      setTokens(data.access_token, data.refresh_token);
      return true;
    } catch { clearTokens(); return false; }
    finally { isRefreshing = false; refreshPromise = null; }
  })();
  return refreshPromise;
}

async function request(method: string, path: string, body?: unknown, retry = true): Promise<unknown> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = getAuthToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (res.status === 401 && retry) {
    const refreshed = await refreshToken();
    if (refreshed) return request(method, path, body, false);
    throw new Error('Authentication failed');
  }
  if (!res.ok) throw new Error(`Request failed: ${res.status}`);
  return res.json();
}

export const get = (path: string) => request('GET', path);
export const post = (path: string, body?: unknown) => request('POST', path, body);
export const put = (path: string, body?: unknown) => request('PUT', path, body);
export const del = (path: string) => request('DELETE', path);