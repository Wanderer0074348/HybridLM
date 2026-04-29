import { useState } from "react";
import { useAuth } from "@/context/AuthContext";
import { Spinner } from "@/components/ui/Spinner";

export function LoginScreen() {
  const { login } = useAuth();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleLogin = async () => {
    setPending(true);
    setError(null);
    try {
      await login();
    } catch (err) {
      setError((err as Error).message);
      setPending(false);
    }
  };

  return (
    <div className="flex h-full w-full items-center justify-center bg-bg-base px-6">
      <div className="w-full max-w-sm animate-slide-up">
        <div className="mb-10 flex flex-col items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-accent text-white">
            <span className="font-serif text-xl font-semibold">H</span>
          </div>
          <div className="text-center">
            <h1 className="font-serif text-2xl text-text-primary">HybridLM</h1>
            <p className="mt-1 text-sm text-text-secondary">
              Cost-aware routing across edge and cloud models.
            </p>
          </div>
        </div>

        <button
          onClick={handleLogin}
          disabled={pending}
          className="flex w-full items-center justify-center gap-3 rounded-xl border border-border-default bg-bg-elevated px-4 py-3 text-sm font-medium text-text-primary transition-colors hover:bg-bg-hover focus-ring disabled:opacity-60"
        >
          {pending ? (
            <Spinner size={16} />
          ) : (
            <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden>
              <path
                fill="#EA4335"
                d="M12 10.2v3.9h5.5c-.2 1.4-1.6 4.1-5.5 4.1-3.3 0-6-2.7-6-6.1s2.7-6.1 6-6.1c1.9 0 3.1.8 3.8 1.5l2.6-2.5C16.8 3.4 14.6 2.4 12 2.4 6.6 2.4 2.3 6.7 2.3 12.1S6.6 21.8 12 21.8c6.9 0 9.5-4.8 9.5-7.3 0-.5 0-.9-.1-1.3H12Z"
              />
            </svg>
          )}
          <span>{pending ? "Redirecting…" : "Continue with Google"}</span>
        </button>

        {error && (
          <p className="mt-4 text-center text-xs text-red-400">{error}</p>
        )}

        <p className="mt-8 text-center text-xs text-text-muted">
          By continuing you agree to the terms of service.
        </p>
      </div>
    </div>
  );
}
