# HybridLM Frontend

React + TypeScript + Vite + Tailwind frontend for the HybridLM Go backend.

## Setup

```bash
cd frontend
npm install
cp .env.example .env
npm run dev
```

The dev server runs on `http://localhost:3000` and proxies `/api/*` to `http://localhost:8080`.

Backend env vars to set so OAuth redirects land here:

```
FRONTEND_URL=http://localhost:3000
ALLOWED_ORIGINS=http://localhost:3000
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/callback
```

## Structure

```
src/
├── api/          fetch wrapper + per-domain endpoints
├── context/      AuthContext, ChatContext (state + actions)
├── components/
│   ├── auth/     LoginScreen, AuthCallback
│   ├── sidebar/  Sidebar, SessionList, UserCard, SidebarItem
│   ├── chat/     ChatView, MessageList, Message, Composer, EmptyState, MetadataBadges
│   ├── ui/       Icon, Spinner, Avatar
│   └── AppShell.tsx
├── types/        shared TS types mirroring backend models
├── utils/        formatters
├── App.tsx       routes + protection
└── main.tsx
```

## Build

```bash
npm run build
```

Outputs to `dist/`. Serve behind any static host or wire into the Go server.
