export type Status =
  "CREATED" | "SENT" | "ACCEPTED" | "IN_PROGRESS" | "RESOLVED" | "REJECTED";
export type Kind = "PROBLEM" | "APPLICATION" | "QUESTION";
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
  kind: Kind;
  text: string;
  hasPhoto: boolean;
}
export interface HistoryItem {
  id: number;
  requestId: string;
  status: Status;
  createdAt: string;
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
  PROBLEM: "Проблема дома",
  APPLICATION: "Заявка в УК",
  QUESTION: "Вопрос",
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
