import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { authApi, ApiError } from "@/api";
import type { User } from "@/types";

type AuthState = {
  user: User | null;
  loading: boolean;
  error: string | null;
};

type AuthContextValue = AuthState & {
  login: () => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ user: null, loading: true, error: null });

  const refresh = useCallback(async () => {
    setState((s) => ({ ...s, loading: true, error: null }));
    try {
      const { user } = await authApi.me();
      setState({ user, loading: false, error: null });
    } catch (err) {
      const status = err instanceof ApiError ? err.status : 0;
      setState({
        user: null,
        loading: false,
        error: status === 401 ? null : (err as Error).message,
      });
    }
  }, []);

  const login = useCallback(async () => {
    const { url } = await authApi.loginUrl();
    window.location.href = url;
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } finally {
      setState({ user: null, loading: false, error: null });
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const value = useMemo<AuthContextValue>(
    () => ({ ...state, login, logout, refresh }),
    [state, login, logout, refresh]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
