'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import type { Order, Plan, Wallet } from '@/lib/types';
import {
  Badge,
  Button,
  Card,
  EmptyState,
  ErrorState,
  Field,
  inputClasses,
  Spinner,
  statusTone,
} from '@/components/ui';
import { useToast } from '@/components/Toast';
import { useAuth } from '@/components/AuthProvider';
import { formatUsd, formatDate, titleCase } from '@/lib/format';

// unitLabel turns a plan unit into a quantity-field label.
function unitLabel(unit?: string): string {
  switch (unit) {
    case 'gb':
      return 'Bandwidth (GB)';
    case 'hour':
      return 'Duration (hours)';
    case 'day':
      return 'Duration (days)';
    case 'ip':
      return 'Number of IPs';
    default:
      return 'Quantity';
  }
}

export default function OrdersPage() {
  const { notify } = useToast();
  const { refresh } = useAuth();
  const router = useRouter();

  const [plans, setPlans] = useState<Plan[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [wallet, setWallet] = useState<Wallet | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // form state
  const [planId, setPlanId] = useState('');
  const [quantity, setQuantity] = useState(1);
  const [submitting, setSubmitting] = useState(false);

  const loadAll = async () => {
    setError(null);
    try {
      const [pl, or, w] = await Promise.all([
        api.getPlans(),
        api.getOrders(),
        api.getWallet(),
      ]);
      setPlans(pl);
      setOrders(or);
      setWallet(w);
      if (pl.length && !planId) {
        setPlanId(pl[0].id);
        setQuantity(pl[0].min_quantity);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to load orders.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const selectedPlan = useMemo(
    () => plans.find((p) => p.id === planId) ?? null,
    [plans, planId],
  );

  const totalCents = selectedPlan ? selectedPlan.price_cents * quantity : 0;
  const insufficient = wallet != null && totalCents > wallet.balance_cents;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedPlan) return;
    if (quantity < selectedPlan.min_quantity) {
      notify(`Minimum quantity is ${selectedPlan.min_quantity}`, 'error');
      return;
    }
    setSubmitting(true);
    try {
      const res = await api.createOrder({ plan_id: planId, quantity });
      notify(
        `Order placed — ${res.proxies.length} proxy endpoint(s) provisioned.`,
        'success',
      );
      await loadAll();
      await refresh();
      router.push('/dashboard/proxies');
    } catch (err) {
      if (err instanceof ApiError && err.code === 'insufficient_funds') {
        notify('Insufficient wallet balance. Please top up.', 'error');
      } else {
        notify(err instanceof ApiError ? err.message : 'Order failed.', 'error');
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) return <Spinner label="Loading…" />;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-white">Orders</h1>

      {error && <ErrorState message={error} onRetry={loadAll} />}

      <div className="grid gap-6 lg:grid-cols-5">
        {/* New order form */}
        <Card className="lg:col-span-2">
          <h2 className="text-base font-semibold text-white">New order</h2>
          {plans.length === 0 ? (
            <p className="mt-4 text-sm text-slate-400">No plans available.</p>
          ) : (
            <form onSubmit={submit} className="mt-4 space-y-4">
              <Field label="Proxy plan" htmlFor="plan">
                <select
                  id="plan"
                  value={planId}
                  onChange={(e) => {
                    setPlanId(e.target.value);
                    const p = plans.find((x) => x.id === e.target.value);
                    if (p) setQuantity(p.min_quantity);
                  }}
                  className={inputClasses}
                >
                  {plans.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} — {formatUsd(p.price_cents)}/{p.unit}
                    </option>
                  ))}
                </select>
              </Field>

              <Field
                label={unitLabel(selectedPlan?.unit)}
                htmlFor="qty"
                hint={selectedPlan ? `Minimum ${selectedPlan.min_quantity} ${selectedPlan.unit}` : undefined}
              >
                <input
                  id="qty"
                  type="number"
                  min={selectedPlan?.min_quantity ?? 1}
                  value={quantity}
                  onChange={(e) => setQuantity(Math.max(1, parseInt(e.target.value || '1', 10)))}
                  className={inputClasses}
                />
              </Field>

              <p className="rounded-lg border border-ink-600 bg-ink-850 px-3 py-2.5 text-xs text-slate-400">
                Need a sticky IP? Residential plans support sessions up to 12h via
                the proxy username — details on the Proxies page after ordering.
              </p>

              <div className="flex items-center justify-between rounded-lg border border-ink-600 bg-ink-850 px-3 py-2.5">
                <span className="text-sm text-slate-400">Total</span>
                <span className="text-lg font-bold text-white">{formatUsd(totalCents)}</span>
              </div>

              {insufficient ? (
                <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2.5 text-sm text-amber-200">
                  Balance {formatUsd(wallet?.balance_cents)} is too low.{' '}
                  <Link href="/dashboard/billing" className="font-semibold underline">
                    Top up
                  </Link>
                </div>
              ) : null}

              <Button type="submit" fullWidth loading={submitting} disabled={insufficient}>
                Place order
              </Button>
            </form>
          )}
        </Card>

        {/* Order history */}
        <Card className="lg:col-span-3">
          <h2 className="text-base font-semibold text-white">Order history</h2>
          {orders.length === 0 ? (
            <div className="mt-4">
              <EmptyState title="No orders yet" description="Your placed orders will appear here." />
            </div>
          ) : (
            <div className="mt-4 overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="text-xs uppercase tracking-wide text-slate-500">
                  <tr>
                    <th className="pb-2 font-medium">Plan</th>
                    <th className="pb-2 font-medium">Qty</th>
                    <th className="pb-2 font-medium">Total</th>
                    <th className="pb-2 font-medium">Status</th>
                    <th className="pb-2 font-medium">Date</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-ink-700">
                  {orders.map((o) => (
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
    </div>
  );
}
