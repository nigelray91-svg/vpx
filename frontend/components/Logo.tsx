import Link from 'next/link';
import { brand } from '@/lib/brand';

/** The brand mark: a stylized proxy/relay network node with an orbiting ring. */
export function LogoMark({
  size = 32,
  className,
  animated = true,
}: {
  size?: number;
  className?: string;
  animated?: boolean;
}) {
  const gid = 'logoGrad';
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 40 40"
      fill="none"
      className={className}
      aria-hidden="true"
    >
      <defs>
        <linearGradient id={gid} x1="4" y1="4" x2="36" y2="36" gradientUnits="userSpaceOnUse">
          <stop stopColor="#818cf8" />
          <stop offset="0.5" stopColor="#6366f1" />
          <stop offset="1" stopColor="#22d3ee" />
        </linearGradient>
      </defs>

      {/* outer rounded container */}
      <rect x="1.5" y="1.5" width="37" height="37" rx="11" stroke={`url(#${gid})`} strokeWidth="1.5" opacity="0.4" />

      {/* orbiting ring */}
      <g
        style={{ transformBox: 'fill-box', transformOrigin: 'center' }}
        className={animated ? 'origin-center animate-orbit' : ''}
      >
        <ellipse
          cx="20"
          cy="20"
          rx="13"
          ry="6.2"
          stroke={`url(#${gid})`}
          strokeWidth="1.6"
          transform="rotate(45 20 20)"
          opacity="0.85"
        />
        <circle cx="29.2" cy="10.8" r="2.4" fill="#22d3ee" />
      </g>

      {/* connecting relay lines */}
      <path d="M20 20 L11 12 M20 20 L30 14 M20 20 L13 29" stroke={`url(#${gid})`} strokeWidth="1.4" strokeLinecap="round" opacity="0.55" />

      {/* satellite nodes */}
      <circle cx="11" cy="12" r="2.1" fill="#818cf8" />
      <circle cx="30" cy="14" r="1.9" fill="#6366f1" />
      <circle cx="13" cy="29" r="1.9" fill="#22d3ee" />

      {/* central core */}
      <circle cx="20" cy="20" r="4.2" fill={`url(#${gid})`} />
      <circle cx="20" cy="20" r="4.2" fill={`url(#${gid})`} className={animated ? 'animate-pulse-glow' : ''} opacity="0.5" />
    </svg>
  );
}

/** Mark + wordmark, optionally wrapped in a link to home. */
export function Logo({
  href = '/',
  size = 30,
  className = '',
  showText = true,
  animated = true,
}: {
  href?: string | null;
  size?: number;
  className?: string;
  showText?: boolean;
  animated?: boolean;
}) {
  const content = (
    <span className={`flex items-center gap-2.5 ${className}`}>
      <LogoMark size={size} animated={animated} />
      {showText && (
        <span className="text-lg font-bold tracking-tight text-white">
          {brand.name}
        </span>
      )}
    </span>
  );
  if (href === null) return content;
  return (
    <Link href={href} className="group inline-flex items-center">
      {content}
    </Link>
  );
}
