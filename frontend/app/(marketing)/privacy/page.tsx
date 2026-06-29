import type { Metadata } from 'next';

export const metadata: Metadata = { title: 'Privacy Policy' };

const siteName = process.env.NEXT_PUBLIC_SITE_NAME ?? 'VaultProxies Reseller';

export default function PrivacyPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold text-white">Privacy Policy</h1>
      <div className="mt-6 max-w-none space-y-4 text-slate-300">
        <p>
          {siteName} collects only the data needed to operate the service: your
          email, authentication credentials (stored hashed), wallet and order
          history, and payment references from our processors.
        </p>
        <h2 className="text-xl font-semibold text-white">Payments</h2>
        <p>
          Card payments are processed by Stripe and crypto payments by
          NOWPayments. We never store full card numbers — those are handled
          entirely by the processor.
        </p>
        <h2 className="text-xl font-semibold text-white">Cookies</h2>
        <p>
          We use strictly-necessary cookies to keep you signed in and to protect
          against cross-site request forgery. We do not use third-party
          advertising cookies.
        </p>
        <h2 className="text-xl font-semibold text-white">Your rights</h2>
        <p>
          You can request export or deletion of your account data by contacting
          support.
        </p>
        <p className="text-sm text-slate-500">
          This is a template. Replace with your own privacy policy before launch.
        </p>
      </div>
    </div>
  );
}
