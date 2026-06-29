'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import type { SiteConfig } from '@/lib/types';

interface UseConfigResult {
  config: SiteConfig | null;
  loading: boolean;
  error: string | null;
}

const FALLBACK_SITE_NAME =
  process.env.NEXT_PUBLIC_SITE_NAME ?? 'VaultProxies Reseller';

export function useConfig(): UseConfigResult {
  const [config, setConfig] = useState<SiteConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    (async () => {
      try {
        const cfg = await api.getConfig(controller.signal);
        setConfig({ ...cfg, site_name: cfg.site_name || FALLBACK_SITE_NAME });
      } catch (err) {
        if ((err as Error)?.name === 'AbortError') return;
        setError('Unable to load site configuration.');
      } finally {
        setLoading(false);
      }
    })();
    return () => controller.abort();
  }, []);

  return { config, loading, error };
}
