import type { SVGProps } from "react";

type IconProps = SVGProps<SVGSVGElement> & { size?: number };

const base = (size: number): SVGProps<SVGSVGElement> => ({
  width: size,
  height: size,
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.75,
  strokeLinecap: "round",
  strokeLinejoin: "round",
});

export const Icon = {
  Plus: ({ size = 18, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <path d="M12 5v14M5 12h14" />
    </svg>
  ),
  Search: ({ size = 18, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" />
    </svg>
  ),
  Chats: ({ size = 18, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <path d="M21 12a8 8 0 0 1-12.6 6.5L4 20l1.5-4.4A8 8 0 1 1 21 12Z" />
    </svg>
  ),
  Send: ({ size = 18, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  ),
  Sidebar: ({ size = 18, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <rect x="3" y="4" width="18" height="16" rx="2" />
      <path d="M9 4v16" />
    </svg>
  ),
  Trash: ({ size = 16, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M6 6l1 14a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-14" />
    </svg>
  ),
  Logout: ({ size = 16, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" />
    </svg>
  ),
  Sparkle: ({ size = 18, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <path d="M12 3v4M12 17v4M3 12h4M17 12h4M5.6 5.6l2.8 2.8M15.6 15.6l2.8 2.8M5.6 18.4l2.8-2.8M15.6 8.4l2.8-2.8" />
    </svg>
  ),
  Bolt: ({ size = 14, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <path d="M13 2 4 14h7l-1 8 9-12h-7l1-8Z" />
    </svg>
  ),
  Cloud: ({ size = 14, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <path d="M17 18a4 4 0 0 0 0-8 6 6 0 0 0-11.5 1.5A4 4 0 0 0 6 18h11Z" />
    </svg>
  ),
  Cache: ({ size = 14, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <ellipse cx="12" cy="6" rx="8" ry="3" />
      <path d="M4 6v6c0 1.7 3.6 3 8 3s8-1.3 8-3V6M4 12v6c0 1.7 3.6 3 8 3s8-1.3 8-3v-6" />
    </svg>
  ),
  Coin: ({ size = 14, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest}>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v10M9 9.5c0-1 1.3-1.7 3-1.7s3 .7 3 1.7-1 1.5-3 1.7-3 .7-3 1.8 1.3 1.8 3 1.8 3-.7 3-1.7" />
    </svg>
  ),
  Logo: ({ size = 22, ...rest }: IconProps) => (
    <svg {...base(size)} {...rest} fill="currentColor" stroke="none">
      <circle cx="12" cy="12" r="10" />
    </svg>
  ),
};
