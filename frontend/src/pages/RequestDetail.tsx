import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import AdminStatusForm from "../components/AdminStatusForm";
import { api, adminApi } from "../api";
import { apiPath } from "../paths";
import {
  date,
  statuses,
  categories,
  kinds,
  type RequestItem,
  type HistoryItem,
} from "../types";
import { ErrorMessage, Loading, StatusBadge } from "../components/UI";
export default function RequestDetail({ admin = false }: { admin?: boolean }) {
  const client = admin ? adminApi : api;
  const { id = "" } = useParams();
  const [item, setItem] = useState<RequestItem | null>(null),
    [history, setHistory] = useState<HistoryItem[]>([]),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [mock, setMock] = useState(false),
    [retry, setRetry] = useState(0);
  useEffect(() => {
    const c = new AbortController();
    setItem(null);
    setError("");
    Promise.all([
      client.get(id, c.signal),
      client.history(id, c.signal),
      admin ? Promise.resolve({ mockStatusEnabled: false }) : api.config(),
    ])
      .then(([r, h, config]) => {
        if (!c.signal.aborted) {
          setItem(r);
          setHistory(h);
          setMock(config.mockStatusEnabled);
        }
      })
      .catch((e) => {
        if (!c.signal.aborted) setError(e.message);
      });
    return () => c.abort();
  }, [id, retry, admin]);
  async function next(reject = false) {
    setBusy(true);
    setError("");
    try {
      const r = await api.next(id, reject);
      setItem(r);
      setHistory(await api.history(id));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <Link className="back" to={admin ? "/admin" : "/requests"}>
        {admin ? "← Заявки жителей" : "← Мои заявки"}
      </Link>
      {error && (
        <>
          <ErrorMessage message={error} />
          <button className="text-button" onClick={() => setRetry(retry + 1)}>
            Обновить данные
          </button>
        </>
      )}
      {!item ? (
        !error && <Loading />
      ) : (
        <>
          <div className="eyebrow">
            {kinds[item.kind]} · № {item.id}
          </div>
          <h1>{admin ? `Заявка № ${item.id}` : "Заявка сохранена"}</h1>
          <p className="intro">
            Создана {date(item.createdAt)}. Все изменения появятся здесь.
          </p>
          {admin && (
            <>
              <p className="admin-notice">
                Демонстрационный кабинет организации · изменения видны жителю.
              </p>
              <AdminStatusForm
                key={`${item.id}-${item.status}`}
                item={item}
                onUpdated={async () => {
                  const [r, h] = await Promise.all([
                    adminApi.get(id),
                    adminApi.history(id),
                  ]);
                  setItem(r);
                  setHistory(h);
                }}
              />
            </>
          )}
          <section className="panel detail">
            <div className="row">
              <h2>Детали заявки</h2>
              <StatusBadge status={item.status} />
            </div>
            <dl>
              <dt>Тип заявки</dt>
              <dd className={item.kind === "EMERGENCY" ? "urgent-label" : ""}>
                {kinds[item.kind]}
              </dd>
              <dt>Описание</dt>
              <dd className="description">{item.description}</dd>
              <dt>Категория</dt>
              <dd>
                {categories[item.problemType]} <code>{item.problemType}</code>
              </dd>
              <dt>Адрес</dt>
              <dd>{item.address}</dd>
              <dt>Ответственный</dt>
              <dd>{item.responsibleOrganization}</dd>
              <dt>Срок обработки</dt>
              <dd>
                до {date(item.deadline)}
                <small className="muted">Демо-срок: 3 календарных дня</small>
              </dd>
            </dl>
            {item.hasPhoto && (
              <a
                href={apiPath(
                  `/${admin ? "admin/" : ""}requests/${id}/photo`,
                )}
                target="_blank"
                rel="noreferrer"
              >
                <img
                  className="attachment"
                  src={apiPath(
                    `/${admin ? "admin/" : ""}requests/${id}/photo`,
                  )}
                  alt="Фото к заявке"
                />
              </a>
            )}
          </section>
          <section className="panel">
            <h2>Текст заявки</h2>
            <p className="request-text">{item.text}</p>
          </section>
          <section className="panel">
            <h2>История статусов</h2>
            <ol className="timeline">
              {history.map((h) => (
                <li key={h.id}>
                  <strong>{statuses[h.status]}</strong>
                  <small>
                    {new Date(h.createdAt).toLocaleString("ru-RU")} · {h.actor}
                  </small>
                  {h.comment && <p className="history-comment">{h.comment}</p>}
                </li>
              ))}
            </ol>
          </section>
          {mock && (
            <section className="demo-controls">
              <span className="eyebrow">ДЛЯ ДЕМОНСТРАЦИИ</span>
              <p>Имитируйте обновление от ответственной организации.</p>
              {item.status === "RESOLVED" || item.status === "REJECTED" ? (
                <p>Заявка завершена.</p>
              ) : (
                <div className="demo-buttons">
                  <button
                    className="button primary"
                    disabled={busy}
                    onClick={() => next()}
                  >
                    {busy ? "Обновляем…" : "Следующий статус →"}
                  </button>
                  <button
                    className="button"
                    disabled={busy}
                    onClick={() => next(true)}
                  >
                    Отклонить
                  </button>
                </div>
              )}
            </section>
          )}
        </>
      )}
    </>
  );
}
