import Link from 'next/link';
import { ArrowRight, Check, Globe2, Network, Server, Smartphone } from 'lucide-react';
import type { Plan } from '@/lib/types';
import { formatUsd, titleCase } from '@/lib/format';

const typeIcon: Record<string, typeof Globe2> = {
  residential: Globe2,
  isp: Network,
  datacenter: Server,
  ipv6: Network,
  mobile: Smartphone,
};

export function PlanCard({
  plan,
  href = '/register',
  ctaLabel = 'Get started',
  popular = false,
}: {
  plan: Plan;
  href?: string;
  ctaLabel?: string;
  popular?: boolean;
}) {
  const Icon = typeIcon[plan.proxy_type] ?? Globe2;
  const features = [
    `Minimum ${plan.min_quantity} ${plan.unit}${plan.min_quantity > 1 ? 's' : ''}`,
    ['residential', 'isp', 'mobile'].includes(plan.proxy_type) ? 'Rotating & sticky sessions' : 'Per-request rotation',
    'Instant provisioning',
    'Live usage in dashboard',
  ];

  const inner = (
    <div
      className={`flex h-full flex-col rounded-[15px] p-6 ${
        popular ? 'bg-ink-800' : 'border border-ink-600 bg-ink-800'
      }`}
    >
      <div className="flex items-center justify-between">
        <div
          className={`flex h-11 w-11 items-center justify-center rounded-xl ${
            popular ? 'bg-brand-gradient text-white' : 'bg-brand-500/15 text-brand-300'
          }`}
        >
          <Icon className="h-5 w-5" />
        </div>
        {popular ? (
          <span className="rounded-full bg-brand-gradient px-3 py-1 text-[11px] font-bold uppercase tracking-wide text-white shadow-lg shadow-brand-600/30">
            Most popular
          </span>
        ) : (
          <span className="rounded-full bg-ink-700 px-2.5 py-1 text-[10px] font-semibold uppercase tracking-wide text-slate-400">
            {titleCase(plan.proxy_type)}
          </span>
        )}
      </div>

      <h3 className="mt-5 flex min-h-[3.25rem] items-start text-lg font-semibold leading-snug text-white">
        {plan.name}
      </h3>

      <div className="mt-3 flex items-baseline gap-1.5">
        <span className="text-5xl font-bold tracking-tight text-white">{formatUsd(plan.price_cents)}</span>
        <span className="text-base text-slate-500">/ {plan.unit}</span>
      </div>

      <div className="my-6 h-px w-full bg-gradient-to-r from-ink-600 to-transparent" />

      <ul className="flex-1 space-y-3.5 text-sm">
        {features.map((f) => (
          <li key={f} className="flex items-center gap-3 text-slate-200">
            <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-brand-500/15">
              <Check className="h-3 w-3 text-brand-300" />
            </span>
            {f}
          </li>
        ))}
      </ul>

      <Link
        href={href}
        className={`group/btn mt-7 inline-flex w-full items-center justify-center gap-2 rounded-xl px-4 py-3 text-sm font-semibold transition-all ${
          popular
            ? 'bg-brand-600 text-white shadow-lg shadow-brand-600/30 hover:bg-brand-500'
            : 'bg-ink-700 text-white hover:bg-ink-600'
        }`}
      >
        {ctaLabel}
        <ArrowRight className="h-4 w-4 transition-transform group-hover/btn:translate-x-0.5" />
      </Link>
    </div>
  );

  if (popular) {
    return (
      <div className="relative h-full rounded-2xl bg-gradient-to-b from-brand-400 via-brand-500 to-accent-500 p-px shadow-2xl shadow-brand-600/25 transition-transform duration-300 hover:-translate-y-1.5">
        <div className="absolute -inset-2 -z-10 rounded-3xl bg-brand-600/20 blur-2xl" />
        {inner}
      </div>
    );
  }
  return (
    <div className="h-full rounded-2xl transition-all duration-300 hover:-translate-y-1.5">{inner}</div>
  );
}
