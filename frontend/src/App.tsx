import { Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "@/context/AuthContext";
import { ChatProvider } from "@/context/ChatContext";
import { LoginScreen } from "@/components/auth/LoginScreen";
import { AuthCallback } from "@/components/auth/AuthCallback";
import { AppShell } from "@/components/AppShell";
import { ReviewPage } from "@/components/review/ReviewPage";
import { Spinner } from "@/components/ui/Spinner";

function Protected({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();

  if (loading) {
    return (
      <div className="flex h-full w-full items-center justify-center bg-bg-base">
        <Spinner size={20} />
      </div>
    );
  }

  if (!user) return <LoginScreen />;
  return <>{children}</>;
}

function ChatHome() {
  return (
    <ChatProvider enabled>
      <AppShell />
    </ChatProvider>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/auth/callback" element={<AuthCallback />} />
        <Route path="/" element={<Protected><ChatHome /></Protected>} />
        <Route path="/review" element={<Protected><ReviewPage /></Protected>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AuthProvider>
  );
}
