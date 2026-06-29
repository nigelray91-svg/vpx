import Link from 'next/link';
import { Shield } from 'lucide-react';

const siteName = process.env.NEXT_PUBLIC_SITE_NAME ?? 'VaultProxies Reseller';

export function SiteFooter() {
  const year = new Date().getFullYear();
  return (
    <footer className="border-t border-ink-700/60 bg-ink-900">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="grid gap-8 md:grid-cols-4">
          <div>
            <div className="flex items-center gap-2">
              <Shield className="h-5 w-5 text-brand-500" />
              <span className="font-bold text-white">{siteName}</span>
            </div>
            <p className="mt-3 max-w-xs text-sm text-slate-400">
              Premium residential, ISP, datacenter, IPv6 and mobile proxies for
              teams that scale.
            </p>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Product</h3>
            <ul className="mt-3 space-y-2 text-sm text-slate-400">
              <li>
                <Link href="/pricing" className="hover:text-white">
                  Pricing
                </Link>
              </li>
              <li>
                <Link href="/#features" className="hover:text-white">
                  Features
                </Link>
              </li>
              <li>
                <Link href="/register" className="hover:text-white">
                  Get started
                </Link>
              </li>
            </ul>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Account</h3>
            <ul className="mt-3 space-y-2 text-sm text-slate-400">
              <li>
                <Link href="/login" className="hover:text-white">
                  Log in
                </Link>
              </li>
              <li>
                <Link href="/dashboard" className="hover:text-white">
                  Dashboard
                </Link>
              </li>
            </ul>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Legal</h3>
            <ul className="mt-3 space-y-2 text-sm text-slate-400">
              <li>
                <Link href="/terms" className="hover:text-white">
                  Terms of Service
                </Link>
              </li>
              <li>
                <Link href="/privacy" className="hover:text-white">
                  Privacy Policy
                </Link>
              </li>
            </ul>
          </div>
        </div>
        <div className="mt-10 border-t border-ink-700/60 pt-6 text-sm text-slate-500">
          © {year} {siteName}. All rights reserved.
        </div>
      </div>
    </footer>
  );
}
