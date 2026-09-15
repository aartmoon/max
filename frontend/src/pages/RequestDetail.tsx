import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api } from "../api";
import {
  date,
  statuses,
  categories,
  kinds,
  type RequestItem,
  type HistoryItem,
} from "../types";
import { ErrorMessage, Loading, StatusBadge } from "../components/UI";
export default function RequestDetail() {
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
      api.get(id, c.signal),
      api.history(id, c.signal),
      api.config(),
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
  }, [id, retry]);
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
      <Link className="back" to="/requests">
        ← Мои обращения
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
          <h1>Обращение сохранено</h1>
          <p className="intro">
            Создано {date(item.createdAt)}. Все изменения появятся здесь.
          </p>
          <section className="panel detail">
            <div className="row">
              <h2>Детали обращения</h2>
              <StatusBadge status={item.status} />
            </div>
            <dl>
              <dt>Проблема / описание</dt>
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
                href={`/api/requests/${id}/photo`}
                target="_blank"
                rel="noreferrer"
              >
                <img
                  className="attachment"
                  src={`/api/requests/${id}/photo`}
                  alt="Фото к обращению"
                />
              </a>
            )}
          </section>
          <section className="panel">
            <h2>Текст обращения</h2>
            <p className="request-text">{item.text}</p>
          </section>
          <section className="panel">
            <h2>История статусов</h2>
            <ol className="timeline">
              {history.map((h) => (
                <li key={h.id}>
                  <strong>{statuses[h.status]}</strong>
                  <small>{new Date(h.createdAt).toLocaleString("ru-RU")}</small>
                </li>
              ))}
            </ol>
          </section>
          {mock && (
            <section className="demo-controls">
              <span className="eyebrow">ДЛЯ ДЕМОНСТРАЦИИ</span>
              <p>Имитируйте обновление от ответственной организации.</p>
              {item.status === "RESOLVED" || item.status === "REJECTED" ? (
                <p>Обращение завершено.</p>
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
