'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import type { Plan } from '@/lib/types';
import { PlanCard } from '@/components/PlanCard';
import { Spinner, ErrorState } from '@/components/ui';
import { useAuth } from '@/components/AuthProvider';

export default function PricingPage() {
  const { user } = useAuth();
  const [plans, setPlans] = useState<Plan[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    api
      .getPlans(controller.signal)
      .then(setPlans)
      .catch((err) => {
        if ((err as Error)?.name !== 'AbortError') {
          setError('Could not load pricing.');
        }
      });
    return () => controller.abort();
  }, []);

  const cta = user ? '/dashboard/orders' : '/register';

  return (
    <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-2xl text-center">
        <h1 className="text-4xl font-bold tracking-tight text-white">
          Simple, pay-as-you-go pricing
        </h1>
        <p className="mt-4 text-lg text-slate-400">
          Top up your wallet and spend it across any proxy type. No contracts, no
          monthly minimums — only pay for what you use.
        </p>
      </div>

      <div className="mt-12">
        {error ? (
          <ErrorState message={error} />
        ) : !plans ? (
          <Spinner label="Loading plans…" />
        ) : (
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {plans.map((plan) => (
              <PlanCard
                key={plan.id}
                plan={plan}
                href={cta}
                ctaLabel={user ? 'Order now' : 'Get started'}
              />
            ))}
          </div>
        )}
      </div>

      <p className="mt-10 text-center text-sm text-slate-500">
        Prices are per GB or IP and billed from your prepaid wallet balance.
        Need volume pricing? <span className="text-brand-400">Contact sales.</span>
      </p>
    </div>
  );
}
