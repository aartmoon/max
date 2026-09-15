import { Link } from "react-router-dom";
import { statuses, type Status } from "../types";
export function StatusBadge({ status }: { status: Status }) {
  return <span className={`badge status-${status}`}>{statuses[status]}</span>;
}
export function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="error" role="alert">
      {message}
    </div>
  );
}
export function Loading() {
  return (
    <p className="loading" role="status">
      Загружаем…
    </p>
  );
}
export function Back() {
  return (
    <Link className="back" to="/">
      ← На главную
    </Link>
  );
}
