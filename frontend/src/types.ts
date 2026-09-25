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
export interface CurrentUser {
  id: string;
  email: string;
  name: string;
  roles: string[];
  organizationId?: string;
}
export interface UserApartment {
  id: string;
  userId: string;
  houseObjectId: string;
  houseObjectGuid?: string;
  apartmentObjectId?: string;
  apartmentObjectGuid?: string;
  address: string;
  label: string;
  isDefault: boolean;
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
  garObjectId: string;
  objectGuid?: string;
  address: string;
  cadastralNumber: string | null;
  totalArea: number | null;
  livingArea: number | null;
  floors: number | null;
  entrances: number | null;
  apartments: number | null;
  yearBuilt: number | null;
  organization: string | null;
  manager: string | null;
  contact: string | null;
  dataSource: string;
  dataUpdatedAt: string | null;
  stale: boolean;
  characteristics?: HouseCharacteristics;
  management?: HouseManagement;
  dataSources?: HouseDataSource[];
}

export interface HouseCharacteristics {
  houseTypeCode: string | null;
  houseType: string | null;
  status: string | null;
  projectSeries: string | null;
  condition: string | null;
  lifecycleStage: string | null;
  yearBuilt: number | null;
  operationYear: number | null;
  reconstructionYear: number | null;
  deteriorationPercent: number | null;
  deteriorationDate: string | null;
  wallMaterial: string | null;
  energyEfficiency: string | null;
  totalArea: number | null;
  livingArea: number | null;
  nonResidentialArea: number | null;
  residentialPremises: number | null;
  residentialPremisesArea: number | null;
  residentialPremisesWithRealty: number | null;
  residentialPremisesWithRealtyArea: number | null;
  nonResidentialPremises: number | null;
  nonResidentialPremisesArea: number | null;
  nonResidentialPremisesNotCommon: number | null;
  nonResidentialPremisesNotCommonArea: number | null;
  floors: number | null;
  entrances: number | null;
  ownersOrShares: number | null;
}

export interface HouseManagement {
  method: string | null;
  organizationGuid: string | null;
  shortName: string | null;
  fullName: string | null;
  address: string | null;
  phone: string | null;
  website: string | null;
  organizationType: string | null;
  registryOrganizationGuid: string | null;
  inn: string | null;
  ogrn: string | null;
  chief: string | null;
  contractStart: string | null;
  contractEnd: string | null;
}

export interface HouseDataSource {
  name: string;
  available: boolean;
  updatedAt: string;
  stale: boolean;
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
