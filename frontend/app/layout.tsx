import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { Providers } from '@/components/Providers';
import { brand } from '@/lib/brand';

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-inter',
  display: 'swap',
});

const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3000';

const description = `${brand.name} — premium residential, ISP, datacenter, IPv6 and mobile proxies. ${brand.ipCount} IPs across 195+ countries, instant provisioning, per-request rotation and sticky sessions up to 60 minutes. Pay-as-you-go with card or crypto.`;

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl),
  title: {
    default: `${brand.name} — Premium Residential, ISP & Mobile Proxies`,
    template: `%s · ${brand.name}`,
  },
  description,
  applicationName: brand.name,
  keywords: [
    'proxies',
    'residential proxies',
    'ISP proxies',
    'datacenter proxies',
    'mobile proxies',
    'IPv6 proxies',
    'rotating proxies',
    'sticky sessions',
    'web scraping proxies',
    'proxy API',
    brand.name,
  ],
  authors: [{ name: brand.name }],
  creator: brand.name,
  publisher: brand.name,
  alternates: { canonical: '/' },
  openGraph: {
    type: 'website',
    url: siteUrl,
    siteName: brand.name,
    title: `${brand.name} — Premium Residential, ISP & Mobile Proxies`,
    description,
    locale: 'en_US',
  },
  twitter: {
    card: 'summary_large_image',
    title: `${brand.name} — Premium Proxies`,
    description,
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-image-preview': 'large',
      'max-snippet': -1,
    },
  },
  category: 'technology',
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
