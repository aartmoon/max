import { useEffect, useState, type FormEvent } from "react";
import { adminApi } from "../api";
import type { CurrentUser, RequestItem } from "../types";
import { ErrorMessage } from "./UI";

const localInput = (value?: string) => value ? new Date(value).toISOString().slice(0, 16) : "";
export default function RequestAssignment({ item, onUpdated }: { item: RequestItem; onUpdated: () => Promise<void> }) {
  const [staff, setStaff] = useState<CurrentUser[]>([]), [assignee, setAssignee] = useState(item.assignedUserId ?? ""), [contact, setContact] = useState(item.executorContact ?? ""), [start, setStart] = useState(localInput(item.visitStart)), [end, setEnd] = useState(localInput(item.visitEnd)), [busy, setBusy] = useState(false), [error, setError] = useState("");
  useEffect(() => { adminApi.staff(item.primaryOrganizationId).then(setStaff).catch(() => setStaff([])); }, [item.primaryOrganizationId]);
  async function submit(event: FormEvent) { event.preventDefault(); if (busy) return; if ((start && !end) || (!start && end) || (start && end && new Date(start) > new Date(end))) { setError("Проверьте интервал визита"); return; } setBusy(true); setError(""); try { await adminApi.updateAssignment(item.id, { assigneeUserId: assignee, contractorOrganizationId: item.contractorOrganizationId ?? "", executorContact: contact.trim(), visitStart: start ? new Date(start).toISOString() : undefined, visitEnd: end ? new Date(end).toISOString() : undefined }); await onUpdated(); } catch (e) { setError((e as Error).message); } finally { setBusy(false); } }
  return <section className="panel assignment-panel"><h2>Исполнитель и визит</h2><form className="form" onSubmit={submit}><fieldset disabled={busy}>
    <label>Ответственный сотрудник<select value={assignee} onChange={(event) => setAssignee(event.target.value)}><option value="">Без исполнителя</option>{staff.map((person) => <option key={person.id} value={person.id}>{person.name}</option>)}</select></label>
    <label>Контакт исполнителя<input value={contact} maxLength={300} onChange={(event) => setContact(event.target.value)}/></label>
    <label>Начало визита<input type="datetime-local" value={start} onChange={(event) => setStart(event.target.value)}/></label><label>Окончание визита<input type="datetime-local" value={end} onChange={(event) => setEnd(event.target.value)}/></label>
    {error && <ErrorMessage message={error}/>}<button className="button primary" type="submit">{busy ? "Сохраняем…" : "Сохранить назначение"}</button>
  </fieldset></form></section>;
}
