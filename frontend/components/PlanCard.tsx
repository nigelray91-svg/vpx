import Link from 'next/link';
import { ArrowRight, Check, Globe2, Network, Server, Smartphone } from 'lucide-react';
import type { Plan } from '@/lib/types';
import { formatUsd, titleCase } from '@/lib/format';

const typeMeta: Record<string, { blurb: string; icon: typeof Globe2 }> = {
  residential: { blurb: 'Real-device IPs from millions of homes worldwide.', icon: Globe2 },
  isp: { blurb: 'Static residential IPs hosted on premium ISPs.', icon: Network },
  datacenter: { blurb: 'Blazing-fast, high-bandwidth datacenter IPs.', icon: Server },
  ipv6: { blurb: 'Cost-efficient IPv6 subnets at massive scale.', icon: Network },
  mobile: { blurb: '4G/5G mobile IPs with the highest trust score.', icon: Smartphone },
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
  const meta = typeMeta[plan.proxy_type] ?? { blurb: 'Premium proxy access on demand.', icon: Globe2 };
  const Icon = meta.icon;
  const features = [
    `Billed per ${plan.unit.toUpperCase()} · min ${plan.min_quantity} ${plan.unit}${plan.min_quantity > 1 ? 's' : ''}`,
    ['residential', 'isp', 'mobile'].includes(plan.proxy_type) ? 'Rotating & sticky sessions' : 'Per-request rotation',
    'Instant provisioning',
  ];

  return (
    <div
      className={`group relative flex h-full flex-col overflow-hidden rounded-2xl border bg-ink-800/60 p-6 transition-all duration-300 hover:-translate-y-1 ${
        popular
          ? 'border-brand-500/60 shadow-xl shadow-brand-600/15'
          : 'border-ink-600 hover:border-brand-500/40 hover:shadow-lg hover:shadow-brand-600/10'
      }`}
    >
      {/* top accent */}
      <div
        className={`absolute inset-x-0 top-0 h-1 ${popular ? 'bg-brand-gradient' : 'bg-transparent group-hover:bg-brand-gradient/60'}`}
      />
      {popular && (
        <span className="absolute right-4 top-4 rounded-full bg-brand-gradient px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider text-white shadow-lg shadow-brand-600/30">
          Popular
        </span>
      )}

      <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-brand-500/10 text-brand-300 ring-1 ring-brand-500/20 transition-colors group-hover:bg-brand-500/20">
        <Icon className="h-5 w-5" />
      </div>

      <h3 className="mt-4 text-lg font-semibold text-white">{plan.name}</h3>
      <span className="mt-1 inline-flex w-fit rounded-full bg-ink-700 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-400">
        {titleCase(plan.proxy_type)}
      </span>
      <p className="mt-3 text-sm text-slate-400">{meta.blurb}</p>

      <div className="mt-5 flex items-baseline gap-1">
        <span className="bg-gradient-to-br from-white to-slate-400 bg-clip-text text-4xl font-extrabold text-transparent">
          {formatUsd(plan.price_cents)}
        </span>
        <span className="text-sm text-slate-500">/ {plan.unit}</span>
      </div>

      <ul className="mb-6 mt-5 flex-1 space-y-2.5 text-sm text-slate-300">
        {features.map((f) => (
          <li key={f} className="flex items-start gap-2.5">
            <Check className="mt-0.5 h-4 w-4 shrink-0 text-brand-400" />
            <span>{f}</span>
          </li>
        ))}
      </ul>

      <Link
        href={href}
        className={`inline-flex w-full items-center justify-center gap-1.5 rounded-lg px-4 py-2.5 text-sm font-semibold transition-all ${
          popular
            ? 'bg-brand-600 text-white shadow-lg shadow-brand-600/25 hover:bg-brand-500'
            : 'border border-ink-600 bg-ink-700/50 text-white hover:border-brand-500/50 hover:bg-ink-700'
        }`}
      >
        {ctaLabel}
        <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
      </Link>
    </div>
  );
}
