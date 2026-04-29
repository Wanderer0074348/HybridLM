type AvatarProps = {
  src?: string;
  name: string;
  size?: number;
};

export function Avatar({ src, name, size = 32 }: AvatarProps) {
  const initials = name
    .split(" ")
    .map((p) => p[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();

  return (
    <span
      className="inline-flex items-center justify-center rounded-full bg-accent/15 text-accent text-xs font-semibold overflow-hidden"
      style={{ width: size, height: size }}
    >
      {src ? (
        <img src={src} alt={name} className="h-full w-full object-cover" referrerPolicy="no-referrer" />
      ) : (
        initials || "?"
      )}
    </span>
  );
}
