import { useChat } from "@/context/ChatContext";
import { Composer } from "./Composer";
import { MessageList } from "./MessageList";
import { EmptyState } from "./EmptyState";
import { Spinner } from "@/components/ui/Spinner";

export function ChatView() {
  const { messages, messagesLoading, sending, sendMessage, error } = useChat();

  if (messagesLoading) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <Spinner size={20} />
      </div>
    );
  }

  if (messages.length === 0) {
    return <EmptyState onSend={sendMessage} sending={sending} />;
  }

  return (
    <div className="flex h-full w-full flex-col">
      <div className="flex-1 overflow-y-auto">
        <MessageList messages={messages} />
      </div>
      <div className="border-t border-border-subtle bg-bg-base px-4 pb-5 pt-4">
        <div className="mx-auto w-full max-w-3xl">
          {error && (
            <p className="mb-2 text-center text-xs text-red-400">{error}</p>
          )}
          <Composer onSend={sendMessage} sending={sending} />
          <p className="mt-2 text-center text-[11px] text-text-muted">
            HybridLM routes between edge and cloud models — responses may include errors.
          </p>
        </div>
      </div>
    </div>
  );
}
