import { useNavigate } from "react-router-dom";
import { Icon } from "@/components/ui/Icon";
import { SidebarItem } from "./SidebarItem";
import { SessionList } from "./SessionList";
import { UserCard } from "./UserCard";
import { useChat } from "@/context/ChatContext";

type SidebarProps = {
  collapsed: boolean;
  onToggle: () => void;
};

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const { newSession, activeSessionId } = useChat();
  const navigate = useNavigate();

  if (collapsed) {
    return (
      <aside className="flex h-full w-14 flex-col items-center justify-between border-r border-border-subtle bg-bg-sidebar py-4">
        <div className="flex flex-col items-center gap-2">
          <button onClick={onToggle} className="icon-btn" aria-label="Expand sidebar">
            <Icon.Sidebar size={18} />
          </button>
          <button onClick={newSession} className="icon-btn" aria-label="New chat" title="New chat">
            <Icon.Plus size={18} />
          </button>
        </div>
      </aside>
    );
  }

  return (
    <aside className="flex h-full w-72 flex-col border-r border-border-subtle bg-bg-sidebar">
      <div className="flex items-center justify-between px-4 py-4">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-accent text-white">
            <span className="font-serif text-sm font-semibold">H</span>
          </div>
          <span className="font-serif text-lg text-text-primary">HybridLM</span>
        </div>
        <button onClick={onToggle} className="icon-btn" aria-label="Collapse sidebar">
          <Icon.Sidebar size={18} />
        </button>
      </div>

      <nav className="px-2">
        <SidebarItem
          icon={<Icon.Plus size={18} />}
          label="New chat"
          active={activeSessionId === null}
          onClick={newSession}
        />
        <SidebarItem icon={<Icon.Search size={18} />} label="Search" />
        <SidebarItem icon={<Icon.Chats size={18} />} label="Chats" />
        <SidebarItem
          icon={<Icon.Sparkle size={18} />}
          label="Review demo"
          onClick={() => navigate("/review")}
        />
      </nav>

      <div className="mt-4 flex-1 overflow-y-auto px-2">
        <p className="px-3 pb-1.5 text-[11px] font-semibold uppercase tracking-wider text-text-muted">
          Recents
        </p>
        <SessionList />
      </div>

      <div className="border-t border-border-subtle px-2 py-3">
        <UserCard />
      </div>
    </aside>
  );
}
