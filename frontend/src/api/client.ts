const BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api/v1";

export class ApiError extends Error {
  constructor(public status: number, message: string, public body?: unknown) {
    super(message);
  }
}

type RequestOptions = Omit<RequestInit, "body"> & { body?: unknown };

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { body, headers, ...rest } = opts;
  const res = await fetch(`${BASE_URL}${path}`, {
    ...rest,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const text = await res.text();
  const data = text ? safeParse(text) : null;

  if (!res.ok) {
    const message =
      (data && typeof data === "object" && "error" in data && String((data as { error: unknown }).error)) ||
      res.statusText ||
      "Request failed";
    throw new ApiError(res.status, message, data);
  }

  return data as T;
}

function safeParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

export const api = {
  get: <T>(path: string, opts?: RequestOptions) => request<T>(path, { ...opts, method: "GET" }),
  post: <T>(path: string, body?: unknown, opts?: RequestOptions) =>
    request<T>(path, { ...opts, method: "POST", body }),
  delete: <T>(path: string, opts?: RequestOptions) => request<T>(path, { ...opts, method: "DELETE" }),
};
