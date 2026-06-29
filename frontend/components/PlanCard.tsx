import Link from 'next/link';
import { Check } from 'lucide-react';
import type { Plan } from '@/lib/types';
import { formatUsd, titleCase } from '@/lib/format';

const typeBlurb: Record<string, string> = {
  residential: 'Real-device IPs from millions of homes worldwide.',
  isp: 'Static residential IPs hosted on premium ISPs.',
  datacenter: 'Blazing-fast, high-bandwidth datacenter IPs.',
  ipv6: 'Cost-efficient IPv6 subnets at massive scale.',
  mobile: '4G/5G mobile IPs with the highest trust score.',
};

export function PlanCard({
  plan,
  href = '/register',
  ctaLabel = 'Get started',
}: {
  plan: Plan;
  href?: string;
  ctaLabel?: string;
}) {
  return (
    <div className="flex flex-col rounded-2xl border border-ink-600 bg-ink-800/60 p-6 transition-colors hover:border-brand-500/50">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-semibold text-white">{plan.name}</h3>
        <span className="rounded-full bg-brand-500/15 px-2.5 py-0.5 text-xs font-medium text-brand-300">
          {titleCase(plan.proxy_type)}
        </span>
      </div>
      <p className="mt-2 text-sm text-slate-400">
        {typeBlurb[plan.proxy_type] ?? 'Premium proxy access on demand.'}
      </p>
      <div className="mt-5 flex items-baseline gap-1">
        <span className="text-3xl font-bold text-white">
          {formatUsd(plan.price_cents)}
        </span>
        <span className="text-sm text-slate-400">/ {plan.unit}</span>
      </div>
      <ul className="mt-5 space-y-2 text-sm text-slate-300">
        <li className="flex items-center gap-2">
          <Check className="h-4 w-4 text-brand-400" />
          Billed per {plan.unit.toUpperCase()} · min {plan.min_quantity} {plan.unit}
          {plan.min_quantity > 1 ? 's' : ''}
        </li>
        <li className="flex items-center gap-2">
          <Check className="h-4 w-4 text-brand-400" />
          {['residential', 'isp', 'mobile'].includes(plan.proxy_type)
            ? 'Rotating & sticky sessions'
            : 'Per-request rotation'}
        </li>
        <li className="flex items-center gap-2">
          <Check className="h-4 w-4 text-brand-400" />
          Instant provisioning
        </li>
      </ul>
      <Link
        href={href}
        className="mt-6 inline-flex w-full items-center justify-center rounded-lg bg-brand-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-brand-500"
      >
        {ctaLabel}
      </Link>
    </div>
  );
}
