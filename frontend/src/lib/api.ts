import axios from "axios";

const isBrowser = typeof window !== "undefined";

export const API_URL = (() => {
  if (process.env.NEXT_PUBLIC_API_URL) return process.env.NEXT_PUBLIC_API_URL;

  if (!isBrowser) {
    console.error("api.ts can not be used in SSR!");
    return "http://localhost:12345";
  }

  // In development, Next.js typically runs on 3000, while Go API runs on 12345.
  // In production, they are often served from the same origin.
  if (window.location.port === "3000") {
    return `http://${window.location.hostname}:12345`;
  }

  return window.location.origin;
})();

export const GET_WS_URL = () => {
  const url = new URL(API_URL);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.pathname = "/api/ws";
  return url.toString();
};

export const api = axios.create({
  baseURL: API_URL,
});

export const fetcher = (url: string) => api.get(url).then((res) => res.data);

// when need stream respone, use apiStream instead of api object as axios does not support stream response well
export async function apiStream(
  path: string,
  body?: unknown,
  options?: {
    method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
    headers?: Record<string, string>;
  },
): Promise<Response> {
  const method = options?.method ?? "POST";

  const response = await fetch(`${API_URL}${path}`, {
    method,
    headers: { "Content-Type": "application/json", ...options?.headers },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    throw new Error(
      `Stream request failed: ${response.status} ${response.statusText}`,
    );
  }

  if (!response.body) throw new Error("No readable stream in response");

  return response;
}
