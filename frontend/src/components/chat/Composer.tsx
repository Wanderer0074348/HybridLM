import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { Icon } from "@/components/ui/Icon";
import { Spinner } from "@/components/ui/Spinner";

type Props = {
  onSend: (text: string) => void | Promise<void>;
  sending: boolean;
  placeholder?: string;
  autoFocus?: boolean;
};

const MAX_HEIGHT = 220;

export function Composer({ onSend, sending, placeholder = "Type / for skills", autoFocus }: Props) {
  const [value, setValue] = useState("");
  const ref = useRef<HTMLTextAreaElement | null>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, MAX_HEIGHT)}px`;
  }, [value]);

  useEffect(() => {
    if (autoFocus) ref.current?.focus();
  }, [autoFocus]);

  const submit = async () => {
    const text = value.trim();
    if (!text || sending) return;
    setValue("");
    await onSend(text);
  };

  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void submit();
    }
  };

  return (
    <div className="rounded-2xl border border-border-default bg-bg-input shadow-lg shadow-black/20 transition-shadow focus-within:border-border-default focus-within:shadow-black/40">
      <textarea
        ref={ref}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={onKeyDown}
        rows={1}
        placeholder={placeholder}
        disabled={sending}
        className="block w-full bg-transparent px-5 py-4 text-[15px] text-text-primary placeholder:text-text-muted focus:outline-none"
        style={{ maxHeight: MAX_HEIGHT }}
      />
      <div className="flex items-center justify-between px-3 pb-3">
        <button
          type="button"
          className="icon-btn"
          aria-label="Add attachment"
          tabIndex={-1}
        >
          <Icon.Plus size={18} />
        </button>
        <button
          type="button"
          onClick={() => void submit()}
          disabled={sending || !value.trim()}
          aria-label="Send message"
          className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent text-white transition-colors hover:bg-accent-hover disabled:bg-bg-hover disabled:text-text-muted"
        >
          {sending ? <Spinner size={14} /> : <Icon.Send size={16} />}
        </button>
      </div>
    </div>
  );
}
