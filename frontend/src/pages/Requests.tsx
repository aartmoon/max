import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";
import { date, kinds, type RequestItem } from "../types";
import { Back, Loading, ErrorMessage, StatusBadge } from "../components/UI";
export default function Requests() {
  const [items, setItems] = useState<RequestItem[] | null>(null),
    [error, setError] = useState(""),
    [retry, setRetry] = useState(0);
  useEffect(() => {
    const c = new AbortController();
    setError("");
    api
      .list(c.signal)
      .then(setItems)
      .catch((e) => {
        if (!c.signal.aborted) setError(e.message);
      });
    return () => c.abort();
  }, [retry]);
  return (
    <>
      <Back />
      <div className="page-title">
        <h1>Мои заявки</h1>
        <Link className="text-button" to="/requests/new">
          + Новая заявка
        </Link>
      </div>
      <p className="intro">Вся история решения вопросов вашего дома.</p>
      {error ? (
        <>
          <ErrorMessage message={error} />
          <button className="button" onClick={() => setRetry(retry + 1)}>
            Повторить
          </button>
        </>
      ) : !items ? (
        <Loading />
      ) : items.length === 0 ? (
        <div className="panel empty">
          <span>⌂</span>
          <h2>Пока здесь тихо</h2>
          <p>Ваша первая заявка появится здесь.</p>
          <Link className="button primary" to="/requests/new">
            Создать заявку
          </Link>
        </div>
      ) : (
        <div className="request-list">
          {items.map((r) => (
            <Link
              className="panel request-card"
              key={r.id}
              to={`/requests/${r.id}`}
            >
              <div className="row">
                <small>
                  № {r.id} · {kinds[r.kind]}
                </small>
                <StatusBadge status={r.status} />
              </div>
              <h2>{r.description}</h2>
              <p>{r.address}</p>
              <div className="row">
                <small>{date(r.createdAt)}</small>
                <span>Подробнее →</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </>
  );
}
