import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  type ReactNode,
} from "react";
import { chatApi } from "@/api";
import type {
  ChatViewMessage,
  SessionMetadata,
} from "@/types";

type State = {
  sessions: SessionMetadata[];
  sessionsLoading: boolean;
  activeSessionId: string | null;
  messages: ChatViewMessage[];
  messagesLoading: boolean;
  sending: boolean;
  error: string | null;
};

type Action =
  | { type: "sessions/loading" }
  | { type: "sessions/loaded"; sessions: SessionMetadata[] }
  | { type: "sessions/error"; error: string }
  | { type: "select"; id: string | null }
  | { type: "messages/loading" }
  | { type: "messages/loaded"; messages: ChatViewMessage[] }
  | { type: "messages/append"; message: ChatViewMessage }
  | { type: "messages/replaceLast"; message: ChatViewMessage }
  | { type: "send/start" }
  | { type: "send/end" }
  | { type: "error"; error: string | null };

const initial: State = {
  sessions: [],
  sessionsLoading: false,
  activeSessionId: null,
  messages: [],
  messagesLoading: false,
  sending: false,
  error: null,
};

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case "sessions/loading":
      return { ...state, sessionsLoading: true, error: null };
    case "sessions/loaded":
      return { ...state, sessions: action.sessions, sessionsLoading: false };
    case "sessions/error":
      return { ...state, sessionsLoading: false, error: action.error };
    case "select":
      return { ...state, activeSessionId: action.id, messages: [], error: null };
    case "messages/loading":
      return { ...state, messagesLoading: true, error: null };
    case "messages/loaded":
      return { ...state, messages: action.messages, messagesLoading: false };
    case "messages/append":
      return { ...state, messages: [...state.messages, action.message] };
    case "messages/replaceLast":
      return { ...state, messages: [...state.messages.slice(0, -1), action.message] };
    case "send/start":
      return { ...state, sending: true, error: null };
    case "send/end":
      return { ...state, sending: false };
    case "error":
      return { ...state, error: action.error, sending: false, messagesLoading: false };
    default:
      return state;
  }
}

type ChatContextValue = State & {
  loadSessions: () => Promise<void>;
  selectSession: (id: string | null) => Promise<void>;
  newSession: () => void;
  sendMessage: (text: string) => Promise<void>;
  deleteSession: (id: string) => Promise<void>;
};

const ChatContext = createContext<ChatContextValue | null>(null);

function isMetadataArray(v: SessionMetadata[] | string[]): v is SessionMetadata[] {
  return v.length === 0 || typeof v[0] === "object";
}

export function ChatProvider({ children, enabled }: { children: ReactNode; enabled: boolean }) {
  const [state, dispatch] = useReducer(reducer, initial);
  const activeRef = useRef<string | null>(null);
  activeRef.current = state.activeSessionId;

  const loadSessions = useCallback(async () => {
    if (!enabled) return;
    dispatch({ type: "sessions/loading" });
    try {
      const res = await chatApi.listSessions();
      const list = isMetadataArray(res.sessions) ? res.sessions : [];
      dispatch({ type: "sessions/loaded", sessions: list });
    } catch (err) {
      dispatch({ type: "sessions/error", error: (err as Error).message });
    }
  }, [enabled]);

  const selectSession = useCallback(async (id: string | null) => {
    dispatch({ type: "select", id });
    if (!id) return;
    dispatch({ type: "messages/loading" });
    try {
      const session = await chatApi.getSession(id);
      const messages: ChatViewMessage[] = (session.messages || []).map((m) => ({ ...m }));
      dispatch({ type: "messages/loaded", messages });
    } catch (err) {
      dispatch({ type: "error", error: (err as Error).message });
    }
  }, []);

  const newSession = useCallback(() => {
    dispatch({ type: "select", id: null });
  }, []);

  const sendMessage = useCallback(
    async (text: string) => {
      const trimmed = text.trim();
      if (!trimmed) return;

      dispatch({ type: "send/start" });
      const userMsg: ChatViewMessage = {
        role: "user",
        content: trimmed,
        timestamp: new Date().toISOString(),
      };
      const placeholder: ChatViewMessage = {
        role: "assistant",
        content: "",
        timestamp: new Date().toISOString(),
        pending: true,
      };
      dispatch({ type: "messages/append", message: userMsg });
      dispatch({ type: "messages/append", message: placeholder });

      try {
        const res = await chatApi.send({
          session_id: activeRef.current ?? undefined,
          message: trimmed,
        });

        const assistant: ChatViewMessage = {
          role: "assistant",
          content: res.response,
          timestamp: res.timestamp,
          meta: {
            model_used: res.model_used,
            routing_reason: res.routing_reason,
            latency: res.latency,
            cache_hit: res.cache_hit,
            cost_metrics: res.cost_metrics,
          },
        };
        dispatch({ type: "messages/replaceLast", message: assistant });

        if (activeRef.current !== res.session_id) {
          dispatch({ type: "select", id: res.session_id });
        }
        await loadSessions();
      } catch (err) {
        dispatch({
          type: "messages/replaceLast",
          message: {
            role: "assistant",
            content: `Error: ${(err as Error).message}`,
            timestamp: new Date().toISOString(),
          },
        });
        dispatch({ type: "error", error: (err as Error).message });
      } finally {
        dispatch({ type: "send/end" });
      }
    },
    [loadSessions]
  );

  const deleteSession = useCallback(
    async (id: string) => {
      try {
        await chatApi.deleteSession(id);
        if (activeRef.current === id) {
          dispatch({ type: "select", id: null });
        }
        await loadSessions();
      } catch (err) {
        dispatch({ type: "error", error: (err as Error).message });
      }
    },
    [loadSessions]
  );

  useEffect(() => {
    if (enabled) void loadSessions();
  }, [enabled, loadSessions]);

  const value = useMemo<ChatContextValue>(
    () => ({
      ...state,
      loadSessions,
      selectSession,
      newSession,
      sendMessage,
      deleteSession,
    }),
    [state, loadSessions, selectSession, newSession, sendMessage, deleteSession]
  );

  return <ChatContext.Provider value={value}>{children}</ChatContext.Provider>;
}

export function useChat(): ChatContextValue {
  const ctx = useContext(ChatContext);
  if (!ctx) throw new Error("useChat must be used within ChatProvider");
  return ctx;
}
