import Link from 'next/link';
import { Check } from 'lucide-react';
import type { Plan } from '@/lib/types';
import { formatUsd, titleCase } from '@/lib/format';

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
  const features = [
    `Minimum ${plan.min_quantity} ${plan.unit}${plan.min_quantity > 1 ? 's' : ''}`,
    ['residential', 'isp', 'mobile'].includes(plan.proxy_type) ? 'Rotating & sticky sessions' : 'Per-request rotation',
    'Instant provisioning',
    'Live usage dashboard',
  ];

  return (
    <div
      className={`flex h-full flex-col rounded-xl border bg-ink-800 p-5 ${
        popular ? 'border-brand-500/60 ring-1 ring-brand-500/30' : 'border-ink-600'
      }`}
    >
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold uppercase tracking-wide text-slate-400">
          {titleCase(plan.proxy_type)}
        </span>
        {popular && (
          <span className="rounded-full bg-brand-500/15 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide text-brand-300">
            Popular
          </span>
        )}
      </div>

      <h3 className="mt-1 text-base font-semibold text-white">{plan.name}</h3>

      <div className="mt-3 flex items-baseline gap-1">
        <span className="text-3xl font-bold text-white">{formatUsd(plan.price_cents)}</span>
        <span className="text-sm text-slate-500">/ {plan.unit}</span>
      </div>

      <ul className="mt-4 flex-1 space-y-2 text-sm text-slate-300">
        {features.map((f) => (
          <li key={f} className="flex items-center gap-2">
            <Check className="h-4 w-4 shrink-0 text-brand-400" />
            {f}
          </li>
        ))}
      </ul>

      <Link
        href={href}
        className={`mt-5 inline-flex w-full items-center justify-center rounded-lg px-4 py-2 text-sm font-semibold transition-colors ${
          popular ? 'bg-brand-600 text-white hover:bg-brand-500' : 'bg-ink-700 text-white hover:bg-ink-600'
        }`}
      >
        {ctaLabel}
      </Link>
    </div>
  );
}
