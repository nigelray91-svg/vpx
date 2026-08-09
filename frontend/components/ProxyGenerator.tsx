'use client';

import { useEffect, useState } from 'react';
import { Check, Copy, Download, Info, Wand2 } from 'lucide-react';
import { api, ApiError } from '@/lib/api';
import { OUTPUT_FORMATS } from '@/lib/types';
import type {
  GenerateResponse,
  LocationCountry,
  OutputFormat,
  Proxy,
  Rotation,
} from '@/lib/types';
import { useToast } from '@/components/Toast';
import { Badge } from '@/components/ui';

// Plans the upstream documents as having no geo targeting.
const NO_GEO = new Set(['ipv6_pergb', 'resi_unlim_budget']);
// Sticky session caps per plan, in seconds (mirrors the backend clamp).
const SESSION_CAPS: Record<string, number> = {
  resi_pergb: 21600,
  shared_isp: 86400,
  resi_unlim_budget: 86400,
  mobile_pergb: 7200,
};

const selectClasses =
  'w-full rounded-lg border border-ink-600 bg-ink-900 px-3 py-2 text-sm text-slate-100 focus:border-brand-500 focus:outline-none';

export function ProxyGenerator({ proxy }: { proxy: Proxy }) {
  const { notify } = useToast();
  // `pool` carries the upstream category key the service was ordered against.
  const planKey = proxy.pool;
  const geoSupported = planKey ? !NO_GEO.has(planKey) : true;

  const [mode, setMode] = useState<Rotation>('rotating');
  const [count, setCount] = useState(1);
  const [protocol, setProtocol] = useState<'HTTP' | 'SOCKS5'>('HTTP');
  const [format, setFormat] = useState<OutputFormat>('ip:port:user:pass');
  const [country, setCountry] = useState('');
  const [city, setCity] = useState('');
  const [sessionSeconds, setSessionSeconds] = useState(600);

  const [countries, setCountries] = useState<LocationCountry[] | null>(null);
  const [result, setResult] = useState<GenerateResponse | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!planKey || !geoSupported) return;
    const ctrl = new AbortController();
    api
      .getLocations(planKey, undefined, ctrl.signal)
      .then(setCountries)
      .catch(() => setCountries([]));
    return () => ctrl.abort();
  }, [planKey, geoSupported]);

  const cities =
    countries
      ?.find((c) => c.code === country)
      ?.states?.flatMap((s) => s.cities ?? []) ?? [];

  const cap = SESSION_CAPS[planKey ?? ''] ?? 86400;

  const generate = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await api.generateProxies(proxy.id, {
        mode,
        count,
        protocol,
        format,
        country: country || undefined,
        city: city || undefined,
        session_seconds: mode === 'sticky' ? sessionSeconds : undefined,
      });
      setResult(res);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : 'Could not generate proxies.',
      );
    } finally {
      setBusy(false);
    }
  };

  const copyAll = async () => {
    if (!result) return;
    try {
      await navigator.clipboard.writeText(result.lines.join('\n'));
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
      notify(`${result.lines.length} line(s) copied`, 'success');
    } catch {
      notify('Could not copy to clipboard', 'error');
    }
  };

  const download = () => {
    if (!result) return;
    const blob = new Blob([result.lines.join('\n')], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `proxies-${proxy.id.slice(0, 8)}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="mt-4 rounded-lg border border-ink-600 bg-ink-850 p-4">
      <div className="flex items-center gap-2">
        <Wand2 className="h-4 w-4 text-brand-400" />
        <h4 className="text-sm font-semibold text-white">Generate proxy list</h4>
      </div>
      <p className="mt-1 text-xs text-slate-500">
        Builds ready-to-use lines with the correct gateway for this plan.
        Generating is free — only traffic consumes bandwidth, so you can
        regenerate as often as you like.
      </p>

      <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <label className="block">
          <span className="mb-1 block text-xs text-slate-400">Mode</span>
          <select
            value={mode}
            onChange={(e) => setMode(e.target.value as Rotation)}
            className={selectClasses}
          >
            <option value="rotating">Rotating (new IP per request)</option>
            <option value="sticky">Sticky (keep the same IP)</option>
          </select>
        </label>

        <label className="block">
          <span className="mb-1 block text-xs text-slate-400">Lines</span>
          <input
            type="number"
            min={1}
            max={10000}
            value={count}
            onChange={(e) =>
              setCount(Math.max(1, Math.min(10000, Number(e.target.value) || 1)))
            }
            className={selectClasses}
          />
        </label>

        <label className="block">
          <span className="mb-1 block text-xs text-slate-400">Protocol</span>
          <select
            value={protocol}
            onChange={(e) => setProtocol(e.target.value as 'HTTP' | 'SOCKS5')}
            className={selectClasses}
          >
            <option value="HTTP">HTTP</option>
            <option value="SOCKS5">SOCKS5</option>
          </select>
        </label>

        <label className="block lg:col-span-2">
          <span className="mb-1 block text-xs text-slate-400">Format</span>
          <select
            value={format}
            onChange={(e) => setFormat(e.target.value as OutputFormat)}
            className={`${selectClasses} font-mono`}
          >
            {OUTPUT_FORMATS.map((f) => (
              <option key={f} value={f}>
                {f}
              </option>
            ))}
          </select>
        </label>

        {mode === 'sticky' && (
          <label className="block">
            <span className="mb-1 block text-xs text-slate-400">
              Session (seconds, max {cap.toLocaleString()})
            </span>
            <input
              type="number"
              min={60}
              max={cap}
              value={sessionSeconds}
              onChange={(e) =>
                setSessionSeconds(
                  Math.max(60, Math.min(cap, Number(e.target.value) || 600)),
                )
              }
              className={selectClasses}
            />
          </label>
        )}

        {geoSupported && countries && countries.length > 0 && (
          <>
            <label className="block">
              <span className="mb-1 block text-xs text-slate-400">Country</span>
              <select
                value={country}
                onChange={(e) => {
                  setCountry(e.target.value);
                  setCity('');
                }}
                className={selectClasses}
              >
                <option value="">Any</option>
                {countries.map((c) => (
                  <option key={c.code} value={c.code}>
                    {c.name}
                  </option>
                ))}
              </select>
            </label>

            {cities.length > 0 && (
              <label className="block">
                <span className="mb-1 block text-xs text-slate-400">City</span>
                <select
                  value={city}
                  onChange={(e) => setCity(e.target.value)}
                  className={selectClasses}
                >
                  <option value="">Any</option>
                  {cities.map((c) => (
                    <option key={c.name} value={c.name}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </label>
            )}
          </>
        )}
      </div>

      {mode === 'rotating' && count > 1 && (
        <p className="mt-3 flex items-start gap-2 rounded-lg border border-amber-500/25 bg-amber-500/5 px-3 py-2 text-xs text-amber-200/90">
          <Info className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span>
            Rotating credentials give a fresh IP on every request, so all{' '}
            {count} lines will be identical — useful for tools that want a list,
            but switch to <strong>sticky</strong> if you need distinct
            concurrent session IPs.
          </span>
        </p>
      )}

      <button
        onClick={generate}
        disabled={busy}
        className="mt-4 inline-flex items-center gap-2 rounded-lg bg-brand-500 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-400 disabled:opacity-50"
      >
        <Wand2 className="h-4 w-4" />
        {busy ? 'Generating…' : 'Generate'}
      </button>

      {error && <p className="mt-3 text-sm text-rose-400">{error}</p>}

      {result && (
        <div className="mt-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <Badge tone="info">{result.mode}</Badge>
              <span className="text-xs text-slate-500">
                {result.lines.length} line{result.lines.length === 1 ? '' : 's'}
                {result.session_seconds
                  ? ` · ${result.session_seconds}s sessions`
                  : ''}
              </span>
            </div>
            <div className="flex gap-2">
              <button
                onClick={copyAll}
                className="inline-flex items-center gap-1.5 rounded-md bg-ink-700 px-2.5 py-1.5 text-xs font-medium text-slate-200 hover:bg-ink-600"
              >
                {copied ? (
                  <Check className="h-3.5 w-3.5 text-emerald-400" />
                ) : (
                  <Copy className="h-3.5 w-3.5" />
                )}
                {copied ? 'Copied' : 'Copy all'}
              </button>
              <button
                onClick={download}
                className="inline-flex items-center gap-1.5 rounded-md bg-ink-700 px-2.5 py-1.5 text-xs font-medium text-slate-200 hover:bg-ink-600"
              >
                <Download className="h-3.5 w-3.5" />
                .txt
              </button>
            </div>
          </div>

          {result.note && (
            <p className="mt-2 text-xs text-slate-500">{result.note}</p>
          )}

          <pre className="mt-2 max-h-64 overflow-auto rounded-lg border border-ink-600 bg-ink-900 p-3 font-mono text-xs leading-relaxed text-slate-200">
            {result.lines.join('\n')}
          </pre>
        </div>
      )}
    </div>
  );
}
