import { useAuth } from "@/context/AuthContext";
import { Avatar } from "@/components/ui/Avatar";
import { Icon } from "@/components/ui/Icon";

export function UserCard() {
  const { user, logout } = useAuth();
  if (!user) return null;

  return (
    <div className="flex items-center gap-3 rounded-xl px-2 py-2 hover:bg-bg-hover transition-colors">
      <Avatar src={user.picture} name={user.name} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-text-primary">{user.name}</p>
        <p className="truncate text-xs text-text-muted">{user.email}</p>
      </div>
      <button
        onClick={() => void logout()}
        className="icon-btn"
        aria-label="Sign out"
        title="Sign out"
      >
        <Icon.Logout size={16} />
      </button>
    </div>
  );
}
