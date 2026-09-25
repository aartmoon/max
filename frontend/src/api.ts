import type {
  RequestItem,
  HistoryItem,
  House,
  Organization,
  Status,
  AddressKind,
  AddressSuggestion,
  CurrentUser,
  UserApartment,
} from "./types";
import { apiPath } from "./paths";
async function call<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(apiPath(path), {
      ...init,
      credentials: "same-origin",
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
  config: () => call<{ mockStatusEnabled: boolean }>("/config"),
};
export const authApi = {
  me: (signal?: AbortSignal) => call<CurrentUser>("/me", { signal }),
  requestCode: (email: string) =>
    call<{ ok: boolean }>("/auth/request-code", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email }),
    }),
  verifyCode: (email: string, code: string) =>
    call<CurrentUser>("/auth/verify-code", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, code }),
    }),
  logout: () => call<{ ok: boolean }>("/auth/logout", { method: "POST" }),
};
export const apartmentApi = {
  list: (signal?: AbortSignal) =>
    call<UserApartment[]>("/me/apartments", { signal }),
  create: (body: {
    houseObjectId: string;
    apartmentObjectId?: string;
    label?: string;
    isDefault?: boolean;
  }) =>
    call<UserApartment>("/me/apartments", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  setDefault: (id: string) =>
    call<UserApartment>(`/me/apartments/${id}/default`, { method: "PATCH" }),
  remove: (id: string) =>
    call<{ ok: boolean }>(`/me/apartments/${id}`, { method: "DELETE" }),
};

export const houseApi = {
  resolve: (objectId: string, signal?: AbortSignal) =>
    call<House>("/houses/resolve", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ objectId }),
      signal,
    }),
};
export const addressApi = {
  search: ({
    q,
    kind,
    parentObjectId,
    limit = 10,
    signal,
  }: {
    q: string;
    kind: AddressKind;
    parentObjectId?: string;
    limit?: number;
    signal?: AbortSignal;
  }) => {
    const params = new URLSearchParams({ q, kind, limit: String(limit) });
    if (parentObjectId) params.set("parentObjectId", parentObjectId);
    return call<{ items: AddressSuggestion[] }>(`/addresses/search?${params}`, {
      signal,
    });
  },
};
export const adminApi = {
  list: (signal?: AbortSignal) =>
    call<RequestItem[]>("/admin/requests", { signal }),
  organizations: (signal?: AbortSignal) =>
    call<Organization[]>("/organizations", { signal }),
  createOrganization: (name: string) =>
    call<Organization>("/admin/organizations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name }),
    }),
  get: (id: string, signal?: AbortSignal) =>
    call<RequestItem>(`/admin/requests/${id}`, { signal }),
  history: (id: string, signal?: AbortSignal) =>
    call<HistoryItem[]>(`/admin/requests/${id}/history`, { signal }),
  setStatus: (id: string, status: Status, comment: string) =>
    call<RequestItem>(`/admin/requests/${id}/status`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status, comment }),
    }),
};
