'use client';

import { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { Bitcoin, CreditCard } from 'lucide-react';
import { api, ApiError } from '@/lib/api';
import type { LedgerEntry, Payment, Wallet } from '@/lib/types';
import { useConfig } from '@/lib/useConfig';
import { useToast } from '@/components/Toast';
import {
  Badge,
  Button,
  Card,
  EmptyState,
  Field,
  inputClasses,
  Spinner,
  statusTone,
} from '@/components/ui';
import { formatUsd, formatDate, titleCase } from '@/lib/format';

type Provider = 'stripe' | 'nowpayments';

function BillingInner() {
  const params = useSearchParams();
  const { config } = useConfig();
  const { notify } = useToast();

  const [wallet, setWallet] = useState<Wallet | null>(null);
  const [ledger, setLedger] = useState<LedgerEntry[]>([]);
  const [payments, setPayments] = useState<Payment[]>([]);
  const [loading, setLoading] = useState(true);

  const [amountDollars, setAmountDollars] = useState('10');
  const [provider, setProvider] = useState<Provider>('stripe');
  const [submitting, setSubmitting] = useState(false);

  const load = async () => {
    try {
      const [w, l, p] = await Promise.all([
        api.getWallet(),
        api.getLedger(),
        api.getPayments(),
      ]);
      setWallet(w);
      setLedger(l);
      setPayments(p);
    } catch {
      /* handled by empty states */
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  useEffect(() => {
    const status = params.get('status');
    if (status === 'success') {
      notify('Payment received — your balance will update shortly.', 'success');
    } else if (status === 'cancelled') {
      notify('Payment cancelled.', 'info');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params]);

  // Default the provider to the first enabled option.
  useEffect(() => {
    if (!config) return;
    if (!config.stripe_enabled && config.crypto_enabled) setProvider('nowpayments');
  }, [config]);

  const minCents = config?.min_topup_cents ?? 500;

  const topup = async (e: React.FormEvent) => {
    e.preventDefault();
    const cents = Math.round(parseFloat(amountDollars || '0') * 100);
    if (!Number.isFinite(cents) || cents < minCents) {
      notify(`Minimum top-up is ${formatUsd(minCents)}.`, 'error');
      return;
    }
    setSubmitting(true);
    try {
      const res = await api.topup({ amount_cents: cents, provider });
      // Redirect to the hosted checkout / invoice page.
      window.location.href = res.checkout_url;
    } catch (err) {
      notify(err instanceof ApiError ? err.message : 'Could not start payment.', 'error');
      setSubmitting(false);
    }
  };

  if (loading) return <Spinner label="Loading billing…" />;

  const stripeEnabled = config?.stripe_enabled ?? false;
  const cryptoEnabled = config?.crypto_enabled ?? false;
  const noProviders = !stripeEnabled && !cryptoEnabled;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-white">Billing</h1>

      <div className="grid gap-6 lg:grid-cols-5">
        <Card className="lg:col-span-2">
          <p className="text-sm text-slate-400">Wallet balance</p>
          <p className="mt-1 text-3xl font-bold text-white">{formatUsd(wallet?.balance_cents)}</p>

          <hr className="my-5 border-ink-700" />

          <h2 className="text-base font-semibold text-white">Add funds</h2>
          {noProviders ? (
            <p className="mt-3 text-sm text-slate-400">
              No payment providers are configured.
            </p>
          ) : (
            <form onSubmit={topup} className="mt-4 space-y-4">
              <Field label="Amount (USD)" htmlFor="amount" hint={`Minimum ${formatUsd(minCents)}`}>
                <input
                  id="amount"
                  type="number"
                  min={minCents / 100}
                  step="1"
                  value={amountDollars}
                  onChange={(e) => setAmountDollars(e.target.value)}
                  className={inputClasses}
                />
              </Field>

              <div className="grid grid-cols-2 gap-3">
                {stripeEnabled && (
                  <ProviderOption
                    active={provider === 'stripe'}
                    onClick={() => setProvider('stripe')}
                    icon={<CreditCard className="h-5 w-5" />}
                    label="Card"
                  />
                )}
                {cryptoEnabled && (
                  <ProviderOption
                    active={provider === 'nowpayments'}
                    onClick={() => setProvider('nowpayments')}
                    icon={<Bitcoin className="h-5 w-5" />}
                    label="Crypto"
                  />
                )}
              </div>

              <Button type="submit" fullWidth loading={submitting}>
                Continue to payment
              </Button>
            </form>
          )}
        </Card>

        <div className="space-y-6 lg:col-span-3">
          <Card>
            <h2 className="text-base font-semibold text-white">Transaction history</h2>
            {ledger.length === 0 ? (
              <div className="mt-4">
                <EmptyState title="No transactions yet" />
              </div>
            ) : (
              <div className="mt-4 overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead className="text-xs uppercase tracking-wide text-slate-500">
                    <tr>
                      <th className="pb-2 font-medium">Type</th>
                      <th className="pb-2 font-medium">Amount</th>
                      <th className="pb-2 font-medium">Balance</th>
                      <th className="pb-2 font-medium">Date</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-ink-700">
                    {ledger.map((e) => (
                      <tr key={e.id} className="text-slate-300">
                        <td className="py-2.5">{titleCase(e.kind)}</td>
                        <td className={`py-2.5 font-medium ${e.amount_cents >= 0 ? 'text-emerald-400' : 'text-slate-200'}`}>
                          {e.amount_cents >= 0 ? '+' : ''}
                          {formatUsd(e.amount_cents)}
                        </td>
                        <td className="py-2.5 text-slate-400">{formatUsd(e.balance_after)}</td>
                        <td className="py-2.5 text-slate-400">{formatDate(e.created_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>

          <Card>
            <h2 className="text-base font-semibold text-white">Payments</h2>
            {payments.length === 0 ? (
              <div className="mt-4">
                <EmptyState title="No payments yet" />
              </div>
            ) : (
              <div className="mt-4 overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead className="text-xs uppercase tracking-wide text-slate-500">
                    <tr>
                      <th className="pb-2 font-medium">Provider</th>
                      <th className="pb-2 font-medium">Amount</th>
                      <th className="pb-2 font-medium">Status</th>
                      <th className="pb-2 font-medium">Date</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-ink-700">
                    {payments.map((p) => (
                      <tr key={p.id} className="text-slate-300">
                        <td className="py-2.5">{titleCase(p.provider)}</td>
                        <td className="py-2.5">{formatUsd(p.amount_cents)}</td>
                        <td className="py-2.5">
                          <Badge tone={statusTone(p.status)}>{p.status}</Badge>
                        </td>
                        <td className="py-2.5 text-slate-400">{formatDate(p.created_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </div>
      </div>
    </div>
  );
}

function ProviderOption({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex items-center justify-center gap-2 rounded-lg border px-4 py-3 text-sm font-semibold transition-colors ${
        active
          ? 'border-brand-500 bg-brand-600/15 text-white'
          : 'border-ink-600 bg-ink-800 text-slate-300 hover:border-ink-500'
      }`}
    >
      {icon}
      {label}
    </button>
  );
}

export default function BillingPage() {
  return (
    <Suspense fallback={<Spinner label="Loading billing…" />}>
      <BillingInner />
    </Suspense>
  );
}
