import { useEffect, useRef } from "react";
import type { ChatViewMessage } from "@/types";
import { Message } from "./Message";

type Props = { messages: ChatViewMessage[] };

export function MessageList({ messages }: Props) {
  const endRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [messages.length, messages[messages.length - 1]?.content]);

  return (
    <div className="mx-auto w-full max-w-3xl space-y-8 px-6 py-8">
      {messages.map((m, i) => (
        <Message key={`${m.timestamp}-${i}`} message={m} />
      ))}
      <div ref={endRef} />
    </div>
  );
}
