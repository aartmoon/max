import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { adminApi } from "../api";
import {
  kinds,
  statuses,
  date,
  type RequestItem,
  type Organization,
} from "../types";
import { ErrorMessage, Loading, StatusBadge } from "../components/UI";
export default function Admin() {
  const [items, setItems] = useState<RequestItem[] | null>(null),
    [organizations, setOrganizations] = useState<Organization[]>([]),
    [error, setError] = useState(""),
    [retry, setRetry] = useState(0);
  const [organization, setOrganization] = useState(""),
    [kind, setKind] = useState(""),
    [status, setStatus] = useState("");
  useEffect(() => {
    const c = new AbortController();
    setError("");
    Promise.all([adminApi.list(c.signal), adminApi.organizations(c.signal)])
      .then(([r, o]) => {
        setItems(r);
        setOrganizations(o);
      })
      .catch((e) => {
        if (!c.signal.aborted) setError(e.message);
      });
    return () => c.abort();
  }, [retry]);
  const active = (r: RequestItem) =>
    !["RESOLVED", "REJECTED"].includes(r.status);
  const scoped = (items ?? []).filter(
    (r) => !organization || r.responsibleOrganizationId === organization,
  );
  const filtered = scoped
    .filter(
      (r) => (!kind || r.kind === kind) && (!status || r.status === status),
    )
    .sort(
      (a, b) =>
        Number(b.kind === "EMERGENCY" && active(b)) -
          Number(a.kind === "EMERGENCY" && active(a)) ||
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    );
  return (
    <>
      <Link className="back" to="/">
        ← В приложение жителя
      </Link>
      <div className="eyebrow">КАБИНЕТ ОРГАНИЗАЦИИ</div>
      <div className="page-title">
        <h1>Заявки жителей</h1>
        <button className="text-button" onClick={() => setRetry(retry + 1)}>
          Обновить
        </button>
      </div>
      <p className="intro">
        Принимайте заявки, управляйте работами и сообщайте жителям о результате.
      </p>
      <p className="admin-notice">
        Демо-кабинет УК и подрядчиков · без авторизации. Доступны все тестовые
        организации.
      </p>
      {error ? (
        <ErrorMessage message={error} />
      ) : !items ? (
        <Loading />
      ) : (
        <>
          <div className="stat-grid">
            <div>
              <strong>{scoped.filter(active).length}</strong>
              <span>Активные</span>
            </div>
            <div className="urgent-stat">
              <strong>
                {
                  scoped.filter((r) => active(r) && r.kind === "EMERGENCY")
                    .length
                }
              </strong>
              <span>Экстренные</span>
            </div>
            <div>
              <strong>
                {scoped.filter((r) => r.status === "RESOLVED").length}
              </strong>
              <span>Решены</span>
            </div>
          </div>
          <section className="panel admin-filters">
            <label>
              Организация
              <select
                value={organization}
                onChange={(e) => setOrganization(e.target.value)}
              >
                <option value="">Все организации</option>
                {organizations.map((o) => (
                  <option key={o.id} value={o.id}>
                    {o.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Тип заявки
              <select value={kind} onChange={(e) => setKind(e.target.value)}>
                <option value="">Все типы</option>
                {Object.entries(kinds).map(([key, label]) => (
                  <option key={key} value={key}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Статус
              <select
                value={status}
                onChange={(e) => setStatus(e.target.value)}
              >
                <option value="">Все статусы</option>
                {Object.entries(statuses).map(([key, label]) => (
                  <option key={key} value={key}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
          </section>
          <div className="row list-summary">
            <span>Найдено: {filtered.length}</span>
            <small>Активные экстренные — в начале списка</small>
          </div>
          {filtered.length === 0 ? (
            <section className="panel empty">
              <h2>Заявок не найдено</h2>
              <p>Измените фильтры или дождитесь новых заявок.</p>
            </section>
          ) : (
            <div className="request-list">
              {filtered.map((r) => (
                <Link
                  className={`panel request-card ${r.kind === "EMERGENCY" && active(r) ? "urgent-card" : ""}`}
                  key={r.id}
                  to={`/admin/requests/${r.id}`}
                >
                  <div className="row">
                    <span
                      className={
                        r.kind === "EMERGENCY" ? "urgent-label" : "request-kind"
                      }
                    >
                      № {r.id} · {kinds[r.kind]}
                    </span>
                    <StatusBadge status={r.status} />
                  </div>
                  <h2>{r.description}</h2>
                  <p>{r.address}</p>
                  <p>{r.responsibleOrganization}</p>
                  <div className="row">
                    <small>Срок: до {date(r.deadline)}</small>
                    <span>Открыть заявку →</span>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </>
      )}
    </>
  );
}
