import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { ChatViewMessage } from "@/types";
import { Avatar } from "@/components/ui/Avatar";
import { MetadataBadges } from "./MetadataBadges";
import { useAuth } from "@/context/AuthContext";

type Props = { message: ChatViewMessage };

export function Message({ message }: Props) {
  const { user } = useAuth();
  const isUser = message.role === "user";

  return (
    <article className="flex gap-4 animate-fade-in">
      <div className="flex-shrink-0 pt-1">
        {isUser ? (
          <Avatar src={user?.picture} name={user?.name || "You"} size={28} />
        ) : (
          <span className="flex h-7 w-7 items-center justify-center rounded-full bg-accent text-[11px] font-serif font-semibold text-white">
            H
          </span>
        )}
      </div>

      <div className="min-w-0 flex-1">
        <p className="mb-1 text-xs font-semibold text-text-secondary">
          {isUser ? "You" : "HybridLM"}
        </p>

        {message.pending ? (
          <div className="flex items-center gap-1.5 py-2">
            <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-text-muted" />
            <span
              className="h-1.5 w-1.5 animate-pulse rounded-full bg-text-muted"
              style={{ animationDelay: "0.2s" }}
            />
            <span
              className="h-1.5 w-1.5 animate-pulse rounded-full bg-text-muted"
              style={{ animationDelay: "0.4s" }}
            />
          </div>
        ) : isUser ? (
          <div className="whitespace-pre-wrap break-words text-[15px] leading-relaxed text-text-primary">
            {message.content}
          </div>
        ) : (
          <div className="markdown text-[15px] text-text-primary">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.content}</ReactMarkdown>
          </div>
        )}

        {!isUser && !message.pending && message.meta && <MetadataBadges meta={message.meta} />}
      </div>
    </article>
  );
}
