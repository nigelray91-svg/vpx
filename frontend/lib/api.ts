// Typed API client for the VPX backend.
//
// Auth model (see docs/API.md):
//  - every request sends credentials: 'include' (httpOnly cookies).
//  - state-changing methods echo the readable `vpx_csrf` cookie as
//    the `X-CSRF-Token` header (double-submit CSRF).
//  - on a 401, we attempt POST /api/v1/auth/refresh once, then retry.

import type {
  AuthResponse,
  CreateOrderRequest,
  CreateOrderResponse,
  GenerateRequest,
  GenerateResponse,
  LedgerResponse,
  LocationCountry,
  LocationsResponse,
  OrdersResponse,
  PaymentsResponse,
  Plan,
  PlansResponse,
  ProxiesResponse,
  ProxyUsage,
  SiteConfig,
  TopupResponse,
  User,
  Wallet,
} from './types';

export const API_BASE_URL = (
  process.env.NEXT_PUBLIC_API_BASE_URL ?? ''
).replace(/\/$/, '');

const API_PREFIX = '/api/v1';

export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly fields?: Record<string, string>;

  constructor(
    message: string,
    status: number,
    code?: string,
    fields?: Record<string, string>,
  ) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.fields = fields;
  }
}

const UNSAFE_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

/** Read a cookie value by name from document.cookie (client only). */
export function readCookie(name: string): string | null {
  if (typeof document === 'undefined') return null;
  const target = `${name}=`;
  const parts = document.cookie ? document.cookie.split('; ') : [];
  for (const part of parts) {
    if (part.startsWith(target)) {
      return decodeURIComponent(part.slice(target.length));
    }
  }
  return null;
}

function buildUrl(path: string): string {
  // Allow callers to pass either "/me" or "/api/v1/me".
  const normalized = path.startsWith(API_PREFIX)
    ? path
    : `${API_PREFIX}${path.startsWith('/') ? '' : '/'}${path}`;
  return `${API_BASE_URL}${normalized}`;
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  // internal: prevents infinite refresh loops
  _retried?: boolean;
  signal?: AbortSignal;
}

async function parseError(res: Response): Promise<ApiError> {
  let message = res.statusText || 'Request failed';
  let code: string | undefined;
  let fields: Record<string, string> | undefined;
  try {
    const data = (await res.json()) as {
      error?: string;
      code?: string;
      fields?: Record<string, string>;
    };
    if (data?.error) message = data.error;
    code = data?.code;
    fields = data?.fields;
  } catch {
    // non-JSON error body — keep status text.
  }
  return new ApiError(message, res.status, code, fields);
}

async function rawRequest<T>(
  path: string,
  options: RequestOptions,
): Promise<T> {
  const method = (options.method ?? 'GET').toUpperCase();
  const headers: Record<string, string> = { Accept: 'application/json' };

  let bodyInit: BodyInit | undefined;
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    bodyInit = JSON.stringify(options.body);
  }

  if (UNSAFE_METHODS.has(method)) {
    const csrf = readCookie('vpx_csrf');
    if (csrf) headers['X-CSRF-Token'] = csrf;
  }

  let res: Response;
  try {
    res = await fetch(buildUrl(path), {
      method,
      headers,
      body: bodyInit,
      credentials: 'include',
      signal: options.signal,
    });
  } catch (err) {
    if ((err as Error)?.name === 'AbortError') throw err;
    throw new ApiError('Network error — could not reach the API.', 0);
  }

  // Auto-refresh on 401, then retry once.
  if (
    res.status === 401 &&
    !options._retried &&
    path !== '/auth/refresh' &&
    path !== `${API_PREFIX}/auth/refresh`
  ) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      return rawRequest<T>(path, { ...options, _retried: true });
    }
  }

  if (!res.ok) {
    throw await parseError(res);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

let refreshInFlight: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight;
  refreshInFlight = (async () => {
    try {
      const headers: Record<string, string> = {};
      const csrf = readCookie('vpx_csrf');
      if (csrf) headers['X-CSRF-Token'] = csrf;
      const res = await fetch(buildUrl('/auth/refresh'), {
        method: 'POST',
        headers,
        credentials: 'include',
      });
      return res.ok;
    } catch {
      return false;
    } finally {
      // cleared after the awaiting callers resolve
      setTimeout(() => {
        refreshInFlight = null;
      }, 0);
    }
  })();
  return refreshInFlight;
}

function get<T>(path: string, signal?: AbortSignal): Promise<T> {
  return rawRequest<T>(path, { method: 'GET', signal });
}

function post<T>(path: string, body?: unknown): Promise<T> {
  return rawRequest<T>(path, { method: 'POST', body });
}

// ---- Public endpoints ----

export const api = {
  getConfig: (signal?: AbortSignal) => get<SiteConfig>('/config', signal),

  getPlans: async (signal?: AbortSignal): Promise<Plan[]> => {
    const data = await get<PlansResponse>('/plans', signal);
    return data.plans ?? [];
  },

  // ---- Auth ----
  register: (body: {
    email: string;
    password: string;
    full_name?: string;
    turnstile_token?: string;
  }) => post<AuthResponse>('/auth/register', body),

  login: (body: {
    email: string;
    password: string;
    turnstile_token?: string;
  }) => post<AuthResponse>('/auth/login', body),

  logout: () => post<void>('/auth/logout'),

  refresh: () => post<void>('/auth/refresh'),

  // ---- Account / wallet ----
  me: (signal?: AbortSignal) => get<User>('/me', signal),

  getWallet: (signal?: AbortSignal) => get<Wallet>('/wallet', signal),

  getLedger: async (signal?: AbortSignal) => {
    const data = await get<LedgerResponse>('/wallet/ledger', signal);
    return data.entries ?? [];
  },

  getPayments: async (signal?: AbortSignal) => {
    const data = await get<PaymentsResponse>('/payments', signal);
    return data.payments ?? [];
  },

  topup: (body: { amount_cents: number; provider: 'stripe' | 'nowpayments' }) =>
    post<TopupResponse>('/wallet/topup', body),

  // ---- Orders / proxies ----
  getOrders: async (signal?: AbortSignal) => {
    const data = await get<OrdersResponse>('/orders', signal);
    return data.orders ?? [];
  },

  createOrder: (body: CreateOrderRequest) =>
    post<CreateOrderResponse>('/orders', body),

  getProxies: async (signal?: AbortSignal) => {
    const data = await get<ProxiesResponse>('/proxies', signal);
    return data.proxies ?? [];
  },

  getProxyUsage: (id: string, signal?: AbortSignal) =>
    get<ProxyUsage>(`/proxies/${encodeURIComponent(id)}/usage`, signal),

  // ---- Generator ----
  // Generating mints credentials; it does not consume bandwidth, so it can be
  // repeated as often as needed.
  generateProxies: (id: string, body: GenerateRequest) =>
    post<GenerateResponse>(`/proxies/${encodeURIComponent(id)}/generate`, body),

  getLocations: async (
    planKey: string,
    country?: string,
    signal?: AbortSignal,
  ): Promise<LocationCountry[]> => {
    const params = new URLSearchParams({ plan_key: planKey });
    if (country) params.set('country', country);
    const data = await get<LocationsResponse>(
      `/locations?${params.toString()}`,
      signal,
    );
    return data.countries ?? [];
  },
};

export function googleAuthUrl(): string {
  return `${API_BASE_URL}${API_PREFIX}/auth/google`;
}
