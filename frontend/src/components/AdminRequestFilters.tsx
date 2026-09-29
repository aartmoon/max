import { useEffect, useState } from "react";
import type { AdminFilters, AdminSort } from "../admin";
import { kinds, statuses, type Kind, type Organization, type Status } from "../types";

export default function AdminRequestFilters({ filters, organizations, onChange, onReset }: { filters: AdminFilters; organizations: Organization[]; onChange: (patch: Partial<AdminFilters>) => void; onReset: () => void }) {
  const [draft,setDraft]=useState(filters.query);
  useEffect(()=>setDraft(filters.query),[filters.query]);
  useEffect(()=>{ const id=window.setTimeout(()=>{if(draft!==filters.query) onChange({query:draft})},300); return()=>window.clearTimeout(id)},[draft,filters.query]);
  const active = [filters.query && `Поиск: ${filters.query}`, filters.organization && organizations.find(o=>o.id===filters.organization)?.name, filters.kind && kinds[filters.kind], filters.status && statuses[filters.status]].filter(Boolean) as string[];
  const hasFilters = active.length > 0 || filters.queue !== "active" || filters.sort !== "priority" || filters.pageSize !== 20;
  return <section className="panel admin-controls">
    <label className="admin-search"><span aria-hidden="true">⌕</span><span className="visually-hidden">Поиск по заявкам</span><input aria-label="Поиск по заявкам" type="search" value={draft} onChange={e=>setDraft(e.target.value)} placeholder="Номер, адрес, описание или организация"/></label>
    <details className="admin-filter-disclosure"><summary>Фильтры{active.length?` · ${active.length}`:""}</summary><div className="admin-filters">
      <label>Организация<select value={filters.organization} onChange={e=>onChange({organization:e.target.value})}><option value="">Все организации</option>{organizations.map(o=><option key={o.id} value={o.id}>{o.name}</option>)}</select></label>
      <label>Тип<select value={filters.kind} onChange={e=>onChange({kind:e.target.value as Kind|""})}><option value="">Все типы</option>{Object.entries(kinds).map(([v,l])=><option key={v} value={v}>{l}</option>)}</select></label>
      <label>Статус<select value={filters.status} onChange={e=>onChange({status:e.target.value as Status|""})}><option value="">Все статусы</option>{Object.entries(statuses).map(([v,l])=><option key={v} value={v}>{l}</option>)}</select></label>
      <label>Сортировка<select value={filters.sort} onChange={e=>onChange({sort:e.target.value as AdminSort})}><option value="priority">По приоритету</option><option value="deadline">По сроку</option><option value="newest">Сначала новые</option></select></label>
    </div></details>
    {hasFilters&&<div className="admin-filter-chips">{active.map(x=><span key={x}>{x}</span>)}<button type="button" onClick={onReset}>Сбросить</button></div>}
  </section>;
}
