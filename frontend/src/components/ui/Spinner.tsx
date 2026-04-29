type SpinnerProps = { size?: number; className?: string };

export function Spinner({ size = 16, className = "" }: SpinnerProps) {
  return (
    <span
      role="status"
      aria-label="Loading"
      className={`inline-block animate-spin rounded-full border-2 border-text-muted/40 border-t-text-primary ${className}`}
      style={{ width: size, height: size }}
    />
  );
}
