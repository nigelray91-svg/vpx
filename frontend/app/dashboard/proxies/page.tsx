'use client';

import { useEffect, useState } from 'react';
import { Check, Copy, Eye, EyeOff, Gauge } from 'lucide-react';
import { api, ApiError } from '@/lib/api';
import type { Proxy, ProxyUsage } from '@/lib/types';
import {
  Badge,
  Card,
  EmptyState,
  ErrorState,
  LinkButton,
  Spinner,
  statusTone,
} from '@/components/ui';
import { useToast } from '@/components/Toast';
import { formatBytes, formatDate, formatDuration, titleCase } from '@/lib/format';

export default function ProxiesPage() {
  const [proxies, setProxies] = useState<Proxy[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = () => {
    setError(null);
    api
      .getProxies()
      .then(setProxies)
      .catch((err) =>
        setError(err instanceof ApiError ? err.message : 'Failed to load proxies.'),
      );
  };

  useEffect(load, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-white">Proxies</h1>

      {error ? (
        <ErrorState message={error} onRetry={load} />
      ) : !proxies ? (
        <Spinner label="Loading proxies…" />
      ) : proxies.length === 0 ? (
        <EmptyState
          title="No proxies yet"
          description="Place an order to provision your first proxy endpoint."
          action={<LinkButton href="/dashboard/orders">Create an order</LinkButton>}
        />
      ) : (
        <div className="space-y-4">
          {proxies.map((p) => (
            <ProxyCard key={p.id} proxy={p} />
          ))}
        </div>
      )}
    </div>
  );
}

function ProxyCard({ proxy }: { proxy: Proxy }) {
  const { notify } = useToast();
  const [reveal, setReveal] = useState(false);
  const [usage, setUsage] = useState<ProxyUsage | null>(null);
  const [loadingUsage, setLoadingUsage] = useState(false);

  const connString = `${proxy.host}:${proxy.port}:${proxy.username}:${proxy.password}`;
  const curl = `curl -x ${proxy.protocol}://${proxy.username}:${proxy.password}@${proxy.host}:${proxy.port} https://api.ipify.org`;

  const copy = async (value: string, label: string) => {
    try {
      await navigator.clipboard.writeText(value);
      notify(`${label} copied`, 'success');
    } catch {
      notify('Could not copy to clipboard', 'error');
    }
  };

  const fetchUsage = async () => {
    setLoadingUsage(true);
    try {
      setUsage(await api.getProxyUsage(proxy.id));
    } catch {
      notify('Could not load usage', 'error');
    } finally {
      setLoadingUsage(false);
    }
  };

  return (
    <Card>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-white">{titleCase(proxy.proxy_type)}</h3>
            <Badge tone={statusTone(proxy.status)}>{proxy.status}</Badge>
            <Badge tone="info">{proxy.rotation}</Badge>
          </div>
          <p className="mt-1 text-xs text-slate-500">
            {proxy.protocol.toUpperCase()} · created {formatDate(proxy.created_at)}
            {proxy.expires_at ? ` · expires ${formatDate(proxy.expires_at)}` : ''}
            {proxy.rotation === 'sticky' && proxy.sticky_ttl_seconds > 0
              ? ` · sticky ${formatDuration(proxy.sticky_ttl_seconds)}`
              : ''}
          </p>
        </div>
        <button
          onClick={fetchUsage}
          disabled={loadingUsage}
          className="inline-flex items-center gap-1.5 rounded-lg border border-ink-600 px-3 py-1.5 text-xs font-medium text-slate-300 hover:border-brand-500/50 disabled:opacity-50"
        >
          <Gauge className="h-3.5 w-3.5" />
          {loadingUsage ? 'Loading…' : 'Check usage'}
        </button>
      </div>

      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        <Detail label="Host" value={proxy.host} onCopy={() => copy(proxy.host, 'Host')} />
        <Detail label="Port" value={String(proxy.port)} onCopy={() => copy(String(proxy.port), 'Port')} />
        <Detail label="Username" value={proxy.username} onCopy={() => copy(proxy.username, 'Username')} />
        <Detail
          label="Password"
          value={reveal ? proxy.password : '••••••••••••'}
          onCopy={() => copy(proxy.password, 'Password')}
          extra={
            <button
              onClick={() => setReveal((v) => !v)}
              className="text-slate-400 hover:text-white"
              aria-label={reveal ? 'Hide password' : 'Reveal password'}
            >
              {reveal ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          }
        />
      </div>

      <div className="mt-4 space-y-2">
        <CopyRow label="host:port:user:pass" value={connString} onCopy={() => copy(connString, 'Connection string')} />
        <CopyRow label="cURL example" value={curl} onCopy={() => copy(curl, 'cURL command')} mono />
      </div>

      {usage && (
        <div className="mt-4 rounded-lg border border-ink-600 bg-ink-850 p-3 text-sm text-slate-300">
          <p>
            Bandwidth used: <span className="font-semibold text-white">{formatBytes(usage.bandwidth_used_bytes)}</span>
            {usage.bandwidth_cap_bytes > 0 && <> / {formatBytes(usage.bandwidth_cap_bytes)}</>}
            {' · '}
            <Badge tone={usage.active ? 'success' : 'danger'}>{usage.active ? 'active' : 'inactive'}</Badge>
          </p>
        </div>
      )}
    </Card>
  );
}

function Detail({
  label,
  value,
  onCopy,
  extra,
}: {
  label: string;
  value: string;
  onCopy: () => void;
  extra?: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-ink-600 bg-ink-850 px-3 py-2">
      <p className="text-xs text-slate-500">{label}</p>
      <div className="flex items-center justify-between gap-2">
        <span className="truncate font-mono text-sm text-slate-100">{value}</span>
        <div className="flex items-center gap-2">
          {extra}
          <button onClick={onCopy} className="text-slate-400 hover:text-white" aria-label={`Copy ${label}`}>
            <Copy className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}

function CopyRow({
  label,
  value,
  onCopy,
  mono,
}: {
  label: string;
  value: string;
  onCopy: () => void;
  mono?: boolean;
}) {
  const [copied, setCopied] = useState(false);
  const handle = () => {
    onCopy();
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return (
    <div className="flex items-center gap-2 rounded-lg border border-ink-600 bg-ink-900 px-3 py-2">
      <div className="min-w-0 flex-1">
        <p className="text-xs text-slate-500">{label}</p>
        <p className={`truncate text-sm text-slate-200 ${mono ? 'font-mono' : ''}`}>{value}</p>
      </div>
      <button
        onClick={handle}
        className="inline-flex shrink-0 items-center gap-1.5 rounded-md bg-ink-700 px-2.5 py-1.5 text-xs font-medium text-slate-200 hover:bg-ink-600"
      >
        {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
        {copied ? 'Copied' : 'Copy'}
      </button>
    </div>
  );
}
