import { useState, useEffect, useRef, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useMaxBridge } from "../integration/MaxIntegration";
import { api } from "../api";
import { apartmentApi } from "../api";
import { kinds, type Kind } from "../types";
import { Back, ErrorMessage } from "../components/UI";
import { AddressAutocomplete } from "../components/AddressAutocomplete";
import type { AddressSuggestion, UserApartment } from "../types";
export default function NewRequest() {
  const [query] = useSearchParams();
  const requested = query.get("kind");
  const [kind, setKind] = useState<Kind>(
    requested && requested in kinds ? (requested as Kind) : "APPLICATION",
  );
  const [description, setDescription] = useState(""),
    [house, setHouse] = useState<AddressSuggestion | null>(null),
    [apartment, setApartment] = useState<AddressSuggestion | null>(null),
    [apartments, setApartments] = useState<UserApartment[]>([]),
    [selectedApartmentId, setSelectedApartmentId] = useState(""),
    [photo, setPhoto] = useState<File | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const navigate = useNavigate();
  const bridge = useMaxBridge();
  const initialKind = useRef(kind);
  const dirty = Boolean(description.trim() || photo || house || apartment || kind !== initialKind.current);
  useEffect(() => {
    bridge.closingConfirmation(dirty);
    return () => bridge.closingConfirmation(false);
  }, [bridge, dirty]);
  useEffect(() => {
    const controller = new AbortController();
    apartmentApi
      .list(controller.signal)
      .then((items) => {
        setApartments(items);
        const current = items.find((item) => item.isDefault) ?? items[0];
        if (!current || house) return;
        applySavedApartment(current);
      })
      .catch(() => {});
    return () => controller.abort();
  }, []);
  function applySavedApartment(current: UserApartment) {
    setSelectedApartmentId(current.id);
    setHouse({
      objectId: current.houseObjectId,
      objectGuid: current.houseObjectGuid,
      objectKind: "house",
      displayName: current.address,
      fullAddress: current.address,
    });
    if (current.apartmentObjectId) {
      setApartment({
        objectId: current.apartmentObjectId,
        objectGuid: current.apartmentObjectGuid,
        parentObjectId: current.houseObjectId,
        objectKind: "apartment",
        displayName: current.label || current.address,
        fullAddress: current.address,
      });
    } else {
      setApartment(null);
    }
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    setError("");
    if (description.trim().length < 5) {
      setError("Описание должно содержать не менее 5 символов.");
      return;
    }
    if (!house) {
      setError("Выберите дом из списка ГАР.");
      return;
    }
    setBusy(true);
    try {
      const body = new FormData();
      body.set("description", description.trim());
      body.set("houseObjectId", house.objectId);
      if (apartment) body.set("apartmentObjectId", apartment.objectId);
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
      <div className="eyebrow">НОВАЯ ЗАЯВКА</div>
      <h1>Новая заявка</h1>
      <p className="intro">
        Опишите ситуацию. Мы сохраним заявку и определим, кому его направить.
      </p>
      <form className="panel form" onSubmit={submit}>
        <fieldset disabled={busy}>
          <label htmlFor="kind">
            Тип заявки <span>*</span>
          </label>
          <select
            id="kind"
            value={kind}
            onChange={(e) => setKind(e.target.value as Kind)}
          >
            {(
              [
                "APPLICATION",
                "QUESTION",
                "EMERGENCY",
                "PROBLEM",
                "COMPLAINT",
              ] as Kind[]
            ).map((value) => (
              <option key={value} value={value}>
                {kinds[value]}
              </option>
            ))}
          </select>
          <p className={kind === "EMERGENCY" ? "urgent-note" : "type-hint"}>
            {kind === "EMERGENCY"
              ? "Экстренная заявка будет выделена в кабинете УК. В демо она не вызывает аварийную службу."
              : kind === "QUESTION"
                ? "Уточните вопрос по обслуживанию вашего дома."
                : kind === "COMPLAINT"
                  ? "Расскажите, что вас не устраивает и какого решения вы ждёте."
                  : kind === "PROBLEM"
                    ? "Опишите неисправность или проблему в доме."
                    : "Оставьте запрос на обслуживание или помощь управляющей компании."}
          </p>
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
          {apartments.length > 0 && (
            <>
              <label htmlFor="saved-apartment">Сохранённая квартира</label>
              <select
                id="saved-apartment"
                value={selectedApartmentId}
                onChange={(event) => {
                  const current = apartments.find(
                    (item) => item.id === event.target.value,
                  );
                  if (current) applySavedApartment(current);
                }}
              >
                {apartments.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.label || item.address}
                  </option>
                ))}
              </select>
            </>
          )}
          <AddressAutocomplete
            label="Адрес дома"
            kind="house"
            required
            value={house}
            onChange={(next) => {
              setSelectedApartmentId("");
              setHouse(next);
              setApartment(null);
            }}
            placeholder="Начните вводить улицу и дом"
          />
          {house && (
            <AddressAutocomplete
              label="Квартира"
              kind="apartment"
              value={apartment}
              onChange={setApartment}
              parentObjectId={house.objectId}
              placeholder="Выберите или начните вводить номер"
            />
          )}
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
            {busy ? "Создаём заявку…" : "Продолжить"}
            <span>→</span>
          </button>
        </fieldset>
        <p className="form-note">
          Тестовый режим: заявка не отправляется в реальные организации.
        </p>
      </form>
    </>
  );
}
