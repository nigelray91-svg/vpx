// Shared API types matching the VPX backend contract (docs/API.md).

export interface SiteConfig {
  site_name: string;
  turnstile_site_key: string;
  turnstile_enabled: boolean;
  google_enabled: boolean;
  stripe_enabled: boolean;
  crypto_enabled: boolean;
  min_topup_cents: number;
}

export type ProxyType =
  | 'residential'
  | 'isp'
  | 'datacenter'
  | 'ipv6'
  | 'mobile'
  | string;

export interface Plan {
  id: string;
  code: string;
  name: string;
  proxy_type: ProxyType;
  unit: string;
  price_cents: number;
  min_quantity: number;
}

export interface PlansResponse {
  plans: Plan[];
}

export type UserRole = 'user' | 'admin' | string;

export interface User {
  id: string;
  email: string;
  full_name: string;
  role: UserRole;
  balance_cents: number;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  expires_in: number;
}

export interface Wallet {
  balance_cents: number;
  currency: string;
}

export interface LedgerEntry {
  id: string;
  amount_cents: number;
  kind: string;
  reference?: string;
  description: string;
  balance_after: number;
  created_at: string;
}

export interface LedgerResponse {
  entries: LedgerEntry[];
}

export interface Payment {
  id: string;
  amount_cents: number;
  provider: string;
  status: string;
  created_at: string;
}

export interface PaymentsResponse {
  payments: Payment[];
}

export interface TopupResponse {
  checkout_url: string;
}

export type Rotation = 'rotating' | 'sticky';

export interface Order {
  id: string;
  plan_id: string;
  plan_code?: string;
  plan_name?: string;
  quantity: number;
  rotation?: Rotation;
  region?: string;
  total_cents: number;
  status: string;
  created_at: string;
}

export interface OrdersResponse {
  orders: Order[];
}

export interface Proxy {
  id: string;
  proxy_type: ProxyType;
  protocol: string;
  host: string;
  port: number;
  username: string;
  password: string;
  pool: string;
  rotation: Rotation;
  sticky_ttl_seconds: number;
  bandwidth_limit_bytes: number;
  bandwidth_used_bytes: number;
  status: string;
  expires_at: string | null;
  created_at: string;
}

export interface ProxiesResponse {
  proxies: Proxy[];
}

export interface ProxyUsage {
  remaining_gb: number | null;
  active: boolean;
  expires_at?: string | null;
}

export interface CreateOrderRequest {
  plan_id: string;
  quantity: number;
}

export interface CreateOrderResponse {
  order: Order;
  proxies: Proxy[];
}

export interface ApiErrorShape {
  error?: string;
  code?: string;
  fields?: Record<string, string>;
}
