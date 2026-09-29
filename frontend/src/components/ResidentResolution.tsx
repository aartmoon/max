import { useState, type FormEvent } from "react";
import { api } from "../api";
import type { RequestItem } from "../types";
import { ErrorMessage } from "./UI";

export default function ResidentResolution({ item, onUpdated }: { item: RequestItem; onUpdated: () => Promise<void> }) {
  const [solved, setSolved] = useState(true), [rating, setRating] = useState(5), [comment, setComment] = useState(""), [files, setFiles] = useState<File[]>([]), [busy, setBusy] = useState(false), [error, setError] = useState("");
  if (item.status !== "RESOLVED") return null;
  async function submit(event: FormEvent) { event.preventDefault(); if (busy) return; if (!solved && !comment.trim()) { setError("Опишите, почему проблема не решена"); return; } setBusy(true); setError(""); try { if (files.length) { const body = new FormData(); body.set("type","RESIDENT_PUBLIC"); body.set("text","Повторные фотографии результата"); files.forEach((file)=>body.append("attachments",file)); await api.sendMessage(item.id,body); } await api.decideResolution(item.id, { solved, rating: solved ? rating : undefined, comment: comment.trim() }); await onUpdated(); } catch (e) { setError((e as Error).message); } finally { setBusy(false); } }
  return <section className="panel resolution-panel"><h2>Подтвердите результат</h2><p>Организация сообщила о выполнении работ. Проверьте результат.</p><form className="form" onSubmit={submit}><fieldset disabled={busy}>
    <div className="decision-buttons"><button type="button" className={solved ? "button primary" : "button"} onClick={() => setSolved(true)}>Проблема решена</button><button type="button" className={!solved ? "button danger-button" : "button"} onClick={() => setSolved(false)}>Проблема не решена</button></div>
    {solved && <label>Оценка<select value={rating} onChange={(event) => setRating(Number(event.target.value))}>{[5,4,3,2,1].map((value) => <option key={value} value={value}>{value}</option>)}</select></label>}
    <label>Комментарий {!solved && "*"}<textarea rows={3} maxLength={2000} value={comment} onChange={(event) => setComment(event.target.value)}/></label>
    <label>Повторные фотографии<input type="file" multiple accept="image/jpeg,image/png,image/webp" onChange={(event)=>setFiles(Array.from(event.target.files ?? []).slice(0,5))}/></label>
    {error && <ErrorMessage message={error}/>}<button className="button primary" type="submit">{busy ? "Сохраняем…" : solved ? "Подтвердить решение" : "Вернуть в работу"}</button>
  </fieldset></form></section>;
}
