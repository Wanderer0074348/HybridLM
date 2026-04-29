import { api } from "./client";
import type {
  ChatRequest,
  ChatResponse,
  ChatSession,
  SessionListResponse,
} from "@/types";

export const chatApi = {
  send: (req: ChatRequest) => api.post<ChatResponse>("/chat", req),
  listSessions: () => api.get<SessionListResponse>("/chat/sessions"),
  getSession: (id: string) => api.get<ChatSession>(`/chat/sessions/${id}`),
  deleteSession: (id: string) =>
    api.delete<{ message: string }>(`/chat/sessions/${id}`),
};
