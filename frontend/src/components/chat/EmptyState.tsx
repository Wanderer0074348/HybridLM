import { Composer } from "./Composer";
import { useAuth } from "@/context/AuthContext";

type Props = {
  onSend: (text: string) => void | Promise<void>;
  sending: boolean;
};

const SUGGESTIONS = [
  { label: "Code", prompt: "Write a Python function that " },
  { label: "Learn", prompt: "Explain in simple terms how " },
  { label: "Write", prompt: "Draft a short post about " },
  { label: "Reason", prompt: "Walk me through the trade-offs of " },
  { label: "Plan", prompt: "Outline a one-week plan to " },
];

export function EmptyState({ onSend, sending }: Props) {
  const { user } = useAuth();
  const greeting = user?.name ? user.name.split(" ")[0] : null;

  return (
    <div className="flex h-full w-full items-center justify-center px-6">
      <div className="w-full max-w-3xl animate-slide-up">
        <h1 className="mb-10 text-center font-serif text-[34px] font-normal leading-tight text-text-primary">
          <span className="mr-2 text-accent">✷</span>
          {greeting ? `Welcome back, ${greeting}.` : "What shall we think through?"}
        </h1>

        <Composer onSend={onSend} sending={sending} autoFocus />

        <div className="mt-5 flex flex-wrap items-center justify-center gap-2">
          {SUGGESTIONS.map((s) => (
            <button
              key={s.label}
              onClick={() => onSend(s.prompt)}
              className="rounded-full border border-border-subtle bg-bg-elevated px-4 py-1.5 text-xs font-medium text-text-secondary transition-colors hover:border-border-default hover:bg-bg-hover hover:text-text-primary"
            >
              {s.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
