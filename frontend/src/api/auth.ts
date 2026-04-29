import { api } from "./client";
import type { User } from "@/types";

export const authApi = {
  loginUrl: () => api.get<{ url: string }>("/auth/login"),
  me: () => api.get<{ user: User }>("/auth/me"),
  logout: () => api.post<{ message: string }>("/auth/logout"),
};
