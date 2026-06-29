import { Check, ShieldCheck, Zap, Globe2 } from 'lucide-react';
import { Logo } from '@/components/Logo';
import { Aurora } from '@/components/Aurora';
import { brand } from '@/lib/brand';

const bullets = [
  { icon: Zap, text: 'Instant provisioning — live credentials in seconds.' },
  { icon: Globe2, text: `${brand.ipCount} IPs across 195+ countries.` },
  { icon: ShieldCheck, text: 'Bank-grade security with bot protection built in.' },
];

/**
 * Premium split-screen layout for auth pages: a branded marketing panel on the
 * left (desktop) and the form on the right.
 */
export function AuthShell({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-2">
      {/* Brand panel */}
      <div className="relative hidden overflow-hidden border-r border-ink-700/60 bg-ink-950 lg:flex lg:flex-col lg:justify-between lg:p-12">
        <Aurora />
        <div className="absolute inset-0 -z-10 bg-grid-pattern bg-[size:40px_40px] opacity-40 [mask-image:radial-gradient(ellipse_60%_60%_at_50%_40%,black,transparent)]" />
        <Logo href="/" size={34} />

        <div>
          <h2 className="max-w-md text-4xl font-bold leading-tight tracking-tight text-white">
            The proxy network built for{' '}
            <span className="text-gradient">scale &amp; trust</span>
          </h2>
          <ul className="mt-8 space-y-4">
            {bullets.map((b) => {
              const Icon = b.icon;
              return (
                <li key={b.text} className="flex items-center gap-3 text-slate-300">
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-brand-500/15 text-brand-300 ring-1 ring-brand-500/20">
                    <Icon className="h-5 w-5" />
                  </span>
                  {b.text}
                </li>
              );
            })}
          </ul>
        </div>

        <div className="max-w-md rounded-2xl border border-ink-600 bg-ink-800/50 p-5 backdrop-blur">
          <div className="flex items-center gap-1 text-amber-300">
            {'★★★★★'.split('').map((s, i) => (
              <span key={i}>{s}</span>
            ))}
          </div>
          <p className="mt-3 text-sm leading-relaxed text-slate-300">
            “Switched our scraping stack to {brand.name} and provisioning that used
            to take days now takes seconds. Rock-solid uptime.”
          </p>
          <p className="mt-3 text-xs text-slate-500">— Head of Data, growth-stage SaaS</p>
        </div>
      </div>

      {/* Form panel */}
      <div className="relative flex min-h-screen items-center justify-center px-4 py-12">
        <div className="w-full max-w-md">
          <div className="mb-8 flex justify-center lg:hidden">
            <Logo href="/" size={34} />
          </div>
          <div className="rounded-2xl border border-ink-600 bg-ink-800/70 p-8 shadow-2xl backdrop-blur">
            <h1 className="text-2xl font-bold text-white">{title}</h1>
            <p className="mt-1 text-sm text-slate-400">{subtitle}</p>
            {children}
          </div>
          <p className="mt-6 flex items-center justify-center gap-1.5 text-xs text-slate-600">
            <Check className="h-3.5 w-3.5 text-emerald-500" />
            Secured with encrypted, httpOnly sessions
          </p>
        </div>
      </div>
    </div>
  );
}
