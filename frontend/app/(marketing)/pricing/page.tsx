import type { Metadata } from 'next';
import Link from 'next/link';
import { ArrowRight, RefreshCw, ShieldCheck, Wallet, Zap } from 'lucide-react';
import { PlanCard } from '@/components/PlanCard';
import { Reveal } from '@/components/Reveal';
import { Aurora } from '@/components/Aurora';
import { fetchPlansServer } from '@/lib/server';
import { brand } from '@/lib/brand';

export const metadata: Metadata = {
  title: 'Pricing',
  description: `${brand.name} pricing — residential, ISP, datacenter, IPv6 and mobile proxies billed per GB or IP. Pay-as-you-go with no contracts or monthly minimums.`,
  alternates: { canonical: '/pricing' },
};

const includes = [
  { icon: Zap, text: 'Instant provisioning' },
  { icon: RefreshCw, text: 'Rotating & sticky sessions' },
  { icon: Wallet, text: 'Prepaid wallet — no contracts' },
  { icon: ShieldCheck, text: 'Secure dashboard & API' },
];

export default async function PricingPage() {
  const plans = await fetchPlansServer();

  return (
    <>
      {/* Header */}
      <section className="relative overflow-hidden pb-4 pt-20">
        <Aurora />
        <div className="mx-auto max-w-2xl px-4 text-center sm:px-6 lg:px-8">
          <span className="inline-flex items-center gap-2 rounded-full border border-brand-500/30 bg-brand-500/10 px-3.5 py-1.5 text-xs font-medium text-brand-200">
            Pay-as-you-go
          </span>
          <h1 className="mt-5 text-4xl font-extrabold tracking-tight text-white sm:text-5xl">
            Pricing that scales with you
          </h1>
          <p className="mt-4 text-lg text-slate-400">
            Top up your wallet and spend it across any proxy type. No contracts,
            no monthly minimums — only pay for what you use.
          </p>
        </div>
      </section>

      {/* Plans */}
      <section className="mx-auto max-w-7xl px-4 pb-8 pt-10 sm:px-6 lg:px-8">
        {plans.length === 0 ? (
          <p className="text-center text-slate-400">Pricing is loading — please refresh.</p>
        ) : (
          <div
            className={`mx-auto grid gap-6 ${
              plans.length <= 2
                ? 'max-w-3xl sm:grid-cols-2'
                : plans.length === 4
                  ? 'max-w-5xl sm:grid-cols-2 lg:grid-cols-4'
                  : 'sm:grid-cols-2 lg:grid-cols-3'
            }`}
          >
            {plans.map((plan, i) => (
              <Reveal key={plan.id} delay={(i % 4) * 70}>
                <PlanCard plan={plan} ctaLabel="Get started" popular={plan.code === 'resi_pergb'} />
              </Reveal>
            ))}
          </div>
        )}
      </section>

      {/* Included with every plan */}
      <section className="mx-auto max-w-5xl px-4 pb-16 sm:px-6 lg:px-8">
        <div className="rounded-2xl border border-ink-600 bg-ink-800/40 p-6">
          <p className="mb-5 text-center text-sm font-semibold uppercase tracking-widest text-slate-500">
            Included with every plan
          </p>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            {includes.map((item) => {
              const Icon = item.icon;
              return (
                <div key={item.text} className="flex items-center gap-2.5 text-sm text-slate-300">
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-brand-500/10 text-brand-300 ring-1 ring-brand-500/20">
                    <Icon className="h-4 w-4" />
                  </span>
                  {item.text}
                </div>
              );
            })}
          </div>
        </div>
        <p className="mt-8 text-center text-sm text-slate-500">
          Prices are per GB or IP and billed from your prepaid wallet balance.
          Need volume pricing?{' '}
          <a href={`mailto:${brand.supportEmail}`} className="text-brand-400 hover:text-brand-300">
            Contact sales
          </a>
          .
        </p>
        <div className="mt-8 flex justify-center">
          <Link
            href="/register"
            className="group inline-flex items-center gap-2 rounded-xl bg-brand-600 px-6 py-3.5 text-sm font-semibold text-white shadow-lg shadow-brand-600/30 transition-all hover:bg-brand-500"
          >
            Create your account
            <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
          </Link>
        </div>
      </section>
    </>
  );
}
