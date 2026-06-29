'use client';

import { useState } from 'react';
import { Minus, Plus } from 'lucide-react';

export interface FaqItem {
  q: string;
  a: string;
}

export function Faq({ items }: { items: FaqItem[] }) {
  const [open, setOpen] = useState<number | null>(0);
  return (
    <div className="mx-auto max-w-3xl divide-y divide-ink-700 rounded-2xl border border-ink-600 bg-ink-800/40">
      {items.map((item, i) => {
        const isOpen = open === i;
        return (
          <div key={i}>
            <button
              onClick={() => setOpen(isOpen ? null : i)}
              className="flex w-full items-center justify-between gap-4 px-6 py-5 text-left"
              aria-expanded={isOpen}
            >
              <span className="text-base font-semibold text-white">{item.q}</span>
              <span className="shrink-0 text-brand-400">
                {isOpen ? <Minus className="h-5 w-5" /> : <Plus className="h-5 w-5" />}
              </span>
            </button>
            <div
              className="grid overflow-hidden px-6 transition-all duration-300 ease-out"
              style={{
                gridTemplateRows: isOpen ? '1fr' : '0fr',
                opacity: isOpen ? 1 : 0,
                paddingBottom: isOpen ? '1.25rem' : 0,
              }}
            >
              <div className="min-h-0">
                <p className="text-sm leading-relaxed text-slate-400">{item.a}</p>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
