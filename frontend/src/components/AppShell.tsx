import { useState } from "react";
import { Sidebar } from "./sidebar/Sidebar";
import { ChatView } from "./chat/ChatView";

export function AppShell() {
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div className="flex h-full w-full bg-bg-base">
      <Sidebar collapsed={collapsed} onToggle={() => setCollapsed((c) => !c)} />
      <main className="relative flex-1 overflow-hidden">
        <ChatView />
      </main>
    </div>
  );
}
