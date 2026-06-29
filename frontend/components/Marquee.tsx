/**
 * An infinite, edge-faded marquee. Children are duplicated so the loop is
 * seamless; the track translates -50% over one cycle.
 */
export function Marquee({ items }: { items: string[] }) {
  const doubled = [...items, ...items];
  return (
    <div className="marquee-mask relative w-full overflow-hidden">
      <div className="flex w-max animate-marquee gap-12 pr-12">
        {doubled.map((item, i) => (
          <span
            key={i}
            className="whitespace-nowrap text-sm font-semibold uppercase tracking-wider text-slate-500"
          >
            {item}
          </span>
        ))}
      </div>
    </div>
  );
}
