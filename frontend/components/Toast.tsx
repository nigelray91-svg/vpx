'use client';

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
} from 'react';
import { CheckCircle2, Info, X, XCircle } from 'lucide-react';

type ToastTone = 'success' | 'error' | 'info';

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
}

interface ToastContextValue {
  notify: (message: string, tone?: ToastTone) => void;
}

const ToastContext = createContext<ToastContextValue | undefined>(undefined);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);

  const remove = useCallback((id: number) => {
    setItems((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const notify = useCallback(
    (message: string, tone: ToastTone = 'info') => {
      const id = Date.now() + Math.random();
      setItems((prev) => [...prev, { id, message, tone }]);
      setTimeout(() => remove(id), 5000);
    },
    [remove],
  );

  const value = useMemo<ToastContextValue>(() => ({ notify }), [notify]);

  const icons: Record<ToastTone, React.ReactNode> = {
    success: <CheckCircle2 className="h-5 w-5 text-emerald-400" />,
    error: <XCircle className="h-5 w-5 text-red-400" />,
    info: <Info className="h-5 w-5 text-brand-400" />,
  };

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed bottom-4 right-4 z-50 flex w-full max-w-sm flex-col gap-2">
        {items.map((t) => (
          <div
            key={t.id}
            className="pointer-events-auto flex items-start gap-3 rounded-lg border border-ink-600 bg-ink-800 px-4 py-3 shadow-xl"
          >
            {icons[t.tone]}
            <p className="flex-1 text-sm text-slate-200">{t.message}</p>
            <button
              onClick={() => remove(t.id)}
              className="text-slate-500 hover:text-slate-300"
              aria-label="Dismiss"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used within a ToastProvider');
  return ctx;
}
