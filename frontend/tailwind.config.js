/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: {
          base: "#1a1a1a",
          elevated: "#212121",
          hover: "#2a2a2a",
          input: "#2c2c2c",
          sidebar: "#171717",
        },
        border: {
          subtle: "#2e2e2e",
          default: "#3a3a3a",
        },
        text: {
          primary: "#ececec",
          secondary: "#a8a8a8",
          muted: "#6e6e6e",
        },
        accent: {
          DEFAULT: "#d97757",
          hover: "#c46748",
          subtle: "#3a2a23",
        },
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        serif: ["'Tiempos Headline'", "Georgia", "serif"],
        mono: ["'JetBrains Mono'", "ui-monospace", "monospace"],
      },
      animation: {
        "fade-in": "fadeIn 0.2s ease-out",
        "slide-up": "slideUp 0.25s ease-out",
        pulse: "pulse 1.6s cubic-bezier(0.4, 0, 0.6, 1) infinite",
      },
      keyframes: {
        fadeIn: { "0%": { opacity: "0" }, "100%": { opacity: "1" } },
        slideUp: {
          "0%": { opacity: "0", transform: "translateY(8px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
      },
    },
  },
  plugins: [],
};
