import { useState, type FormEvent } from "react";
import { adminApi } from "../api";
import {
  adminTransitions,
  statuses,
  type RequestItem,
  type Status,
} from "../types";
import { ErrorMessage } from "./UI";

const statusHints: Partial<Record<Status, string>> = {
  ACCEPTED: "Организация подтверждает, что заявка принята в работу.",
  IN_PROGRESS: "Исполнитель уже занимается заявкой.",
  RESOLVED: "Работы завершены. Опишите результат для жителя.",
  REJECTED: "Заявка закрывается без выполнения. Причина обязательна.",
};

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
  const commentRequired = target === "REJECTED" || target === "RESOLVED";

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (busy || !target) return;
    const trimmedComment = comment.trim();
    if (commentRequired && !trimmedComment) {
      setError(
        target === "REJECTED"
          ? "Укажите причину отклонения заявки"
          : "Опишите результат выполненных работ",
      );
      return;
    }
    setBusy(true);
    setError("");
    let statusWasUpdated = false;
    try {
      await adminApi.setStatus(item.id, target, trimmedComment);
      statusWasUpdated = true;
      await onUpdated();
    } catch (e) {
      setError(
        statusWasUpdated
          ? "Статус сохранён, но свежие данные не загрузились. Обновите страницу."
          : (e as Error).message,
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel admin-status-panel">
      <div className="eyebrow">ДЕЙСТВИЯ УПРАВЛЯЮЩЕЙ КОМПАНИИ</div>
      <h2>Обновить статус</h2>
      <div className="current-status-row">
        <span>Сейчас</span>
        <strong>{statuses[item.status]}</strong>
      </div>
      {allowed.length === 0 ? (
        <div className="status-finished">
          <span aria-hidden="true">✓</span>
          <p>Заявка завершена. Результат и история доступны жителю.</p>
        </div>
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
            {target && <p className="status-hint">{statusHints[target]}</p>}
            <label htmlFor="comment">
              Комментарий для жителя{" "}
              {commentRequired ? "*" : "(необязательно)"}
            </label>
            <textarea
              id="comment"
              rows={3}
              required={commentRequired}
              maxLength={2000}
              value={comment}
              onChange={(e) => {
                setComment(e.target.value);
                if (error) setError("");
              }}
              placeholder={
                target === "RESOLVED"
                  ? "Что сделали и когда завершили работы"
                  : target === "REJECTED"
                    ? "Почему заявка не может быть выполнена"
                    : "Например: мастер назначен на завтра, 10:00"
              }
            />
            <div className="comment-help">
              <small>Сообщение появится в истории заявки.</small>
              <small>{comment.length}/2000</small>
            </div>
            {error && <ErrorMessage message={error} />}
            <button
              className={`button full status-submit ${target === "REJECTED" ? "danger-button" : "primary"}`}
              type="submit"
              aria-label="Сохранить статус"
              disabled={commentRequired && !comment.trim()}
            >
              {busy
                ? "Сохраняем…"
                : target
                  ? `Перевести в «${statuses[target]}»`
                  : "Сохранить статус"}
              {!busy && <span aria-hidden="true">→</span>}
            </button>
          </fieldset>
        </form>
      )}
    </section>
  );
}
