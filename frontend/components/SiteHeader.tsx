'use client';

import Link from 'next/link';
import { useState } from 'react';
import { Menu, Shield, X } from 'lucide-react';
import { useAuth } from '@/components/AuthProvider';

const siteName = process.env.NEXT_PUBLIC_SITE_NAME ?? 'VaultProxies Reseller';

const navLinks = [
  { href: '/#features', label: 'Features' },
  { href: '/pricing', label: 'Pricing' },
  { href: '/#network', label: 'Network' },
];

export function SiteHeader() {
  const { user, loading } = useAuth();
  const [open, setOpen] = useState(false);

  return (
    <header className="sticky top-0 z-40 border-b border-ink-700/60 bg-ink-900/80 backdrop-blur">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3.5 sm:px-6 lg:px-8">
        <Link href="/" className="flex items-center gap-2">
          <Shield className="h-6 w-6 text-brand-500" />
          <span className="text-lg font-bold tracking-tight text-white">
            {siteName}
          </span>
        </Link>

        <nav className="hidden items-center gap-8 md:flex">
          {navLinks.map((l) => (
            <Link
              key={l.href}
              href={l.href}
              className="text-sm font-medium text-slate-300 transition-colors hover:text-white"
            >
              {l.label}
            </Link>
          ))}
        </nav>

        <div className="hidden items-center gap-3 md:flex">
          {!loading && user ? (
            <Link
              href="/dashboard"
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-brand-500"
            >
              Dashboard
            </Link>
          ) : (
            <>
              <Link
                href="/login"
                className="text-sm font-medium text-slate-300 transition-colors hover:text-white"
              >
                Log in
              </Link>
              <Link
                href="/register"
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-brand-500"
              >
                Get started
              </Link>
            </>
          )}
        </div>

        <button
          className="text-slate-300 md:hidden"
          onClick={() => setOpen((v) => !v)}
          aria-label="Toggle menu"
        >
          {open ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
        </button>
      </div>

      {open && (
        <div className="border-t border-ink-700/60 md:hidden">
          <div className="space-y-1 px-4 py-3">
            {navLinks.map((l) => (
              <Link
                key={l.href}
                href={l.href}
                onClick={() => setOpen(false)}
                className="block rounded-lg px-3 py-2 text-sm font-medium text-slate-300 hover:bg-ink-800"
              >
                {l.label}
              </Link>
            ))}
            <div className="flex gap-2 pt-2">
              {!loading && user ? (
                <Link
                  href="/dashboard"
                  className="flex-1 rounded-lg bg-brand-600 px-4 py-2 text-center text-sm font-semibold text-white"
                >
                  Dashboard
                </Link>
              ) : (
                <>
                  <Link
                    href="/login"
                    className="flex-1 rounded-lg border border-ink-600 px-4 py-2 text-center text-sm font-semibold text-slate-200"
                  >
                    Log in
                  </Link>
                  <Link
                    href="/register"
                    className="flex-1 rounded-lg bg-brand-600 px-4 py-2 text-center text-sm font-semibold text-white"
                  >
                    Get started
                  </Link>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </header>
  );
}
