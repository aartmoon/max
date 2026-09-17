import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  deadlineInfo,
  filterAdminRequests,
  formatAdminDateTime,
  isEmergency,
  isOverdue,
  isTerminal,
  type AdminFilters,
  type AdminSort,
  type QueueFilter,
} from "../admin";
import { adminApi } from "../api";
import { ErrorMessage, Loading, StatusBadge } from "../components/UI";
import {
  categories,
  kinds,
  statuses,
  type Kind,
  type Organization,
  type RequestItem,
  type Status,
} from "../types";

const queueOptions: { value: QueueFilter; label: string }[] = [
  { value: "active", label: "Активные" },
  { value: "new", label: "Новые" },
  { value: "inWork", label: "В работе" },
  { value: "overdue", label: "Просроченные" },
  { value: "done", label: "Завершённые" },
  { value: "all", label: "Все" },
];

const validQueues = new Set(queueOptions.map(({ value }) => value));
const validSorts = new Set<AdminSort>(["priority", "newest", "deadline"]);

export default function Admin() {
  const [items, setItems] = useState<RequestItem[] | null>(null);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [refresh, setRefresh] = useState(0);
  const [updatedAt, setUpdatedAt] = useState<Date | null>(null);
  const [searchParams, setSearchParams] = useSearchParams();

  const rawQueue = searchParams.get("queue") ?? "active";
  const rawSort = searchParams.get("sort") ?? "priority";
  const filters: AdminFilters = {
    query: searchParams.get("q") ?? "",
    organization: searchParams.get("organization") ?? "",
    kind: (searchParams.get("kind") ?? "") as Kind | "",
    status: (searchParams.get("status") ?? "") as Status | "",
    queue: validQueues.has(rawQueue as QueueFilter)
      ? (rawQueue as QueueFilter)
      : "active",
    sort: validSorts.has(rawSort as AdminSort)
      ? (rawSort as AdminSort)
      : "priority",
  };

  function updateFilters(patch: Partial<AdminFilters>) {
    const next = new URLSearchParams(searchParams);
    Object.entries(patch).forEach(([key, value]) => {
      const parameter = key === "query" ? "q" : key;
      const defaultValue =
        (key === "queue" && value === "active") ||
        (key === "sort" && value === "priority");
      if (!value || defaultValue) next.delete(parameter);
      else next.set(parameter, value);
    });
    setSearchParams(next, { replace: true });
  }

  function resetFilters() {
    setSearchParams({}, { replace: true });
  }

  useEffect(() => {
    const controller = new AbortController();
    setError("");
    setLoading(true);
    Promise.all([
      adminApi.list(controller.signal),
      adminApi.organizations(controller.signal),
    ])
      .then(([requests, organizationList]) => {
        if (controller.signal.aborted) return;
        setItems(requests);
        setOrganizations(organizationList);
        setUpdatedAt(new Date());
      })
      .catch((reason) => {
        if (!controller.signal.aborted) setError(reason.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [refresh]);

  const now = new Date();
  const organizationItems = (items ?? []).filter(
    (request) =>
      !filters.organization ||
      request.responsibleOrganizationId === filters.organization,
  );
  const stats = {
    active: organizationItems.filter((request) => !isTerminal(request)).length,
    overdue: organizationItems.filter((request) => isOverdue(request, now))
      .length,
    emergency: organizationItems.filter(isEmergency).length,
    done: organizationItems.filter(isTerminal).length,
  };
  const visibleItems = useMemo(
    () => filterAdminRequests(items ?? [], filters, now),
    [items, searchParams.toString()],
  );
  const hasFilters = Boolean(
    filters.query ||
      filters.organization ||
      filters.kind ||
      filters.status ||
      filters.queue !== "active" ||
      filters.sort !== "priority",
  );

  return (
    <div className="admin-page">
      <Link className="back" to="/">
        ← В приложение жителя
      </Link>

      <section className="admin-hero">
        <div>
          <div className="eyebrow">КАБИНЕТ УПРАВЛЯЮЩЕЙ КОМПАНИИ</div>
          <h1>Очередь обращений</h1>
          <p>
            Сначала показываем экстренные и просроченные заявки, чтобы важное не
            потерялось в общем потоке.
          </p>
        </div>
        <div className="admin-refresh-wrap">
          <button
            className="admin-refresh"
            type="button"
            disabled={loading}
            onClick={() => setRefresh((value) => value + 1)}
          >
            <span aria-hidden="true">↻</span>
            {loading ? "Обновляем…" : "Обновить"}
          </button>
          {updatedAt && (
            <small>Данные на {formatAdminDateTime(updatedAt)}</small>
          )}
        </div>
      </section>

      <p className="admin-notice">
        Демо-кабинет без авторизации. Доступны заявки всех тестовых организаций.
      </p>

      {error && <ErrorMessage message={error} />}
      {!items ? (
        !error && <Loading />
      ) : (
        <>
          <section className="admin-stats" aria-label="Сводка по заявкам">
            <button
              className={filters.queue === "active" ? "active" : ""}
              type="button"
              onClick={() => updateFilters({ queue: "active" })}
            >
              <span className="admin-stat-icon neutral" aria-hidden="true">
                ↗
              </span>
              <span>Активные</span>
              <strong>{stats.active}</strong>
              <small>требуют внимания</small>
            </button>
            <button
              className={filters.queue === "overdue" ? "active" : ""}
              type="button"
              onClick={() => updateFilters({ queue: "overdue" })}
            >
              <span className="admin-stat-icon danger" aria-hidden="true">
                !
              </span>
              <span>Просрочены</span>
              <strong>{stats.overdue}</strong>
              <small>срок уже вышел</small>
            </button>
            <button
              className={filters.queue === "emergency" ? "active" : ""}
              type="button"
              onClick={() => updateFilters({ queue: "emergency" })}
            >
              <span className="admin-stat-icon urgent" aria-hidden="true">
                ⚡
              </span>
              <span>Экстренные</span>
              <strong>{stats.emergency}</strong>
              <small>активные сейчас</small>
            </button>
            <button
              className={filters.queue === "done" ? "active" : ""}
              type="button"
              onClick={() => updateFilters({ queue: "done" })}
            >
              <span className="admin-stat-icon success" aria-hidden="true">
                ✓
              </span>
              <span>Завершены</span>
              <strong>{stats.done}</strong>
              <small>решены или закрыты</small>
            </button>
          </section>

          <nav className="admin-queue-tabs" aria-label="Очереди заявок">
            {queueOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                className={filters.queue === option.value ? "active" : ""}
                aria-current={
                  filters.queue === option.value ? "page" : undefined
                }
                onClick={() => updateFilters({ queue: option.value })}
              >
                {option.label}
              </button>
            ))}
          </nav>

          <section className="panel admin-controls">
            <div className="admin-search">
              <span aria-hidden="true">⌕</span>
              <label className="visually-hidden" htmlFor="admin-search">
                Поиск по заявкам
              </label>
              <input
                id="admin-search"
                type="search"
                value={filters.query}
                onChange={(event) =>
                  updateFilters({ query: event.target.value })
                }
                placeholder="Номер, адрес, описание или организация"
              />
            </div>
            <div className="admin-filters">
              <label>
                Организация
                <select
                  value={filters.organization}
                  onChange={(event) =>
                    updateFilters({ organization: event.target.value })
                  }
                >
                  <option value="">Все организации</option>
                  {organizations.map((organization) => (
                    <option key={organization.id} value={organization.id}>
                      {organization.name}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Тип заявки
                <select
                  value={filters.kind}
                  onChange={(event) =>
                    updateFilters({ kind: event.target.value as Kind | "" })
                  }
                >
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
                  value={filters.status}
                  onChange={(event) =>
                    updateFilters({ status: event.target.value as Status | "" })
                  }
                >
                  <option value="">Все статусы</option>
                  {Object.entries(statuses).map(([key, label]) => (
                    <option key={key} value={key}>
                      {label}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Сортировка
                <select
                  value={filters.sort}
                  onChange={(event) =>
                    updateFilters({ sort: event.target.value as AdminSort })
                  }
                >
                  <option value="priority">По приоритету</option>
                  <option value="deadline">По сроку</option>
                  <option value="newest">Сначала новые</option>
                </select>
              </label>
            </div>
            {hasFilters && (
              <button
                className="admin-reset"
                type="button"
                onClick={resetFilters}
              >
                Сбросить фильтры
              </button>
            )}
          </section>

          <div className="admin-list-header" aria-live="polite">
            <div>
              <strong>{visibleItems.length}</strong>
              <span> из {items.length} заявок</span>
            </div>
            <small>
              {filters.sort === "priority"
                ? "Экстренные и просроченные — выше"
                : filters.sort === "deadline"
                  ? "Ближайший срок — выше"
                  : "Последние обращения — выше"}
            </small>
          </div>

          {visibleItems.length === 0 ? (
            <section className="panel empty admin-empty">
              <span aria-hidden="true">⌕</span>
              <h2>Заявок не найдено</h2>
              <p>Попробуйте изменить запрос или сбросить фильтры.</p>
              {hasFilters && (
                <button className="button" type="button" onClick={resetFilters}>
                  Сбросить фильтры
                </button>
              )}
            </section>
          ) : (
            <div className="admin-request-list">
              {visibleItems.map((request) => {
                const deadline = deadlineInfo(request, now);
                const urgent = isEmergency(request);
                const overdue = isOverdue(request, now);
                return (
                  <Link
                    className={`admin-request-card ${urgent ? "urgent-card" : ""}`}
                    key={request.id}
                    to={`/admin/requests/${request.id}`}
                  >
                    <div className="admin-request-main">
                      <div className="admin-request-meta">
                        <span className="request-number">№ {request.id}</span>
                        {urgent && (
                          <span className="priority-label emergency">
                            Экстренная
                          </span>
                        )}
                        {!urgent && overdue && (
                          <span className="priority-label overdue">
                            Просрочена
                          </span>
                        )}
                        <StatusBadge status={request.status} />
                      </div>
                      <h2>{request.description}</h2>
                      <p className="admin-address">{request.address}</p>
                      <div className="admin-request-tags">
                        <span>{kinds[request.kind]}</span>
                        <span>
                          {categories[request.problemType] ??
                            request.problemType}
                        </span>
                        <span>{request.responsibleOrganization}</span>
                      </div>
                    </div>
                    <div className="admin-request-side">
                      <span className={`deadline-pill ${deadline.tone}`}>
                        {deadline.label}
                      </span>
                      <small>
                        Создана {formatAdminDateTime(request.createdAt)}
                      </small>
                      <span className="admin-open-request">Открыть →</span>
                    </div>
                  </Link>
                );
              })}
            </div>
          )}
        </>
      )}
    </div>
  );
}
