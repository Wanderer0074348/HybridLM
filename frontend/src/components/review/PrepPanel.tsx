import type { RunState } from "@/hooks/usePRReview";
import { SubtaskCard } from "./SubtaskCard";
import { Icon } from "@/components/ui/Icon";

type Props = { prep: RunState };

export function PrepPanel({ prep }: Props) {
  if (prep.order.length === 0) return null;

  return (
    <section className="rounded-xl border border-border-subtle bg-bg-base">
      <header className="flex items-center gap-2 border-b border-border-subtle px-4 py-3">
        <span className="flex h-7 w-7 items-center justify-center rounded-md bg-amber-400/15 text-amber-300">
          <Icon.Search size={14} />
        </span>
        <div>
          <h2 className="text-sm font-semibold text-text-primary">Code-gathering prep</h2>
          <p className="text-[11px] text-text-muted">
            Shared by both runs — keyword extract → repo search → file fetch.
          </p>
        </div>
      </header>
      <div className="space-y-2 p-4">
        {prep.order.map((id) => (
          <SubtaskCard key={id} task={prep.subtasks[id]} />
        ))}
      </div>
    </section>
  );
}
