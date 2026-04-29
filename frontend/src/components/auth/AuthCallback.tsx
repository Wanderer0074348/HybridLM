import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@/context/AuthContext";
import { Spinner } from "@/components/ui/Spinner";

export function AuthCallback() {
  const { refresh } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    void (async () => {
      await refresh();
      navigate("/", { replace: true });
    })();
  }, [refresh, navigate]);

  return (
    <div className="flex h-full w-full items-center justify-center bg-bg-base">
      <div className="flex items-center gap-3 text-text-secondary">
        <Spinner size={18} />
        <span className="text-sm">Signing you in…</span>
      </div>
    </div>
  );
}
