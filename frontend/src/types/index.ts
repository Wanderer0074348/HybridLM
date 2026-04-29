export interface User {
  id: string;
  email: string;
  name: string;
  picture: string;
  email_verified: boolean;
  created_at: string;
  updated_at: string;
}

export interface CostMetrics {
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  cost: number;
  cache_cost: number;
  total_cost: number;
  estimated_savings: number;
  model: string;
}

export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
  timestamp: string;
}

export interface ChatSession {
  session_id: string;
  messages: ChatMessage[];
  created_at: string;
  last_interaction: string;
  total_tokens: number;
  message_count: number;
  model_preference: string;
}

export interface SessionMetadata {
  session_id: string;
  title: string;
  last_interaction: string;
  message_count: number;
  created_at: string;
}

export interface ChatResponse {
  session_id: string;
  response: string;
  model_used: string;
  routing_reason: string;
  latency: number;
  cache_hit: boolean;
  timestamp: string;
  message_count: number;
  cost_metrics?: CostMetrics;
}

export interface ChatRequest {
  session_id?: string;
  message: string;
  max_tokens?: number;
  temperature?: number;
}

export interface SessionListResponse {
  sessions: SessionMetadata[] | string[];
  count: number;
}

export interface MessageMeta {
  model_used?: string;
  routing_reason?: string;
  latency?: number;
  cache_hit?: boolean;
  cost_metrics?: CostMetrics;
}

export type ChatViewMessage = ChatMessage & { meta?: MessageMeta; pending?: boolean };
