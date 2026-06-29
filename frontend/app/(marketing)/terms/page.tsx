import type { Metadata } from 'next';

export const metadata: Metadata = { title: 'Terms of Service' };

const siteName = process.env.NEXT_PUBLIC_SITE_NAME ?? 'VaultProxies Reseller';

export default function TermsPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold text-white">Terms of Service</h1>
      <div className="prose prose-invert mt-6 max-w-none space-y-4 text-slate-300">
        <p>
          These Terms govern your use of {siteName}. By creating an account you
          agree to use the proxy network only for lawful purposes and in
          compliance with the acceptable-use policy below.
        </p>
        <h2 className="text-xl font-semibold text-white">Acceptable use</h2>
        <p>
          You may not use the service to send spam, conduct fraud, distribute
          malware, attempt unauthorized access to systems, or violate the rights
          of third parties. We may suspend accounts that breach these terms.
        </p>
        <h2 className="text-xl font-semibold text-white">Billing</h2>
        <p>
          The service is prepaid. Wallet top-ups are non-refundable except where
          required by law or where provisioning fails (in which case the charge
          is automatically reversed to your wallet).
        </p>
        <h2 className="text-xl font-semibold text-white">Availability</h2>
        <p>
          We strive for high availability but do not guarantee uninterrupted
          service. Upstream network capacity is provided on a best-effort basis.
        </p>
        <p className="text-sm text-slate-500">
          This is a template. Replace with your own legal terms before launch.
        </p>
      </div>
    </div>
  );
}
