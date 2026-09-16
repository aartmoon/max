import { useState, type FormEvent } from "react";
import { adminApi } from "../api";
import {
  adminTransitions,
  statuses,
  type RequestItem,
  type Status,
} from "../types";
import { ErrorMessage } from "./UI";
export default function AdminStatusForm({
  item,
  onUpdated,
}: {
  item: RequestItem;
  onUpdated: () => Promise<void>;
}) {
  const allowed = adminTransitions[item.status] ?? [];
  const [target, setTarget] = useState<Status | "">(allowed[0] ?? ""),
    [comment, setComment] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (busy || !target) return;
    setBusy(true);
    setError("");
    try {
      await adminApi.setStatus(item.id, target, comment);
      await onUpdated();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel">
      <div className="eyebrow">ДЕЙСТВИЯ УПРАВЛЯЮЩЕЙ КОМПАНИИ</div>
      <h2>Обновить статус</h2>
      {allowed.length === 0 ? (
        <p>Заявка завершена. История доступна жителю.</p>
      ) : (
        <form className="form" onSubmit={submit}>
          <fieldset disabled={busy}>
            <label htmlFor="status">Новый статус</label>
            <select
              id="status"
              value={target}
              onChange={(e) => setTarget(e.target.value as Status)}
            >
              {allowed.map((s) => (
                <option key={s} value={s}>
                  {statuses[s]}
                </option>
              ))}
            </select>
            <label htmlFor="comment">
              Комментарий для жителя{" "}
              {target === "REJECTED" ? "*" : "(необязательно)"}
            </label>
            <textarea
              id="comment"
              rows={3}
              required={target === "REJECTED"}
              maxLength={2000}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder="Например: мастер назначен, работы начнутся завтра"
            />
            <small>
              Комментарий появится в истории заявки. При отклонении укажите
              причину.
            </small>
            {error && <ErrorMessage message={error} />}
            <button className="button primary full status-submit" type="submit">
              {busy ? "Сохраняем…" : "Сохранить статус"}
            </button>
          </fieldset>
        </form>
      )}
    </section>
  );
}
