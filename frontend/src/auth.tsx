import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { authApi } from "./api";
import type { CurrentUser } from "./types";
import { ErrorMessage, Loading } from "./components/UI";

interface AuthState {
  user: CurrentUser;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("AuthContext is not available");
  return value;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loading, setLoading] = useState(true);

  async function refresh() {
    const next = await authApi.me();
    setUser(next);
  }

  async function logout() {
    await authApi.logout();
    setUser(null);
  }

  useEffect(() => {
    authApi
      .me()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setLoading(false));
  }, []);

  const value = useMemo(
    () => (user ? { user, refresh, logout } : null),
    [user],
  );

  if (loading) return <Loading />;
  if (!user) return <LoginScreen onLogin={setUser} />;
  return <AuthContext.Provider value={value!}>{children}</AuthContext.Provider>;
}

function LoginScreen({ onLogin }: { onLogin: (user: CurrentUser) => void }) {
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (busy) return;
    setError("");
    setBusy(true);
    try {
      if (!sent) {
        await authApi.requestCode(email);
        setSent(true);
      } else {
        const user = await authApi.verifyCode(email, code);
        onLogin(user);
      }
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="auth-page">
      <form className="panel form auth-panel" onSubmit={submit}>
        <div className="eyebrow">ВХОД</div>
        <h1>Войдите по почте</h1>
        <p className="intro">Без пароля. Пришлём одноразовый код.</p>
        <fieldset disabled={busy}>
          <label htmlFor="auth-email">Почта</label>
          <input
            id="auth-email"
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
          />
          {sent && (
            <>
              <label htmlFor="auth-code">Код</label>
              <input
                id="auth-code"
                inputMode="numeric"
                pattern="[0-9]{6}"
                maxLength={6}
                required
                value={code}
                onChange={(event) => setCode(event.target.value)}
              />
            </>
          )}
          {error && <ErrorMessage message={error} />}
          <button className="button primary full" type="submit">
            {busy
              ? "Проверяем…"
              : sent
                ? "Войти"
                : "Получить код"}
          </button>
        </fieldset>
      </form>
    </main>
  );
}
