'use client';

import { Suspense, useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/components/AuthProvider';
import { useConfig } from '@/lib/useConfig';
import { Turnstile } from '@/components/Turnstile';
import { AuthShell } from '@/components/AuthShell';
import { ApiError, googleAuthUrl } from '@/lib/api';
import { Button, Field, inputClasses } from '@/components/ui';

const oauthErrors: Record<string, string> = {
  invalid_oauth_state: 'Your sign-in session expired. Please try again.',
  invalid_oauth_response: 'Google sign-in was cancelled or failed.',
  oauth_exchange_failed: 'Could not complete Google sign-in. Please try again.',
  oauth_account_error: 'There was a problem with your account. Contact support.',
};

function LoginInner() {
  const router = useRouter();
  const params = useSearchParams();
  const { login } = useAuth();
  const { config } = useConfig();

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const code = params.get('error');
    if (code) setError(oauthErrors[code] ?? 'Sign-in failed. Please try again.');
  }, [params]);

  const turnstileRequired = config?.turnstile_enabled && !!config?.turnstile_site_key;

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    if (turnstileRequired && !turnstileToken) {
      setError('Please complete the verification challenge.');
      return;
    }
    setSubmitting(true);
    try {
      await login({ email, password, turnstile_token: turnstileToken });
      router.push('/dashboard');
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : 'Login failed. Please try again.',
      );
      setSubmitting(false);
    }
  };

  return (
    <AuthShell title="Welcome back" subtitle="Sign in to manage your proxies and billing.">
      {error && (
        <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {error}
        </div>
      )}

      {config?.google_enabled && (
        <>
          <a
            href={googleAuthUrl()}
            className="mt-6 flex w-full items-center justify-center gap-2 rounded-lg border border-ink-600 bg-ink-700 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-ink-600"
          >
            <GoogleIcon />
            Continue with Google
          </a>
          <div className="my-6 flex items-center gap-3 text-xs text-slate-500">
            <div className="h-px flex-1 bg-ink-600" />
            OR
            <div className="h-px flex-1 bg-ink-600" />
          </div>
        </>
      )}

      <form onSubmit={onSubmit} className={config?.google_enabled ? 'space-y-4' : 'mt-6 space-y-4'}>
        <Field label="Email" htmlFor="email">
          <input
            id="email"
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={inputClasses}
            placeholder="you@example.com"
          />
        </Field>
        <Field label="Password" htmlFor="password">
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className={inputClasses}
            placeholder="••••••••"
          />
        </Field>

        {turnstileRequired && (
          <Turnstile
            siteKey={config!.turnstile_site_key}
            onVerify={setTurnstileToken}
            onExpire={() => setTurnstileToken('')}
          />
        )}

        <Button type="submit" fullWidth loading={submitting}>
          Sign in
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-slate-400">
        Don&apos;t have an account?{' '}
        <Link href="/register" className="font-semibold text-brand-400 hover:text-brand-300">
          Create one
        </Link>
      </p>
    </AuthShell>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginInner />
    </Suspense>
  );
}

function GoogleIcon() {
  return (
    <svg className="h-4 w-4" viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="#FFC107"
        d="M43.6 20.5h-1.9V20H24v8h11.3c-1.6 4.7-6.1 8-11.3 8-6.6 0-12-5.4-12-12s5.4-12 12-12c3.1 0 5.9 1.2 8 3.1l5.7-5.7C34 4.1 29.3 2 24 2 11.8 2 2 11.8 2 24s9.8 22 22 22 22-9.8 22-22c0-1.5-.2-2.6-.4-3.5z"
      />
      <path
        fill="#FF3D00"
        d="M6.3 14.7l6.6 4.8C14.7 16 19 13 24 13c3.1 0 5.9 1.2 8 3.1l5.7-5.7C34 6.1 29.3 4 24 4 16 4 9.1 8.6 6.3 14.7z"
      />
      <path
        fill="#4CAF50"
        d="M24 44c5.2 0 9.9-2 13.4-5.2l-6.2-5.2C29.2 35.5 26.7 36 24 36c-5.2 0-9.6-3.3-11.3-7.9l-6.5 5C9.1 39.4 16 44 24 44z"
      />
      <path
        fill="#1976D2"
        d="M43.6 20.5H24v8h11.3c-.8 2.2-2.2 4.1-4.1 5.4l6.2 5.2C40.9 36.7 44 31.1 44 24c0-1.5-.2-2.6-.4-3.5z"
      />
    </svg>
  );
}
