// Server-side fetch helpers for public, cookie-less data (used in RSC).
// These never run in the browser and never carry credentials.

import { API_BASE_URL } from '@/lib/api';
import type { Plan, PlansResponse, SiteConfig } from '@/lib/types';

async function publicGet<T>(path: string): Promise<T | null> {
  if (!API_BASE_URL) return null;
  try {
    const res = await fetch(`${API_BASE_URL}/api/v1${path}`, {
      // Revalidate public catalog periodically.
      next: { revalidate: 300 },
      headers: { Accept: 'application/json' },
    });
    if (!res.ok) return null;
    return (await res.json()) as T;
  } catch {
    return null;
  }
}

export async function fetchPlansServer(): Promise<Plan[]> {
  const data = await publicGet<PlansResponse>('/plans');
  return data?.plans ?? [];
}

export async function fetchConfigServer(): Promise<SiteConfig | null> {
  return publicGet<SiteConfig>('/config');
}
