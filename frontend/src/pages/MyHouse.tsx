import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { houseApi } from "../api";
import type { AddressSuggestion, House } from "../types";
import { AddressAutocomplete } from "../components/AddressAutocomplete";
import { Back, ErrorMessage, Loading } from "../components/UI";

const storageKey = "tvoy-dom:selected-house-object-id";

export default function MyHouse() {
  const [objectId, setObjectId] = useState(() => localStorage.getItem(storageKey) ?? "");
  const [selection, setSelection] = useState<AddressSuggestion | null>(null);
  const [house, setHouse] = useState<House | null>(null);
  const [loading, setLoading] = useState(Boolean(objectId));
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    if (!objectId) return;
    const controller = new AbortController();
    setLoading(true);
    setError("");
    houseApi.resolve(objectId, controller.signal).then((value) => {
      setHouse(value);
      setSelection({ objectId: value.garObjectId, objectGuid: value.objectGuid, objectKind: "house", displayName: value.address, fullAddress: value.address });
    }).catch((reason: Error) => {
      if (!controller.signal.aborted) setError(reason.message);
    }).finally(() => {
      if (!controller.signal.aborted) setLoading(false);
    });
    return () => controller.abort();
  }, [objectId, retry]);

  const choose = (value: AddressSuggestion | null) => {
    setSelection(value);
    setHouse(null);
    setError("");
    if (value) {
      localStorage.setItem(storageKey, value.objectId);
      setObjectId(value.objectId);
    } else {
      localStorage.removeItem(storageKey);
      setObjectId("");
    }
  };
  const text = (value: string | number | null) => value ?? "Нет данных";
  const area = (value: number | null) => value == null ? "Нет данных" : `${new Intl.NumberFormat("ru-RU").format(value)} м²`;

  return <>
    <Back />
    <div className="eyebrow">ПАСПОРТ ДОМА</div>
    <h1>Мой дом</h1>
    <p className="intro">Выберите свой адрес — характеристики загрузятся из открытого реестра ГИС ЖКХ.</p>
    {!objectId && <section className="panel form house-picker">
      <AddressAutocomplete label="Адрес дома" kind="house" value={selection} onChange={choose} required placeholder="Начните вводить адрес" />
    </section>}
    {loading ? <Loading /> : error ? <>
      <ErrorMessage message={error} />
      <div className="house-actions">
        <button className="button primary" onClick={() => setRetry((value) => value + 1)}>Повторить</button>
        <button className="button" onClick={() => choose(null)}>Выбрать другой дом</button>
      </div>
    </> : house ? <>
      {house.stale && <p className="house-warning">ГИС ЖКХ сейчас недоступна — данные могут быть устаревшими.</p>}
      <section className="house-banner"><span className="house-symbol" aria-hidden="true">⌂</span><div><small>{house.dataSource}</small><h2>{house.address}</h2><p>{text(house.yearBuilt)} год постройки · {text(house.floors)} этажей</p></div></section>
      <button className="text-button" onClick={() => choose(null)}>Выбрать другой дом</button>
      <section className="panel"><h2>Характеристики дома</h2><dl className="house-facts">
        <div><dt>Кадастровый номер</dt><dd>{text(house.cadastralNumber)}</dd></div>
        <div><dt>Общая площадь</dt><dd>{area(house.totalArea)}</dd></div>
        <div><dt>Жилая площадь</dt><dd>{area(house.livingArea)}</dd></div>
        <div><dt>Этажей</dt><dd>{text(house.floors)}</dd></div>
        <div><dt>Подъездов</dt><dd>{text(house.entrances)}</dd></div>
        <div><dt>Квартир</dt><dd>{text(house.apartments)}</dd></div>
        <div><dt>Год постройки</dt><dd>{text(house.yearBuilt)}</dd></div>
      </dl></section>
      <section className="panel"><div className="eyebrow">ОБСЛУЖИВАНИЕ ДОМА</div><h2>{text(house.organization)}</h2><dl><dt>Ответственный за дом</dt><dd>{text(house.manager)}</dd><dt>Контакт</dt><dd>{text(house.contact)}</dd></dl><Link className="button primary full" to="/requests/new">Создать заявку <span>→</span></Link></section>
      <p className="demo-note">Источник: {house.dataSource}. Обновлено: {new Date(house.dataUpdatedAt).toLocaleString("ru-RU")}.</p>
    </> : null}
  </>;
}
