/** Decorative animated gradient blobs for hero/section backdrops. */
export function Aurora({ className = '' }: { className?: string }) {
  return (
    <div className={`pointer-events-none absolute inset-0 -z-10 overflow-hidden ${className}`} aria-hidden="true">
      <div className="absolute left-1/2 top-[-10%] h-[420px] w-[680px] -translate-x-1/2 rounded-full bg-brand-600/30 blur-[120px] animate-aurora-1" />
      <div className="absolute right-[5%] top-[20%] h-[360px] w-[420px] rounded-full bg-accent-500/20 blur-[120px] animate-aurora-2" />
      <div className="absolute left-[2%] top-[35%] h-[320px] w-[380px] rounded-full bg-brand-400/20 blur-[110px] animate-aurora-1" />
    </div>
  );
}
