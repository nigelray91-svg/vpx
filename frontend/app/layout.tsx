import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { Providers } from '@/components/Providers';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

const siteName = process.env.NEXT_PUBLIC_SITE_NAME ?? 'VaultProxies Reseller';

export const metadata: Metadata = {
  title: {
    default: `${siteName} — Premium Proxy Network`,
    template: `%s · ${siteName}`,
  },
  description:
    'Residential, ISP, datacenter, IPv6 and mobile proxies. 32M+ IPs, instant provisioning, per-request rotation and sticky sessions.',
  robots: { index: true, follow: true },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={inter.variable}>
      <body className="min-h-screen">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
