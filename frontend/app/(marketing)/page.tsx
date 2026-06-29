import Link from 'next/link';
import {
  ArrowRight,
  Check,
  Copy,
  Gauge,
  Globe2,
  Lock,
  Network,
  RefreshCw,
  Server,
  Shield,
  Smartphone,
  Sparkles,
  Terminal,
  Timer,
  Wallet,
  Zap,
} from 'lucide-react';
import type { Metadata } from 'next';
import { PlanCard } from '@/components/PlanCard';
import { fetchPlansServer } from '@/lib/server';
import { brand } from '@/lib/brand';
import { Reveal } from '@/components/Reveal';
import { Counter } from '@/components/Counter';
import { Aurora } from '@/components/Aurora';
import { Marquee } from '@/components/Marquee';
import { Faq } from '@/components/Faq';
import { JsonLd } from '@/components/JsonLd';

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3000';

export const metadata: Metadata = {
  title: {
    absolute: `${brand.name} — Premium Residential, ISP, Datacenter & Mobile Proxies`,
  },
  description: `${brand.tagline} ${brand.ipCount} IPs, 195+ countries, instant provisioning, rotating & sticky sessions. Pay-as-you-go with card or crypto.`,
  alternates: { canonical: '/' },
};

const proxyTypes = [
  { icon: Globe2, title: 'Residential', tag: '32M+ IPs', desc: 'Real-device IPs sourced ethically from millions of homes worldwide.' },
  { icon: Network, title: 'ISP / Static', tag: 'Premium', desc: 'Static residential IPs on premium ISPs — datacenter speed, residential trust.' },
  { icon: Server, title: 'Datacenter', tag: 'Fastest', desc: 'High-bandwidth, low-latency IPs for heavy, high-volume workloads.' },
  { icon: Network, title: 'IPv6', tag: 'Scale', desc: 'Massive IPv6 subnets at the lowest cost per IP for huge scale.' },
  { icon: Smartphone, title: 'Mobile 4G/5G', tag: 'Highest trust', desc: 'Carrier-grade mobile IPs with the highest trust score available.' },
];

const features = [
  { icon: Zap, title: 'Instant provisioning', desc: 'Order and receive live proxy credentials in seconds — no waiting, no tickets.' },
  { icon: RefreshCw, title: 'Per-request rotation', desc: 'Rotate IP on every request for maximum coverage and freshness.' },
  { icon: Timer, title: 'Sticky sessions', desc: 'Hold the same IP for up to 60 minutes for stateful, multi-step workflows.' },
  { icon: Gauge, title: 'Real-time usage', desc: 'Track bandwidth per proxy with live usage reporting in your dashboard.' },
  { icon: Lock, title: 'Secure by default', desc: 'HttpOnly session cookies, CSRF protection, rotating tokens, encrypted transport.' },
  { icon: Wallet, title: 'Pay-as-you-go wallet', desc: 'Top up with cards or crypto and spend to the cent across any proxy type.' },
];

const steps = [
  { n: '01', title: 'Create your account', desc: 'Sign up in seconds with email or Google — protected by managed bot defense.' },
  { n: '02', title: 'Top up your wallet', desc: 'Add funds with a card (Stripe) or crypto (NOWPayments). No contracts.' },
  { n: '03', title: 'Provision & connect', desc: 'Pick a proxy type, generate credentials, and start routing traffic instantly.' },
];

const faqs = [
  { q: 'What types of proxies can I get?', a: 'Residential, ISP/static, datacenter, IPv6 and mobile 4G/5G — all from a single dashboard, switchable per order.' },
  { q: 'How does billing work?', a: 'It is fully prepaid. Top up your wallet with a card or cryptocurrency, then spend it to the cent. You only pay for what you use, with no monthly minimums.' },
  { q: 'Can I rotate IPs or keep a sticky session?', a: 'Both. Choose per-request rotation for maximum freshness, or sticky sessions that hold the same IP for up to 60 minutes for stateful workflows.' },
  { q: 'How fast is provisioning?', a: 'Instant. Once you place an order, live proxy credentials are generated and shown in your dashboard within seconds.' },
  { q: 'Is my data secure?', a: 'Yes. Passwords are hashed with bcrypt, sessions use httpOnly cookies with CSRF protection and rotating refresh tokens, and all traffic is encrypted.' },
];

const tools = ['Python', 'Scrapy', 'Playwright', 'Puppeteer', 'Selenium', 'cURL', 'Node.js', 'Go', 'Postman', 'Octoparse'];

export default async function LandingPage() {
  const allPlans = await fetchPlansServer();
  // Feature a diverse spread across proxy types (incl. mobile & datacenter/GB)
  // rather than just the first few by sort order.
  const featuredCodes = ['resi-rotating', 'isp-static', 'dc-gb', 'ipv6-pool', 'mobile-4g'];
  const byCode = new Map(allPlans.map((p) => [p.code, p]));
  const plans = featuredCodes.map((c) => byCode.get(c)).filter(Boolean).slice(0, 6) as typeof allPlans;
  const previewPlans = plans.length ? plans : allPlans.slice(0, 3);

  const jsonLd = [
    {
      '@context': 'https://schema.org',
      '@type': 'Organization',
      name: brand.name,
      url: siteUrl,
      description: brand.tagline,
      email: brand.supportEmail,
    },
    {
      '@context': 'https://schema.org',
      '@type': 'WebSite',
      name: brand.name,
      url: siteUrl,
    },
    {
      '@context': 'https://schema.org',
      '@type': 'Product',
      name: `${brand.name} Proxies`,
      description: `Residential, ISP, datacenter, IPv6 and mobile proxies from ${brand.name}.`,
      brand: { '@type': 'Brand', name: brand.name },
      offers: allPlans.map((p) => ({
        '@type': 'Offer',
        name: p.name,
        price: (p.price_cents / 100).toFixed(2),
        priceCurrency: 'USD',
        availability: 'https://schema.org/InStock',
      })),
    },
    {
      '@context': 'https://schema.org',
      '@type': 'FAQPage',
      mainEntity: faqs.map((f) => ({
        '@type': 'Question',
        name: f.q,
        acceptedAnswer: { '@type': 'Answer', text: f.a },
      })),
    },
  ];

  return (
    <>
      <JsonLd data={jsonLd} />
      {/* ===== Hero ===== */}
      <section className="relative overflow-hidden">
        <Aurora />
        <div className="absolute inset-0 -z-10 bg-grid-pattern bg-[size:44px_44px] opacity-60 [mask-image:radial-gradient(ellipse_70%_60%_at_50%_0%,black,transparent)]" />
        <div className="mx-auto max-w-7xl px-4 pb-20 pt-20 sm:px-6 lg:px-8 lg:pt-28">
          <div className="grid items-center gap-12 lg:grid-cols-2">
            <div className="animate-fade-up">
              <span className="inline-flex items-center gap-2 rounded-full border border-brand-500/30 bg-brand-500/10 px-3.5 py-1.5 text-xs font-medium text-brand-200">
                <Sparkles className="h-3.5 w-3.5" />
                {brand.ipCount} premium IPs · instant provisioning
              </span>
              <h1 className="mt-6 text-4xl font-extrabold leading-[1.05] tracking-tight text-white sm:text-6xl">
                The proxy network <br />
                built for{' '}
                <span className="text-shimmer">scale &amp; trust</span>
              </h1>
              <p className="mt-6 max-w-xl text-lg leading-relaxed text-slate-400">
                {brand.tagline} Per-request rotation, sticky sessions up to 60
                minutes, and a wallet that bills to the cent.
              </p>
              <div className="mt-8 flex flex-wrap items-center gap-3">
                <Link
                  href="/register"
                  className="group inline-flex items-center gap-2 rounded-xl bg-brand-600 px-6 py-3.5 text-sm font-semibold text-white shadow-lg shadow-brand-600/30 transition-all hover:bg-brand-500 hover:shadow-brand-500/40"
                >
                  Start now — it&apos;s free
                  <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
                </Link>
                <Link
                  href="/pricing"
                  className="inline-flex items-center gap-2 rounded-xl border border-ink-600 bg-ink-800/60 px-6 py-3.5 text-sm font-semibold text-white backdrop-blur transition-colors hover:border-brand-500/50"
                >
                  View pricing
                </Link>
              </div>
              <dl className="mt-12 grid max-w-md grid-cols-3 gap-6">
                <div>
                  <dt className="text-2xl font-bold text-white sm:text-3xl">
                    <Counter value={32} suffix="M+" />
                  </dt>
                  <dd className="mt-1 text-xs text-slate-500">IP addresses</dd>
                </div>
                <div>
                  <dt className="text-2xl font-bold text-white sm:text-3xl">
                    <Counter value={99.9} decimals={1} suffix="%" />
                  </dt>
                  <dd className="mt-1 text-xs text-slate-500">Uptime</dd>
                </div>
                <div>
                  <dt className="text-2xl font-bold text-white sm:text-3xl">
                    <Counter value={195} suffix="+" />
                  </dt>
                  <dd className="mt-1 text-xs text-slate-500">Countries</dd>
                </div>
              </dl>
            </div>

            {/* Floating terminal card */}
            <div className="relative animate-fade-in lg:animate-float">
              <div className="absolute -inset-4 -z-10 rounded-3xl bg-brand-gradient opacity-20 blur-2xl" />
              <div className="overflow-hidden rounded-2xl border border-ink-600 bg-ink-850/90 shadow-2xl backdrop-blur">
                <div className="flex items-center gap-2 border-b border-ink-700 bg-ink-900/60 px-4 py-3">
                  <span className="h-3 w-3 rounded-full bg-red-400/80" />
                  <span className="h-3 w-3 rounded-full bg-amber-400/80" />
                  <span className="h-3 w-3 rounded-full bg-emerald-400/80" />
                  <span className="ml-2 flex items-center gap-1.5 text-xs text-slate-500">
                    <Terminal className="h-3.5 w-3.5" /> session.sh
                  </span>
                </div>
                <div className="space-y-3 p-5 font-mono text-[13px] leading-relaxed">
                  <p className="text-slate-500"># rotating residential endpoint</p>
                  <p className="text-slate-300">
                    <span className="text-accent-400">curl</span> -x http://
                    <span className="text-brand-300">user</span>:
                    <span className="text-brand-300">pass</span>@gw.{brand.name.toLowerCase()}.io:7777 \
                  </p>
                  <p className="pl-6 text-slate-300">https://api.ipify.org</p>
                  <p className="text-emerald-400">→ 188.42.19.107 <span className="text-slate-600">(Berlin, DE)</span></p>
                  <div className="!mt-4 flex items-center justify-between rounded-lg border border-ink-700 bg-ink-900/60 px-3 py-2">
                    <span className="flex items-center gap-2 text-xs text-slate-400">
                      <span className="h-2 w-2 animate-pulse rounded-full bg-emerald-400" /> live
                    </span>
                    <span className="flex items-center gap-1.5 text-xs text-slate-500">
                      <Copy className="h-3.5 w-3.5" /> host:port:user:pass
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* ===== Tools marquee ===== */}
      <section className="border-y border-ink-700/60 bg-ink-850/40 py-8">
        <p className="mb-6 text-center text-xs font-medium uppercase tracking-widest text-slate-600">
          Drops into any stack
        </p>
        <Marquee items={tools} />
      </section>

      {/* ===== Proxy types ===== */}
      <section id="network" className="mx-auto max-w-7xl px-4 py-24 sm:px-6 lg:px-8">
        <Reveal className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
            Every proxy type, one platform
          </h2>
          <p className="mt-4 text-slate-400">
            Pick the right network for the job — and switch any time from your dashboard.
          </p>
        </Reveal>
        <div className="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-5">
          {proxyTypes.map((t, i) => {
            const Icon = t.icon;
            return (
              <Reveal key={t.title} delay={i * 80}>
                <div className="card-hover group h-full rounded-2xl border border-ink-600 bg-ink-800/50 p-6">
                  <div className="flex items-center justify-between">
                    <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-brand-500/10 text-brand-300 ring-1 ring-brand-500/20 transition-colors group-hover:bg-brand-500/20">
                      <Icon className="h-5 w-5" />
                    </div>
                    <span className="rounded-full bg-ink-700 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-accent-300">
                      {t.tag}
                    </span>
                  </div>
                  <h3 className="mt-4 font-semibold text-white">{t.title}</h3>
                  <p className="mt-2 text-sm text-slate-400">{t.desc}</p>
                </div>
              </Reveal>
            );
          })}
        </div>
      </section>

      {/* ===== How it works ===== */}
      <section className="relative overflow-hidden border-y border-ink-700/60 bg-ink-850/30 py-24">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <Reveal className="mx-auto max-w-2xl text-center">
            <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
              Live in three steps
            </h2>
            <p className="mt-4 text-slate-400">From signup to your first request in under a minute.</p>
          </Reveal>
          <div className="mt-14 grid gap-6 md:grid-cols-3">
            {steps.map((s, i) => (
              <Reveal key={s.n} delay={i * 120}>
                <div className="relative h-full rounded-2xl border border-ink-600 bg-ink-800/50 p-7">
                  <span className="bg-brand-gradient bg-clip-text font-mono text-4xl font-bold text-transparent">
                    {s.n}
                  </span>
                  <h3 className="mt-3 text-lg font-semibold text-white">{s.title}</h3>
                  <p className="mt-2 text-sm text-slate-400">{s.desc}</p>
                </div>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* ===== Features ===== */}
      <section id="features" className="mx-auto max-w-7xl px-4 py-24 sm:px-6 lg:px-8">
        <Reveal className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
            Built for serious operators
          </h2>
          <p className="mt-4 text-slate-400">Everything you need to provision, rotate and monitor at scale.</p>
        </Reveal>
        <div className="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((f, i) => {
            const Icon = f.icon;
            return (
              <Reveal key={f.title} delay={(i % 3) * 80}>
                <div className="card-hover h-full rounded-2xl border border-ink-600 bg-ink-800/50 p-6">
                  <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-accent-500/10 text-accent-300 ring-1 ring-accent-500/20">
                    <Icon className="h-5 w-5" />
                  </div>
                  <h3 className="mt-4 font-semibold text-white">{f.title}</h3>
                  <p className="mt-2 text-sm text-slate-400">{f.desc}</p>
                </div>
              </Reveal>
            );
          })}
        </div>
      </section>

      {/* ===== Stats band ===== */}
      <section className="relative overflow-hidden py-20">
        <Aurora />
        <div className="mx-auto grid max-w-5xl grid-cols-2 gap-8 px-4 sm:px-6 md:grid-cols-4 lg:px-8">
          {[
            { v: 32, suffix: 'M+', label: 'IP addresses' },
            { v: 195, suffix: '+', label: 'Countries' },
            { v: 99.9, suffix: '%', label: 'Uptime SLA', decimals: 1 },
            { v: 60, suffix: ' min', label: 'Max sticky session' },
          ].map((s) => (
            <Reveal key={s.label} className="text-center">
              <div className="text-3xl font-extrabold text-white sm:text-4xl">
                <Counter value={s.v} suffix={s.suffix} decimals={s.decimals ?? 0} />
              </div>
              <p className="mt-2 text-sm text-slate-400">{s.label}</p>
            </Reveal>
          ))}
        </div>
      </section>

      {/* ===== Pricing preview ===== */}
      {previewPlans.length > 0 && (
        <section id="pricing" className="mx-auto max-w-7xl px-4 py-24 sm:px-6 lg:px-8">
          <Reveal className="mx-auto max-w-2xl text-center">
            <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
              Simple, usage-based pricing
            </h2>
            <p className="mt-4 text-slate-400">
              Every proxy type — residential, ISP, datacenter, IPv6 and mobile — billed per GB or IP from one wallet.
            </p>
          </Reveal>
          <div className="mt-14 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {previewPlans.map((plan, i) => (
              <Reveal key={plan.id} delay={(i % 3) * 90}>
                <PlanCard plan={plan} popular={plan.code === 'resi-rotating'} />
              </Reveal>
            ))}
          </div>
          <div className="mt-8 text-center">
            <Link href="/pricing" className="inline-flex items-center gap-1.5 text-sm font-semibold text-brand-400 hover:text-brand-300">
              See the full catalog <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </section>
      )}

      {/* ===== FAQ ===== */}
      <section className="border-t border-ink-700/60 bg-ink-850/30 py-24">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <Reveal className="mx-auto mb-12 max-w-2xl text-center">
            <h2 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">
              Frequently asked questions
            </h2>
          </Reveal>
          <Reveal>
            <Faq items={faqs} />
          </Reveal>
        </div>
      </section>

      {/* ===== Final CTA ===== */}
      <section className="mx-auto max-w-7xl px-4 py-24 sm:px-6 lg:px-8">
        <Reveal>
          <div className="relative overflow-hidden rounded-3xl border border-brand-500/30 bg-ink-800/60 px-6 py-16 text-center">
            <div className="absolute inset-0 -z-10 bg-brand-gradient opacity-[0.08]" />
            <Shield className="mx-auto h-10 w-10 text-brand-400" />
            <h2 className="mt-5 text-3xl font-bold tracking-tight text-white sm:text-4xl">
              Ready to deploy your first proxy?
            </h2>
            <p className="mx-auto mt-4 max-w-xl text-slate-400">
              Create an account, top up your wallet, and provision proxies in under a minute with {brand.name}.
            </p>
            <Link
              href="/register"
              className="group mt-8 inline-flex items-center gap-2 rounded-xl bg-brand-600 px-7 py-3.5 text-sm font-semibold text-white shadow-lg shadow-brand-600/30 transition-all hover:bg-brand-500"
            >
              Create free account
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
            </Link>
          </div>
        </Reveal>
      </section>
    </>
  );
}
