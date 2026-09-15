import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../api";
import { kinds, type Kind } from "../types";
import { Back, ErrorMessage } from "../components/UI";
const testAddress = "г. Москва, ул. Тестовая, д. 1";
export default function NewRequest() {
  const [query] = useSearchParams();
  const requested = query.get("kind");
  const kind: Kind =
    requested === "APPLICATION" || requested === "QUESTION"
      ? requested
      : "PROBLEM";
  const [description, setDescription] = useState(""),
    [address, setAddress] = useState(testAddress),
    [photo, setPhoto] = useState<File | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const navigate = useNavigate();
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    setError("");
    if (description.trim().length < 5 || address.trim().length < 5) {
      setError(
        "Заполните описание и адрес: не менее 5 символов в каждом поле.",
      );
      return;
    }
    setBusy(true);
    try {
      const body = new FormData();
      body.set("description", description.trim());
      body.set("address", address.trim());
      body.set("kind", kind);
      if (photo) body.set("photo", photo);
      const item = await api.create(body);
      navigate(`/requests/${item.id}`, { replace: true });
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <Back />
      <div className="eyebrow">НОВОЕ ОБРАЩЕНИЕ</div>
      <h1>{kind === "PROBLEM" ? "Что случилось?" : kinds[kind]}</h1>
      <p className="intro">
        Опишите ситуацию. Мы сохраним обращение и определим, кому его направить.
      </p>
      <form className="panel form" onSubmit={submit}>
        <fieldset disabled={busy}>
          <label htmlFor="description">
            {kind === "QUESTION" ? "Ваш вопрос" : "Описание"} <span>*</span>
          </label>
          <textarea
            id="description"
            required
            minLength={5}
            maxLength={5000}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Например: в подъезде течёт труба на первом этаже"
            rows={5}
          />
          <small>Укажите, где и когда вы заметили проблему.</small>
          <label htmlFor="address">
            Адрес дома <span>*</span>
          </label>
          <input
            id="address"
            required
            minLength={5}
            maxLength={300}
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            autoComplete="street-address"
          />
          <button
            className="text-button"
            type="button"
            onClick={() => setAddress(testAddress)}
          >
            Использовать тестовый адрес
          </button>
          <label htmlFor="photo">
            Фотография <small>необязательно</small>
          </label>
          <div className="upload">
            <input
              id="photo"
              type="file"
              accept="image/jpeg,image/png,image/webp"
              onChange={(e) => {
                const file = e.target.files?.[0];
                setError("");
                if (
                  file &&
                  (file.size > 5 * 1024 * 1024 ||
                    !["image/jpeg", "image/png", "image/webp"].includes(
                      file.type,
                    ))
                ) {
                  setError("Выберите JPEG, PNG или WebP размером до 5 МБ.");
                  e.target.value = "";
                  setPhoto(null);
                  return;
                }
                setPhoto(file ?? null);
              }}
            />
            <small>JPEG, PNG или WebP · до 5 МБ</small>
            {photo && (
              <button
                type="button"
                className="text-button"
                onClick={() => {
                  setPhoto(null);
                  const input = document.getElementById(
                    "photo",
                  ) as HTMLInputElement;
                  input.value = "";
                }}
              >
                Удалить фото
              </button>
            )}
          </div>
          {error && <ErrorMessage message={error} />}
          <button className="button primary full" type="submit">
            {busy ? "Создаём обращение…" : "Продолжить"}
            <span>→</span>
          </button>
        </fieldset>
        <p className="form-note">
          Тестовый режим: обращение не отправляется в реальные организации.
        </p>
      </form>
    </>
  );
}
