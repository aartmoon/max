import type { AdminRequestSummary as Summary } from "../types";
import type { QueueFilter } from "../admin";

export default function AdminRequestSummary({ summary, queue, onQueue }: { summary: Summary; queue: QueueFilter; onQueue: (queue: QueueFilter) => void }) {
  const items: [QueueFilter, string, number][] = [["active","Активные",summary.active],["new","Новые",summary.new],["overdue","Просроченные",summary.overdue],["unassigned","Без исполнителя",summary.unassigned],["done","Завершённые",summary.done]];
  return <section className="admin-stats" aria-label="Сводка по заявкам">{items.map(([value,label,count])=><button key={value} className={queue===value?"active":""} type="button" onClick={()=>onQueue(value)}><span>{label}</span><strong>{count}</strong></button>)}</section>;
}
