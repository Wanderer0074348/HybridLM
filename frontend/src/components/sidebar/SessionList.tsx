import { useChat } from "@/context/ChatContext";
import { Icon } from "@/components/ui/Icon";
import { Spinner } from "@/components/ui/Spinner";
import { truncate } from "@/utils/format";

export function SessionList() {
  const { sessions, sessionsLoading, activeSessionId, selectSession, deleteSession } = useChat();

  if (sessionsLoading && sessions.length === 0) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 text-xs text-text-muted">
        <Spinner size={12} />
        <span>Loading…</span>
      </div>
    );
  }

  if (sessions.length === 0) {
    return (
      <p className="px-3 py-2 text-xs text-text-muted">No recent chats yet.</p>
    );
  }

  return (
    <ul className="space-y-0.5">
      {sessions.map((s) => {
        const active = s.session_id === activeSessionId;
        return (
          <li key={s.session_id} className="group relative">
            <button
              onClick={() => selectSession(s.session_id)}
              className={`flex w-full items-center rounded-lg px-3 py-2 pr-8 text-left text-sm transition-colors ${
                active
                  ? "bg-bg-hover text-text-primary"
                  : "text-text-secondary hover:bg-bg-hover hover:text-text-primary"
              }`}
            >
              <span className="truncate">{truncate(s.title || "Untitled", 36)}</span>
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                if (confirm("Delete this conversation?")) void deleteSession(s.session_id);
              }}
              aria-label="Delete chat"
              className="absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-1 text-text-muted opacity-0 transition-opacity hover:bg-bg-base hover:text-text-primary group-hover:opacity-100"
            >
              <Icon.Trash size={14} />
            </button>
          </li>
        );
      })}
    </ul>
  );
}
