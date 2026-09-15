import type { RequestItem, HistoryItem } from "./types";
async function call<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`/api${path}`, {
      ...init,
      signal: init?.signal ?? AbortSignal.timeout(20000),
    });
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") throw e;
    throw new Error(
      "Нет связи с сервером. Проверьте подключение и попробуйте ещё раз.",
    );
  }
  if (!response.ok) {
    const data = await response.json().catch(() => null);
    throw new Error(
      data?.error ?? "Сервер временно недоступен. Попробуйте ещё раз.",
    );
  }
  return response.json();
}
export const api = {
  list: (signal?: AbortSignal) => call<RequestItem[]>("/requests", { signal }),
  get: (id: string, signal?: AbortSignal) =>
    call<RequestItem>(`/requests/${id}`, { signal }),
  history: (id: string, signal?: AbortSignal) =>
    call<HistoryItem[]>(`/requests/${id}/history`, { signal }),
  create: (body: FormData) =>
    call<RequestItem>("/requests", { method: "POST", body }),
  next: (id: string, reject = false) =>
    call<RequestItem>(
      `/requests/${id}/mock-next-status${reject ? "?reject=true" : ""}`,
      { method: "POST" },
    ),
  config: () => call<{ mockStatusEnabled: boolean }>("/config"),
};
