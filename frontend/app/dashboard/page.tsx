'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { Boxes, CreditCard, Plus, Wallet } from 'lucide-react';
import { api } from '@/lib/api';
import type { Order, Proxy, Wallet as WalletType } from '@/lib/types';
import { Badge, Card, LinkButton, Spinner, statusTone } from '@/components/ui';
import { formatUsd, formatDate, titleCase } from '@/lib/format';

export default function DashboardOverview() {
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

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold text-white">Overview</h1>
        <LinkButton href="/dashboard/orders">
          <Plus className="h-4 w-4" /> New order
        </LinkButton>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          icon={<Wallet className="h-5 w-5 text-brand-400" />}
          label="Wallet balance"
          value={formatUsd(wallet?.balance_cents)}
          href="/dashboard/billing"
        />
        <StatCard
          icon={<Boxes className="h-5 w-5 text-emerald-400" />}
          label="Active proxies"
          value={String(activeProxies)}
          href="/dashboard/proxies"
        />
        <StatCard
          icon={<CreditCard className="h-5 w-5 text-amber-400" />}
          label="Total orders"
          value={String(orders.length)}
          href="/dashboard/orders"
        />
      </div>

      <Card>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-base font-semibold text-white">Recent orders</h2>
          <Link href="/dashboard/orders" className="text-sm text-brand-400 hover:text-brand-300">
            View all
          </Link>
        </div>
        {orders.length === 0 ? (
          <p className="py-6 text-center text-sm text-slate-400">
            No orders yet.{' '}
            <Link href="/dashboard/orders" className="text-brand-400">Place your first order.</Link>
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="text-xs uppercase tracking-wide text-slate-500">
                <tr>
                  <th className="pb-2 font-medium">Type</th>
                  <th className="pb-2 font-medium">Qty</th>
                  <th className="pb-2 font-medium">Total</th>
                  <th className="pb-2 font-medium">Status</th>
                  <th className="pb-2 font-medium">Date</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink-700">
                {orders.slice(0, 5).map((o) => (
                  <tr key={o.id} className="text-slate-300">
                    <td className="py-2.5">{titleCase(o.plan_name || o.plan_code || '—')}</td>
                    <td className="py-2.5">{o.quantity}</td>
                    <td className="py-2.5">{formatUsd(o.total_cents)}</td>
                    <td className="py-2.5">
                      <Badge tone={statusTone(o.status)}>{o.status}</Badge>
                    </td>
                    <td className="py-2.5 text-slate-400">{formatDate(o.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}

function StatCard({
  icon,
  label,
  value,
  href,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  href: string;
}) {
  return (
    <Link href={href}>
      <Card className="transition-colors hover:border-brand-500/50">
        <div className="flex items-center gap-3">
          <div className="rounded-lg bg-ink-700 p-2">{icon}</div>
          <div>
            <p className="text-xs text-slate-400">{label}</p>
            <p className="text-xl font-bold text-white">{value}</p>
          </div>
        </div>
      </Card>
    </Link>
  );
}
