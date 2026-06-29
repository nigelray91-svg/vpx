import Link from 'next/link';
import {
  ArrowRight,
  Gauge,
  Globe2,
  Lock,
  Network,
  RefreshCw,
  Server,
  Smartphone,
  Timer,
  Zap,
} from 'lucide-react';
import { PlanCard } from '@/components/PlanCard';
import { fetchPlansServer } from '@/lib/server';

const siteName = process.env.NEXT_PUBLIC_SITE_NAME ?? 'VaultProxies Reseller';

const proxyTypes = [
  {
    icon: Globe2,
    title: 'Residential',
    desc: 'Real-device IPs sourced ethically from millions of homes.',
  },
  {
    icon: Network,
    title: 'ISP',
    desc: 'Static residential IPs on premium ISPs — speed meets trust.',
  },
  {
    icon: Server,
    title: 'Datacenter',
    desc: 'High-bandwidth, low-latency IPs for heavy workloads.',
  },
  {
    icon: Network,
    title: 'IPv6',
    desc: 'Massive IPv6 subnets at the lowest cost per IP.',
  },
  {
    icon: Smartphone,
    title: 'Mobile',
    desc: '4G/5G carrier IPs with the highest trust score available.',
  },
];

const features = [
  {
    icon: Zap,
    title: 'Instant provisioning',
    desc: 'Order and receive live proxy credentials in seconds — no waiting.',
  },
  {
    icon: RefreshCw,
    title: 'Per-request rotation',
    desc: 'Rotate IP on every request for maximum coverage and freshness.',
  },
  {
    icon: Timer,
    title: 'Sticky sessions',
    desc: 'Hold the same IP for up to 60 minutes for stateful workflows.',
  },
  {
    icon: Gauge,
    title: 'Usage analytics',
    desc: 'Track bandwidth per proxy with real-time usage reporting.',
  },
  {
    icon: Lock,
    title: 'Secure by default',
    desc: 'HttpOnly session cookies, CSRF protection, and encrypted transport.',
  },
  {
    icon: Globe2,
    title: '32M+ IP pool',
    desc: 'Global coverage across every continent and major geography.',
  },
];

export default async function LandingPage() {
  const plans = await fetchPlansServer();
  const previewPlans = plans.slice(0, 3);

  return (
    <>
      {/* Hero */}
      <section className="relative overflow-hidden bg-grid-fade">
        <div className="mx-auto max-w-7xl px-4 py-24 sm:px-6 lg:px-8 lg:py-32">
          <div className="mx-auto max-w-3xl text-center">
            <span className="inline-flex items-center gap-2 rounded-full border border-brand-500/30 bg-brand-500/10 px-4 py-1.5 text-sm font-medium text-brand-300">
              <Zap className="h-4 w-4" />
              32M+ premium IPs · instant provisioning
            </span>
            <h1 className="mt-6 text-4xl font-extrabold tracking-tight text-white sm:text-6xl">
              The proxy network built for{' '}
              <span className="bg-gradient-to-r from-brand-400 to-indigo-400 bg-clip-text text-transparent">
                scale and trust
              </span>
            </h1>
            <p className="mx-auto mt-6 max-w-2xl text-lg text-slate-300">
              Residential, ISP, datacenter, IPv6 and mobile proxies on a single
              dashboard. Per-request rotation, sticky sessions up to 60 minutes,
              and a wallet that bills to the cent.
            </p>
            <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
              <Link
                href="/register"
                className="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-6 py-3 text-base font-semibold text-white shadow-lg shadow-brand-600/30 transition-colors hover:bg-brand-500"
              >
                Start now <ArrowRight className="h-4 w-4" />
              </Link>
              <Link
                href="/pricing"
                className="inline-flex items-center gap-2 rounded-lg border border-ink-600 px-6 py-3 text-base font-semibold text-slate-200 transition-colors hover:bg-ink-800"
              >
                View pricing
              </Link>
            </div>
            <dl className="mx-auto mt-14 grid max-w-2xl grid-cols-3 gap-6">
              {[
                ['32M+', 'IP addresses'],
                ['99.9%', 'Uptime'],
                ['60 min', 'Sticky sessions'],
              ].map(([stat, label]) => (
                <div key={label}>
                  <dt className="text-2xl font-bold text-white sm:text-3xl">
                    {stat}
                  </dt>
                  <dd className="mt-1 text-sm text-slate-400">{label}</dd>
                </div>
              ))}
            </dl>
          </div>
        </div>
      </section>

      {/* Proxy types / network */}
      <section id="network" className="border-t border-ink-700/60 py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="max-w-2xl">
            <h2 className="text-3xl font-bold tracking-tight text-white">
              Every proxy type, one platform
            </h2>
            <p className="mt-3 text-slate-400">
              Pick the right network for the job — and switch any time from your
              dashboard.
            </p>
          </div>
          <div className="mt-10 grid gap-5 sm:grid-cols-2 lg:grid-cols-5">
            {proxyTypes.map((t) => (
              <div
                key={t.title}
                className="rounded-xl border border-ink-600 bg-ink-800/60 p-5"
              >
                <t.icon className="h-7 w-7 text-brand-400" />
                <h3 className="mt-4 font-semibold text-white">{t.title}</h3>
                <p className="mt-1.5 text-sm text-slate-400">{t.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="border-t border-ink-700/60 py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="max-w-2xl">
            <h2 className="text-3xl font-bold tracking-tight text-white">
              Built for serious operators
            </h2>
            <p className="mt-3 text-slate-400">
              Everything you need to provision, rotate, and monitor proxies at
              scale.
            </p>
          </div>
          <div className="mt-10 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((f) => (
              <div
                key={f.title}
                className="rounded-xl border border-ink-600 bg-ink-800/60 p-6"
              >
                <f.icon className="h-7 w-7 text-brand-400" />
                <h3 className="mt-4 font-semibold text-white">{f.title}</h3>
                <p className="mt-1.5 text-sm text-slate-400">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Pricing preview */}
      <section className="border-t border-ink-700/60 py-20">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div className="max-w-2xl">
              <h2 className="text-3xl font-bold tracking-tight text-white">
                Simple, usage-based pricing
              </h2>
              <p className="mt-3 text-slate-400">
                Pay for exactly what you use. Top up your wallet and order in
                seconds.
              </p>
            </div>
            <Link
              href="/pricing"
              className="inline-flex items-center gap-1 text-sm font-semibold text-brand-300 hover:text-brand-200"
            >
              See full catalog <ArrowRight className="h-4 w-4" />
            </Link>
          </div>

          {previewPlans.length > 0 ? (
            <div className="mt-10 grid gap-6 md:grid-cols-3">
              {previewPlans.map((plan) => (
                <PlanCard key={plan.id} plan={plan} />
              ))}
            </div>
          ) : (
            <div className="mt-10 rounded-xl border border-dashed border-ink-600 bg-ink-800/40 p-10 text-center text-slate-400">
              Pricing is loading. Visit the{' '}
              <Link href="/pricing" className="text-brand-300 underline">
                pricing page
              </Link>{' '}
              for the full catalog.
            </div>
          )}
        </div>
      </section>

      {/* CTA */}
      <section className="border-t border-ink-700/60 py-20">
        <div className="mx-auto max-w-5xl px-4 sm:px-6 lg:px-8">
          <div className="rounded-3xl border border-brand-500/30 bg-gradient-to-br from-brand-600/20 to-indigo-600/10 p-10 text-center sm:p-14">
            <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
              Ready to deploy your first proxy?
            </h2>
            <p className="mx-auto mt-3 max-w-xl text-slate-300">
              Create an account, top up your wallet, and provision proxies in
              under a minute with {siteName}.
            </p>
            <Link
              href="/register"
              className="mt-8 inline-flex items-center gap-2 rounded-lg bg-brand-600 px-6 py-3 text-base font-semibold text-white shadow-lg shadow-brand-600/30 transition-colors hover:bg-brand-500"
            >
              Create free account <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </div>
      </section>
    </>
  );
}
