import type { ReactNode } from "react";

type Props = {
  icon: ReactNode;
  label: string;
  active?: boolean;
  onClick?: () => void;
};

export function SidebarItem({ icon, label, active, onClick }: Props) {
  return (
    <button
      onClick={onClick}
      className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
        active
          ? "bg-bg-hover text-text-primary"
          : "text-text-secondary hover:bg-bg-hover hover:text-text-primary"
      }`}
    >
      <span className="flex-shrink-0">{icon}</span>
      <span className="truncate">{label}</span>
    </button>
  );
}
