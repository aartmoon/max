export type Status =
  "CREATED" | "SENT" | "ACCEPTED" | "IN_PROGRESS" | "RESOLVED" | "REJECTED";
export type Kind =
  "PROBLEM" | "APPLICATION" | "QUESTION" | "EMERGENCY" | "COMPLAINT";
export interface RequestItem {
  id: string;
  userId: string;
  houseId: string;
  description: string;
  problemType: string;
  responsibleOrganizationId: string;
  responsibleOrganization: string;
  status: Status;
  deadline: string;
  createdAt: string;
  address: string;
  houseObjectId: string;
  houseObjectGuid?: string;
  apartmentObjectId?: string;
  apartmentObjectGuid?: string;
  kind: Kind;
  text: string;
  hasPhoto: boolean;
}
export type AddressKind =
  | "address_object"
  | "house"
  | "apartment"
  | "room"
  | "carplace"
  | "stead";
export interface AddressSuggestion {
  objectId: string;
  objectGuid?: string;
  parentObjectId?: string;
  objectKind: AddressKind;
  displayName: string;
  fullAddress: string;
}
export interface HistoryItem {
  id: number;
  requestId: string;
  status: Status;
  createdAt: string;
  comment: string;
  actor: string;
}
export const statuses: Record<Status, string> = {
  CREATED: "Создано",
  SENT: "Отправлено",
  ACCEPTED: "Принято",
  IN_PROGRESS: "В работе",
  RESOLVED: "Решено",
  REJECTED: "Отклонено",
};
export const kinds: Record<Kind, string> = {
  PROBLEM: "Проблема",
  APPLICATION: "Запрос",
  QUESTION: "Вопрос",
  EMERGENCY: "Экстренно",
  COMPLAINT: "Жалоба",
};
export const categories: Record<string, string> = {
  PIPE_LEAK: "Протечка трубы",
  ELEVATOR: "Лифт",
  HEATING: "Отопление",
  OTHER: "Другое",
};
export const date = (s: string) =>
  new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "long" }).format(
    new Date(s),
  );

export interface House {
  id: string;
  address: string;
  totalArea: number;
  livingArea: number;
  floors: number;
  entrances: number;
  apartments: number;
  yearBuilt: number;
  organization: string;
  manager: string;
  contact: string;
}
export interface Organization {
  id: string;
  name: string;
}
export const adminTransitions: Partial<Record<Status, Status[]>> = {
  CREATED: ["ACCEPTED", "REJECTED"],
  SENT: ["ACCEPTED", "REJECTED"],
  ACCEPTED: ["IN_PROGRESS", "REJECTED"],
  IN_PROGRESS: ["RESOLVED", "REJECTED"],
};
