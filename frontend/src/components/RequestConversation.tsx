import { useEffect, useState, type FormEvent } from "react";
import { api, adminApi } from "../api";
import { apiPath } from "../paths";
import type { RequestMessage } from "../types";
import { ErrorMessage, Loading } from "./UI";

export default function RequestConversation({ requestId, organization = false }: { requestId: string; organization?: boolean }) {
  const client = organization ? adminApi : api;
  const [messages, setMessages] = useState<RequestMessage[] | null>(null);
  const [text, setText] = useState("");
  const [type, setType] = useState(organization ? "ORGANIZATION_PUBLIC" : "RESIDENT_PUBLIC");
  const [files, setFiles] = useState<File[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [retry, setRetry] = useState(0);
  useEffect(() => { const controller = new AbortController(); setError(""); client.messages(requestId, controller.signal).then(setMessages).catch((e) => { if (!controller.signal.aborted) setError(e.message); }); return () => controller.abort(); }, [requestId, organization, retry]);
  async function submit(event: FormEvent) {
    event.preventDefault(); if (busy) return; setBusy(true); setError("");
    const body = new FormData(); body.set("type", type); body.set("text", text.trim()); files.forEach((file) => body.append("attachments", file));
    try { await client.sendMessage(requestId, body); setText(""); setFiles([]); setMessages(await client.messages(requestId)); }
    catch (e) { setError((e as Error).message); }
    finally { setBusy(false); }
  }
  return <section className="panel conversation-panel">
    <div className="row"><h2>Переписка</h2>{messages === null && !error && <Loading />}</div>
    {error && <><ErrorMessage message={error}/><button className="text-button" type="button" onClick={() => setRetry((value) => value + 1)}>Повторить загрузку</button></>}
    {messages?.length === 0 && <p className="muted">Сообщений пока нет.</p>}
    <div className="message-list">{messages?.map((message) => <article key={message.id} className={`request-message ${message.type.toLowerCase()}`}>
      <div className="row"><strong>{message.type === "INFO_REQUEST" ? "Запрос информации" : message.authorName}</strong><small>{new Date(message.createdAt).toLocaleString("ru-RU")}</small></div>
      {message.text && <p>{message.text}</p>}
      {message.attachments?.map((attachment) => <a key={attachment.id} href={apiPath(`/request-attachments/${attachment.id}`)} target="_blank" rel="noreferrer">{attachment.name}</a>)}
    </article>)}</div>
    <form className="form message-form" onSubmit={submit}><fieldset disabled={busy}>
      {organization && <label>Тип сообщения<select value={type} onChange={(event) => setType(event.target.value)}><option value="ORGANIZATION_PUBLIC">Ответ жителю</option><option value="INFO_REQUEST">Запросить информацию</option><option value="INTERNAL_NOTE">Внутренняя заметка</option></select></label>}
      <label htmlFor={`message-${requestId}`}>Новое сообщение</label><textarea id={`message-${requestId}`} value={text} maxLength={5000} rows={3} onChange={(event) => setText(event.target.value)} />
      <label>Фотографии<input type="file" multiple accept="image/jpeg,image/png,image/webp" onChange={(event) => setFiles(Array.from(event.target.files ?? []).slice(0, 5))}/></label><small>До 5 файлов, каждый до 5 МБ</small>
      <button className="button primary" type="submit" disabled={!text.trim() && files.length === 0}>{busy ? "Отправляем…" : "Отправить"}</button>
    </fieldset></form>
  </section>;
}
