import * as React from "react";
import { api } from "@/lib/api";

interface AuthState {
  loading: boolean;
  authenticated: boolean;
  username?: string;
  refresh: () => Promise<void>;
}

const AuthContext = React.createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [loading, setLoading] = React.useState(true);
  const [authenticated, setAuthenticated] = React.useState(false);
  const [username, setUsername] = React.useState<string | undefined>();

  const refresh = React.useCallback(async () => {
    setLoading(true);
    try {
      const session = await api.session();
      setAuthenticated(session.authenticated);
      setUsername(session.username);
    } catch {
      setAuthenticated(false);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    refresh();
  }, [refresh]);

  return (
    <AuthContext.Provider value={{ loading, authenticated, username, refresh }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = React.useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
