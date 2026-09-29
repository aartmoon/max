import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import type { AdminFilters, AdminSort, QueueFilter } from "../admin";
import { adminApi } from "../api";
import { useAuth } from "../auth";
import AdminPagination from "../components/AdminPagination";
import AdminRequestFilters from "../components/AdminRequestFilters";
import AdminRequestList from "../components/AdminRequestList";
import AdminRequestSummary from "../components/AdminRequestSummary";
import { ErrorMessage, Loading } from "../components/UI";
import type { AdminRequestPage, Kind, Organization, Status } from "../types";

const queues:{value:QueueFilter;label:string}[]=[{value:"active",label:"Активные"},{value:"new",label:"Новые"},{value:"inWork",label:"В работе"},{value:"unassigned",label:"Без исполнителя"},{value:"mine",label:"Мои"},{value:"visitToday",label:"Визит сегодня"},{value:"overdue",label:"Просроченные"},{value:"done",label:"Завершённые"},{value:"all",label:"Все"}];
const queueSet=new Set(queues.map(x=>x.value));
function read(p:URLSearchParams):AdminFilters{const queue=p.get("queue") as QueueFilter, sort=p.get("sort") as AdminSort, size=Number(p.get("pageSize"));return{query:p.get("q")||"",organization:p.get("organization")||"",kind:(p.get("kind")||"") as Kind|"",status:(p.get("status")||"") as Status|"",queue:queueSet.has(queue)?queue:"active",sort:["priority","newest","deadline"].includes(sort)?sort:"priority",page:Math.max(1,Number(p.get("page"))||1),pageSize:([20,50,100].includes(size)?size:20) as 20|50|100}}
export default function Admin(){
  const {user}=useAuth(),[params,setParams]=useSearchParams(),filters=read(params),isAdmin=user.roles.includes("admin");
  const [data,setData]=useState<AdminRequestPage|null>(null),[organizations,setOrganizations]=useState<Organization[]>([]),[error,setError]=useState(""),[loading,setLoading]=useState(true),[refresh,setRefresh]=useState(0);
  const update=(patch:Partial<AdminFilters>)=>{const next={...filters,...patch};if(Object.keys(patch).some(k=>k!=="page"))next.page=1;const p=new URLSearchParams();if(next.query)p.set("q",next.query);if(next.organization)p.set("organization",next.organization);if(next.kind)p.set("kind",next.kind);if(next.status)p.set("status",next.status);if(next.queue!=="active")p.set("queue",next.queue);if(next.sort!=="priority")p.set("sort",next.sort);if(next.page!==1)p.set("page",String(next.page));if(next.pageSize!==20)p.set("pageSize",String(next.pageSize));setParams(p,{replace:true})};
  useEffect(()=>{const c=new AbortController();setLoading(true);setError("");Promise.all([adminApi.list(filters,c.signal),adminApi.organizations(c.signal)]).then(([page,orgs])=>{if(c.signal.aborted)return;setData(page);setOrganizations(orgs);if(!page.items.length&&page.total>0&&filters.page>1)update({page:Math.ceil(page.total/page.pageSize)})}).catch(e=>{if(!c.signal.aborted)setError(e.message)}).finally(()=>{if(!c.signal.aborted)setLoading(false)});return()=>c.abort()},[params.toString(),refresh]);
  return <div className="admin-page" aria-busy={loading}>
    <Link className="back" to="/">← В приложение жителя</Link>
    <section className="admin-hero"><div><div className="eyebrow">КАБИНЕТ УПРАВЛЯЮЩЕЙ КОМПАНИИ</div><h1>Очередь обращений</h1><p>Поиск, приоритеты и сроки в одном рабочем окне.</p></div><div className="admin-refresh-wrap">{isAdmin&&<Link className="button" to="/admin/settings/users">Настройки</Link>}<button className="admin-refresh" type="button" onClick={()=>setRefresh(x=>x+1)}>↻ Обновить</button></div></section>
    {error&&<div className="admin-inline-error"><ErrorMessage message={error}/><button type="button" onClick={()=>setRefresh(x=>x+1)}>Повторить</button></div>}
    {!data&&!error?<Loading/>:data&&<><AdminRequestSummary summary={data.summary} queue={filters.queue} onQueue={queue=>update({queue})}/><nav className="admin-queue-tabs" aria-label="Очереди заявок">{queues.map(q=><button type="button" key={q.value} className={filters.queue===q.value?"active":""} onClick={()=>update({queue:q.value})}>{q.label}</button>)}</nav><AdminRequestFilters filters={filters} organizations={organizations} onChange={update} onReset={()=>setParams({}, {replace:true})}/><AdminRequestList items={data.items}/><AdminPagination page={data.page} pageSize={data.pageSize} total={data.total} onPage={page=>update({page})} onPageSize={pageSize=>update({pageSize})}/></>}
  </div>;
}
