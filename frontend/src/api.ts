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
  UserRole,
  RequestMessage,
  ResponsibilityRule,
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
  messages: (id: string, signal?: AbortSignal) =>
    call<RequestMessage[]>(`/requests/${id}/messages`, { signal }),
  sendMessage: (id: string, body: FormData) =>
    call<RequestMessage>(`/requests/${id}/messages`, { method: "POST", body }),
  decideResolution: (id: string, body: { solved: boolean; rating?: number; comment: string }) =>
    call<RequestItem>(`/requests/${id}/resolution`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
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
    call<UserApartment[] | null>("/me/apartments", { signal }).then(
      (items) => items ?? [],
    ),
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
  list: (signal?: AbortSignal, queue = "ACTIVE") => {
    const params = new URLSearchParams({ queue });
    if (queue === "VISIT_TODAY") { const start = new Date(); start.setHours(0,0,0,0); const end = new Date(start); end.setDate(end.getDate()+1); params.set("visitStart",start.toISOString()); params.set("visitEnd",end.toISOString()); }
    return call<RequestItem[]>(`/admin/requests?${params}`, { signal });
  },
  organizations: (signal?: AbortSignal) =>
    call<Organization[]>("/organizations", { signal }),
  createOrganization: (name: string) =>
    call<Organization>("/admin/organizations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name }),
    }),
  users: (signal?: AbortSignal) =>
    call<CurrentUser[]>("/admin/users", { signal }),
  updateUserRoles: (id: string, roles: UserRole[]) =>
    call<CurrentUser>(`/admin/users/${id}/roles`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ roles }),
    }),
  updateUserOrganization: (id: string, organizationId: string) =>
    call<CurrentUser>(`/admin/users/${id}/organization`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ organizationId }),
    }),
  get: (id: string, signal?: AbortSignal) =>
    call<RequestItem>(`/admin/requests/${id}`, { signal }),
  history: (id: string, signal?: AbortSignal) =>
    call<HistoryItem[]>(`/admin/requests/${id}/history`, { signal }),
  messages: (id: string, signal?: AbortSignal) =>
    call<RequestMessage[]>(`/organization/requests/${id}/messages`, { signal }),
  sendMessage: (id: string, body: FormData) =>
    call<RequestMessage>(`/organization/requests/${id}/messages`, { method: "POST", body }),
  staff: (organizationId?: string) =>
    call<CurrentUser[]>(`/organization/staff${organizationId ? `?organizationId=${encodeURIComponent(organizationId)}` : ""}`),
  updateAssignment: (id: string, body: { assigneeUserId: string; contractorOrganizationId: string; executorContact: string; visitStart?: string; visitEnd?: string }) =>
    call<RequestItem>(`/organization/requests/${id}/assignment`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
  setStatus: (id: string, status: Status, comment: string) =>
    call<RequestItem>(`/admin/requests/${id}/status`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status, comment }),
    }),
};

export const routingAdminApi = {
  rules: () => call<ResponsibilityRule[]>("/admin/responsibility-rules"),
  houses: () => call<House[]>("/admin/houses"),
  organizationTypes: () => call<{ code: string; name: string }[]>("/admin/organization-types"),
  saveRule: (id: string | undefined, body: Omit<ResponsibilityRule, "id" | "organizationName">) =>
    call<ResponsibilityRule>(id ? `/admin/responsibility-rules/${id}` : "/admin/responsibility-rules", { method: id ? "PATCH" : "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
  saveOrganization: (id: string | undefined, body: Partial<Organization>) =>
    call<Organization>(id ? `/admin/organizations/${id}` : "/admin/organizations", { method: id ? "PATCH" : "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
  route: (requestId: string, body: { primaryOrganizationId: string; contractorOrganizationId?: string; reason: string }) =>
    call<RequestItem>(`/admin/requests/${requestId}/route`, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
};
