'use client';

import Link from 'next/link';
import { Loader2 } from 'lucide-react';
import type { ButtonHTMLAttributes, ReactNode } from 'react';

function cx(...classes: Array<string | false | null | undefined>): string {
  return classes.filter(Boolean).join(' ');
}

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';

const variantClasses: Record<Variant, string> = {
  primary:
    'bg-brand-600 hover:bg-brand-500 text-white shadow-lg shadow-brand-600/20',
  secondary:
    'bg-ink-700 hover:bg-ink-600 text-white border border-ink-600',
  ghost: 'bg-transparent hover:bg-ink-700 text-slate-200',
  danger: 'bg-red-600 hover:bg-red-500 text-white',
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  loading?: boolean;
  fullWidth?: boolean;
}

export function Button({
  variant = 'primary',
  loading = false,
  fullWidth = false,
  className,
  children,
  disabled,
  ...rest
}: ButtonProps) {
  return (
    <button
      {...rest}
      disabled={disabled || loading}
      className={cx(
        'inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-semibold transition-colors',
        'focus:outline-none focus:ring-2 focus:ring-brand-500 focus:ring-offset-2 focus:ring-offset-ink-900',
        'disabled:cursor-not-allowed disabled:opacity-50',
        variantClasses[variant],
        fullWidth && 'w-full',
        className,
      )}
    >
      {loading && <Loader2 className="h-4 w-4 animate-spin" />}
      {children}
    </button>
  );
}

interface LinkButtonProps {
  href: string;
  variant?: Variant;
  className?: string;
  children: ReactNode;
}

export function LinkButton({
  href,
  variant = 'primary',
  className,
  children,
}: LinkButtonProps) {
  return (
    <Link
      href={href}
      className={cx(
        'inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-semibold transition-colors',
        variantClasses[variant],
        className,
      )}
    >
      {children}
    </Link>
  );
}

export function Card({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cx(
        'rounded-xl border border-ink-600 bg-ink-800/60 p-5 backdrop-blur',
        className,
      )}
    >
      {children}
    </div>
  );
}

export function Badge({
  children,
  tone = 'default',
}: {
  children: ReactNode;
  tone?: 'default' | 'success' | 'warning' | 'danger' | 'info';
}) {
  const tones: Record<string, string> = {
    default: 'bg-ink-600 text-slate-300',
    success: 'bg-emerald-500/15 text-emerald-300 border border-emerald-500/30',
    warning: 'bg-amber-500/15 text-amber-300 border border-amber-500/30',
    danger: 'bg-red-500/15 text-red-300 border border-red-500/30',
    info: 'bg-brand-500/15 text-brand-300 border border-brand-500/30',
  };
  return (
    <span
      className={cx(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        tones[tone],
      )}
    >
      {children}
    </span>
  );
}

export function statusTone(
  status: string,
): 'default' | 'success' | 'warning' | 'danger' | 'info' {
  const s = (status || '').toLowerCase();
  if (['active', 'paid', 'completed', 'success', 'succeeded'].includes(s))
    return 'success';
  if (['pending', 'processing', 'provisioning'].includes(s)) return 'warning';
  if (['failed', 'cancelled', 'canceled', 'expired', 'refunded'].includes(s))
    return 'danger';
  return 'default';
}

export function Spinner({ label }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-slate-400">
      <Loader2 className="h-5 w-5 animate-spin" />
      {label && <span className="text-sm">{label}</span>}
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-ink-600 bg-ink-800/40 px-6 py-12 text-center">
      <p className="text-base font-semibold text-slate-200">{title}</p>
      {description && (
        <p className="mt-1 max-w-md text-sm text-slate-400">{description}</p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

export function ErrorState({
  message,
  onRetry,
}: {
  message: string;
  onRetry?: () => void;
}) {
  return (
    <div className="rounded-xl border border-red-500/30 bg-red-500/10 px-5 py-4 text-sm text-red-200">
      <p>{message}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-2 text-red-100 underline underline-offset-2 hover:text-white"
        >
          Try again
        </button>
      )}
    </div>
  );
}

export function Field({
  label,
  htmlFor,
  error,
  children,
  hint,
}: {
  label: string;
  htmlFor?: string;
  error?: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <div>
      <label
        htmlFor={htmlFor}
        className="mb-1.5 block text-sm font-medium text-slate-300"
      >
        {label}
      </label>
      {children}
      {hint && !error && (
        <p className="mt-1 text-xs text-slate-500">{hint}</p>
      )}
      {error && <p className="mt-1 text-xs text-red-400">{error}</p>}
    </div>
  );
}

export const inputClasses =
  'w-full rounded-lg border border-ink-600 bg-ink-850 px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500';
