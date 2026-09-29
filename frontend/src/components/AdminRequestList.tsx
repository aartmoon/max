import { Link } from "react-router-dom";
import { deadlineInfo, isEmergency, isOverdue } from "../admin";
import { StatusBadge } from "./UI";
import type { RequestItem } from "../types";

function marker(r:RequestItem,now:Date){if(isEmergency(r))return "Экстренная";if(isOverdue(r,now))return "Просрочена";return ""}
export default function AdminRequestList({items}:{items:RequestItem[]}){
  const now=new Date();
  if(!items.length)return <section className="panel empty admin-empty"><h2>Заявок не найдено</h2><p>Измените поиск или фильтры.</p></section>;
  return <>
    <div className="admin-request-table-wrap"><table className="admin-request-table"><thead><tr><th>№</th><th>Заявка</th><th>Статус</th><th>Исполнитель</th><th>Срок</th></tr></thead><tbody>{items.map(r=><tr key={r.id}><td><Link to={`/admin/requests/${r.id}`}>#{r.id}</Link>{marker(r,now)&&<strong className="request-marker">{marker(r,now)}</strong>}</td><td><Link to={`/admin/requests/${r.id}`}><strong>{r.description}</strong><small>{r.address}</small></Link></td><td><StatusBadge status={r.status}/></td><td>{r.assignedUserName||"Без исполнителя"}</td><td>{deadlineInfo(r,now).label}</td></tr>)}</tbody></table></div>
    <div className="admin-request-cards">{items.map(r=><Link className="admin-request-card" key={r.id} to={`/admin/requests/${r.id}`}><div className="admin-request-meta"><strong>#{r.id}</strong><StatusBadge status={r.status}/></div>{marker(r,now)&&<strong className="request-marker">{marker(r,now)}</strong>}<h2>{r.description}</h2><p>{r.address}</p><small>{r.assignedUserName||"Без исполнителя"} · {deadlineInfo(r,now).label}</small></Link>)}</div>
  </>;
}
