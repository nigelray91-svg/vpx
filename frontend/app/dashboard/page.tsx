'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { ArrowUpRight, Boxes, Plus, Receipt, Wallet, Zap } from 'lucide-react';
import { api } from '@/lib/api';
import type { Order, Proxy, Wallet as WalletType } from '@/lib/types';
import { useAuth } from '@/components/AuthProvider';
import { Badge, Spinner, statusTone } from '@/components/ui';
import { formatUsd, formatDate, titleCase } from '@/lib/format';

export default function DashboardOverview() {
  const { user } = useAuth();
  const [wallet, setWallet] = useState<WalletType | null>(null);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    (async () => {
      try {
        const [w, p, o] = await Promise.all([
          api.getWallet(controller.signal),
          api.getProxies(controller.signal),
          api.getOrders(controller.signal),
        ]);
        setWallet(w);
        setProxies(p);
        setOrders(o);
      } catch (err) {
        if ((err as Error)?.name === 'AbortError') return;
      } finally {
        setLoading(false);
      }
    })();
    return () => controller.abort();
  }, []);

  if (loading) return <Spinner label="Loading dashboard…" />;

  const activeProxies = proxies.filter((p) => p.status === 'active').length;
  const firstName = (user?.full_name || user?.email || '').split(/[\s@]/)[0];

  return (
    <div className="space-y-7">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-white">
            Welcome back{firstName ? `, ${firstName}` : ''}
          </h1>
          <p className="mt-1 text-sm text-slate-400">Here&apos;s what&apos;s happening with your account.</p>
        </div>
        <div className="flex gap-2">
          <Link
            href="/dashboard/billing"
            className="inline-flex items-center gap-1.5 rounded-lg border border-ink-600 bg-ink-800 px-4 py-2 text-sm font-semibold text-white transition-colors hover:border-brand-500/50"
          >
            <Wallet className="h-4 w-4" /> Top up
          </Link>
          <Link
            href="/dashboard/orders"
            className="inline-flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white shadow-lg shadow-brand-600/25 transition-colors hover:bg-brand-500"
          >
            <Plus className="h-4 w-4" /> New order
          </Link>
        </div>
      </div>

      {/* Hero wallet + stats */}
      <div className="grid gap-4 lg:grid-cols-3">
        <div className="relative overflow-hidden rounded-2xl border border-brand-500/30 bg-ink-800/60 p-6">
          <div className="absolute inset-0 -z-10 bg-brand-gradient opacity-[0.10]" />
          <div className="flex items-center gap-2 text-sm text-slate-300">
            <Wallet className="h-4 w-4 text-brand-300" /> Wallet balance
          </div>
          <p className="mt-2 text-4xl font-extrabold text-white">{formatUsd(wallet?.balance_cents)}</p>
          <Link
            href="/dashboard/billing"
            className="mt-4 inline-flex items-center gap-1 text-sm font-semibold text-brand-300 hover:text-brand-200"
          >
            Add funds <ArrowUpRight className="h-4 w-4" />
          </Link>
        </div>

        <StatCard
          icon={<Boxes className="h-5 w-5" />}
          tone="emerald"
          label="Active proxies"
          value={String(activeProxies)}
          href="/dashboard/proxies"
        />
        <StatCard
          icon={<Receipt className="h-5 w-5" />}
          tone="amber"
          label="Total orders"
          value={String(orders.length)}
          href="/dashboard/orders"
        />
      </div>

      {/* Recent orders */}
      <div className="rounded-2xl border border-ink-600 bg-ink-800/50 p-5">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-base font-semibold text-white">Recent orders</h2>
          <Link href="/dashboard/orders" className="text-sm font-medium text-brand-400 hover:text-brand-300">
            View all
          </Link>
        </div>
        {orders.length === 0 ? (
          <div className="flex flex-col items-center gap-3 py-10 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-brand-500/10 text-brand-300">
              <Zap className="h-6 w-6" />
            </div>
            <p className="text-sm text-slate-400">No orders yet.</p>
            <Link
              href="/dashboard/orders"
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white hover:bg-brand-500"
            >
              Place your first order
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="text-xs uppercase tracking-wide text-slate-500">
                <tr className="border-b border-ink-700">
                  <th className="pb-2.5 font-medium">Plan</th>
                  <th className="pb-2.5 font-medium">Qty</th>
                  <th className="pb-2.5 font-medium">Total</th>
                  <th className="pb-2.5 font-medium">Status</th>
                  <th className="pb-2.5 font-medium">Date</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-700/70">
                {orders.slice(0, 5).map((o) => (
                  <tr key={o.id} className="text-slate-300 transition-colors hover:bg-ink-700/30">
                    <td className="py-3 font-medium text-white">{o.plan_name || titleCase(o.plan_code || '—')}</td>
                    <td className="py-3">{o.quantity}</td>
                    <td className="py-3">{formatUsd(o.total_cents)}</td>
                    <td className="py-3"><Badge tone={statusTone(o.status)}>{o.status}</Badge></td>
                    <td className="py-3 text-slate-400">{formatDate(o.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

const toneStyles: Record<string, string> = {
  emerald: 'bg-emerald-500/10 text-emerald-300 ring-emerald-500/20',
  amber: 'bg-amber-500/10 text-amber-300 ring-amber-500/20',
  brand: 'bg-brand-500/10 text-brand-300 ring-brand-500/20',
};

function StatCard({
  icon,
  label,
  value,
  href,
  tone = 'brand',
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  href: string;
  tone?: 'emerald' | 'amber' | 'brand';
}) {
  return (
    <Link
      href={href}
      className="group rounded-2xl border border-ink-600 bg-ink-800/50 p-6 transition-all hover:-translate-y-0.5 hover:border-brand-500/40"
    >
      <div className="flex items-start justify-between">
        <div className={`flex h-11 w-11 items-center justify-center rounded-xl ring-1 ${toneStyles[tone]}`}>
          {icon}
        </div>
        <ArrowUpRight className="h-4 w-4 text-slate-600 transition-colors group-hover:text-brand-300" />
      </div>
      <p className="mt-4 text-3xl font-bold text-white">{value}</p>
      <p className="mt-1 text-sm text-slate-400">{label}</p>
    </Link>
  );
}
