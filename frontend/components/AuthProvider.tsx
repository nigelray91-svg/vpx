'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { ApiError, api } from '@/lib/api';
import type { User } from '@/lib/types';

interface LoginArgs {
  email: string;
  password: string;
  turnstile_token?: string;
}

interface RegisterArgs {
  email: string;
  password: string;
  full_name?: string;
  turnstile_token?: string;
}

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  login: (args: LoginArgs) => Promise<User>;
  register: (args: RegisterArgs) => Promise<User>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
  setUser: (user: User | null) => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const loadMe = useCallback(async (signal?: AbortSignal) => {
    try {
      const me = await api.me(signal);
      setUser(me);
    } catch (err) {
      if ((err as Error)?.name === 'AbortError') return;
      // 401 (unauthenticated) is expected for logged-out visitors.
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void loadMe(controller.signal);
    return () => controller.abort();
  }, [loadMe]);

  const login = useCallback(async (args: LoginArgs) => {
    const res = await api.login(args);
    setUser(res.user);
    return res.user;
  }, []);

  const register = useCallback(async (args: RegisterArgs) => {
    const res = await api.register(args);
    setUser(res.user);
    return res.user;
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.logout();
    } catch (err) {
      // Swallow CSRF/expired errors — we clear local state regardless.
      if (!(err instanceof ApiError)) throw err;
    } finally {
      setUser(null);
    }
  }, []);

  const refresh = useCallback(async () => {
    await loadMe();
  }, [loadMe]);

  const value = useMemo<AuthContextValue>(
    () => ({ user, loading, login, register, logout, refresh, setUser }),
    [user, loading, login, register, logout, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
}
