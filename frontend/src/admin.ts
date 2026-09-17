import {
  categories,
  kinds,
  statuses,
  type Kind,
  type RequestItem,
  type Status,
} from "./types";

export type QueueFilter =
  | "active"
  | "new"
  | "inWork"
  | "overdue"
  | "emergency"
  | "done"
  | "all";

export type AdminSort = "priority" | "newest" | "deadline";

export interface AdminFilters {
  query: string;
  organization: string;
  kind: Kind | "";
  status: Status | "";
  queue: QueueFilter;
  sort: AdminSort;
}

const hour = 60 * 60 * 1000;
const day = 24 * hour;

export const terminalStatuses: Status[] = ["RESOLVED", "REJECTED"];
export const newStatuses: Status[] = ["CREATED", "SENT"];
export const inWorkStatuses: Status[] = ["ACCEPTED", "IN_PROGRESS"];

export const isTerminal = (request: RequestItem) =>
  terminalStatuses.includes(request.status);

export const isOverdue = (request: RequestItem, now = new Date()) =>
  !isTerminal(request) && new Date(request.deadline).getTime() < now.getTime();

export const isEmergency = (request: RequestItem) =>
  request.kind === "EMERGENCY" && !isTerminal(request);

export function matchesQueue(
  request: RequestItem,
  queue: QueueFilter,
  now = new Date(),
) {
  switch (queue) {
    case "active":
      return !isTerminal(request);
    case "new":
      return newStatuses.includes(request.status);
    case "inWork":
      return inWorkStatuses.includes(request.status);
    case "overdue":
      return isOverdue(request, now);
    case "emergency":
      return isEmergency(request);
    case "done":
      return isTerminal(request);
    default:
      return true;
  }
}

function normalize(value: string) {
  return value.toLocaleLowerCase("ru-RU").replaceAll("ё", "е").trim();
}

export function matchesQuery(request: RequestItem, query: string) {
  const normalized = normalize(query);
  if (!normalized) return true;

  const searchable = [
    request.id,
    request.description,
    request.text,
    request.address,
    request.responsibleOrganization,
    categories[request.problemType] ?? request.problemType,
    kinds[request.kind],
    statuses[request.status],
  ]
    .map(normalize)
    .join(" ");

  return normalized
    .split(/\s+/)
    .filter(Boolean)
    .every((part) => searchable.includes(part));
}

function priority(request: RequestItem, now: Date) {
  if (isEmergency(request)) return 5;
  if (isOverdue(request, now)) return 4;
  if (newStatuses.includes(request.status)) return 3;
  if (request.status === "ACCEPTED") return 2;
  if (request.status === "IN_PROGRESS") return 1;
  return 0;
}

export function compareAdminRequests(
  a: RequestItem,
  b: RequestItem,
  sort: AdminSort,
  now = new Date(),
) {
  const aDeadline = new Date(a.deadline).getTime();
  const bDeadline = new Date(b.deadline).getTime();
  const aCreated = new Date(a.createdAt).getTime();
  const bCreated = new Date(b.createdAt).getTime();

  if (sort === "newest") return bCreated - aCreated;
  if (sort === "deadline") {
    const terminalDifference = Number(isTerminal(a)) - Number(isTerminal(b));
    return terminalDifference || aDeadline - bDeadline || bCreated - aCreated;
  }

  return (
    priority(b, now) - priority(a, now) ||
    aDeadline - bDeadline ||
    bCreated - aCreated
  );
}

export function filterAdminRequests(
  requests: RequestItem[],
  filters: AdminFilters,
  now = new Date(),
) {
  return requests
    .filter(
      (request) =>
        (!filters.organization ||
          request.responsibleOrganizationId === filters.organization) &&
        (!filters.kind || request.kind === filters.kind) &&
        (!filters.status || request.status === filters.status) &&
        matchesQueue(request, filters.queue, now) &&
        matchesQuery(request, filters.query),
    )
    .sort((a, b) => compareAdminRequests(a, b, filters.sort, now));
}

function pluralDays(value: number) {
  const mod10 = value % 10;
  const mod100 = value % 100;
  if (mod10 === 1 && mod100 !== 11) return "день";
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return "дня";
  }
  return "дней";
}

export type DeadlineTone = "overdue" | "soon" | "normal" | "closed";

export function deadlineInfo(request: RequestItem, now = new Date()) {
  const deadline = new Date(request.deadline);
  const difference = deadline.getTime() - now.getTime();

  if (isTerminal(request)) {
    return {
      tone: "closed" as DeadlineTone,
      label: `Срок был ${formatAdminDate(deadline)}`,
    };
  }
  if (difference < 0) {
    const elapsedHours = Math.max(1, Math.ceil(Math.abs(difference) / hour));
    if (elapsedHours < 24) {
      return {
        tone: "overdue" as DeadlineTone,
        label: `Просрочено на ${elapsedHours} ч`,
      };
    }
    const elapsedDays = Math.ceil(Math.abs(difference) / day);
    return {
      tone: "overdue" as DeadlineTone,
      label: `Просрочено на ${elapsedDays} ${pluralDays(elapsedDays)}`,
    };
  }
  if (difference <= day) {
    const remainingHours = Math.max(1, Math.ceil(difference / hour));
    return {
      tone: "soon" as DeadlineTone,
      label: `Осталось ${remainingHours} ч`,
    };
  }
  return {
    tone: "normal" as DeadlineTone,
    label: `До ${formatAdminDate(deadline)}`,
  };
}

export function formatAdminDate(value: string | Date) {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "short",
  }).format(new Date(value));
}

export function formatAdminDateTime(value: string | Date) {
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}
