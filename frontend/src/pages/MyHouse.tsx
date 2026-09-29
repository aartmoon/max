import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apartmentApi, houseApi } from "../api";
import type { AddressSuggestion, House, UserApartment } from "../types";
import { AddressAutocomplete } from "../components/AddressAutocomplete";
import { Back, ErrorMessage, Loading } from "../components/UI";

function formatValue(value: string | number | boolean | null | undefined): string {
  if (value == null) return "-";
  if (typeof value === "string") {
    const trimmed = value.trim();
    const normalized = trimmed.replace(/\s+/g, " ").toLocaleLowerCase("ru-RU");
    if (!trimmed || normalized.includes("не опубликовано в гис жкх")) return "-";
    return trimmed;
  }
  return String(value);
}

function formatUnitValue(value: string | number | null | undefined, unit: string): string {
  const formatted = formatValue(value);
  if (formatted === "-") return formatted;
  const number = typeof value === "number" ? new Intl.NumberFormat("ru-RU").format(value) : formatted;
  return number.endsWith(unit) ? number : `${number} ${unit}`;
}

function formatDate(value: string | null | undefined): string {
  const formatted = formatValue(value);
  if (formatted === "-") return formatted;
  const date = new Date(formatted);
  return Number.isNaN(date.getTime()) ? formatted : new Intl.DateTimeFormat("ru-RU").format(date);
}

function formatDateTime(value: string | null | undefined): string {
  const formatted = formatValue(value);
  if (formatted === "-") return formatted;
  const date = new Date(formatted);
  return Number.isNaN(date.getTime()) ? formatted : date.toLocaleString("ru-RU");
}
export default function MyHouse() {
  const [apartments, setApartments] = useState<UserApartment[]>([]);
  const [current, setCurrent] = useState<UserApartment | null>(null);
  const [houseSelection, setHouseSelection] = useState<AddressSuggestion | null>(null);
  const [apartmentSelection, setApartmentSelection] = useState<AddressSuggestion | null>(null);
  const [house, setHouse] = useState<House | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [retry, setRetry] = useState(0);

  async function loadApartments(signal?: AbortSignal) {
    const items = await apartmentApi.list(signal);
    setApartments(items);
    setCurrent((previous) => {
      if (previous) {
        const retained = items.find((item) => item.id === previous.id);
        if (retained) return retained;
      }
      return items.find((item) => item.isDefault) ?? items[0] ?? null;
    });
  }

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    loadApartments(controller.signal)
      .catch((reason: Error) => {
        if (!controller.signal.aborted) setError(reason.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, []);

  useEffect(() => {
    if (!current) {
      setHouse(null);
      return;
    }
    const controller = new AbortController();
    setLoading(true);
    setError("");
    houseApi
      .resolve(current.houseObjectId, controller.signal)
      .then(setHouse)
      .catch((reason: Error) => {
        if (!controller.signal.aborted) setError(reason.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [current?.id, retry]);

  async function addApartment() {
    if (!houseSelection || saving) return;
    setSaving(true);
    setError("");
    try {
      const saved = await apartmentApi.create({
        houseObjectId: houseSelection.objectId,
        apartmentObjectId: apartmentSelection?.objectId,
        label: apartmentSelection?.displayName || houseSelection.displayName,
        isDefault: apartments.length === 0,
      });
      await loadApartments();
      setCurrent(saved);
      setHouseSelection(null);
      setApartmentSelection(null);
    } catch (reason) {
      setError((reason as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function makeDefault(item: UserApartment) {
    setSaving(true);
    setError("");
    try {
      const updated = await apartmentApi.setDefault(item.id);
      await loadApartments();
      setCurrent(updated);
    } catch (reason) {
      setError((reason as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function removeApartment(item: UserApartment) {
    setSaving(true);
    setError("");
    try {
      await apartmentApi.remove(item.id);
      await loadApartments();
    } catch (reason) {
      setError((reason as Error).message);
    } finally {
      setSaving(false);
    }
  }

  const area = (value: number | null | undefined) => formatUnitValue(value, "м²");
  const percent = (value: number | null | undefined) => formatUnitValue(value, "%");
  const characteristics = house?.characteristics;
  const management = house?.management;
  const sources = house?.dataSources ?? (house?.dataSource ? [{ name: house.dataSource, available: true, updatedAt: house.dataUpdatedAt ?? "", stale: house.stale }] : []);
  const website = formatValue(management?.website);
  const safeWebsite = website !== "-" && /^https?:\/\//i.test(website) ? website : null;
  const websiteLabel = safeWebsite ? safeWebsite.replace(/^https?:\/\//i, "").replace(/\/$/, "") : website;
  const phone = management?.phone ?? house?.contact;
  const phoneLabel = formatValue(phone);
  const phoneDigits = phoneLabel.replace(/[^+\d]/g, "");
  const phoneHref = phoneLabel !== "-" && phoneDigits ? `tel:${phoneDigits}` : null;

  return <>
    <Back />
    <div className="eyebrow">ПАСПОРТ ДОМА</div>
    <h1>Мой дом</h1>
    <p className="intro">Добавьте одну или несколько квартир. Основная квартира будет автоматически подставляться в новую заявку.</p>

    <section className="panel form house-picker">
      <h2>Мои квартиры</h2>
      {apartments.length > 0 ? (
        <div className="apartment-list">
          {apartments.map((item) => (
            <div className={`apartment-item ${current?.id === item.id ? "selected" : ""}`} key={item.id}>
              <button className="text-button apartment-title" type="button" onClick={() => setCurrent(item)}>
                {item.label || item.address}
              </button>
              {item.isDefault && <span className="badge">Основная</span>}
              <div className="apartment-actions">
                {!item.isDefault && <button className="text-button" disabled={saving} type="button" onClick={() => makeDefault(item)}>Сделать основной</button>}
                <button className="text-button" disabled={saving} type="button" onClick={() => removeApartment(item)}>Удалить</button>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <p className="muted">Сохранённых квартир пока нет.</p>
      )}
      <AddressAutocomplete label="Адрес дома" kind="house" value={houseSelection} onChange={(value) => {
        setHouseSelection(value);
        setApartmentSelection(null);
      }} required placeholder="Начните вводить адрес" />
      {houseSelection && <AddressAutocomplete label="Квартира" kind="apartment" value={apartmentSelection} onChange={setApartmentSelection} parentObjectId={houseSelection.objectId} placeholder="Выберите квартиру" />}
      {error && <ErrorMessage message={error} />}
      <button className="button primary full" type="button" disabled={!houseSelection || saving} onClick={addApartment}>
        {saving ? "Сохраняем…" : "Сохранить квартиру"}
      </button>
    </section>

    {loading ? <Loading /> : error && current ? <>
      <ErrorMessage message={error} />
      <div className="house-actions">
        <button className="button primary" onClick={() => setRetry((value) => value + 1)}>Повторить</button>
      </div>
    </> : house ? <>
      {house.stale && <p className="house-warning">ГИС ЖКХ сейчас недоступна — данные могут быть устаревшими.</p>}
      <section className="house-banner"><span className="house-symbol" aria-hidden="true">⌂</span><div><small>{formatValue(current?.label || current?.address)}</small><h2>{formatValue(house.address)}</h2></div></section>
      <section className="panel"><h2>Паспорт дома</h2><dl className="house-facts house-main-facts">
        <div><dt>Кадастровый номер</dt><dd>{formatValue(house.cadastralNumber)}</dd></div>
        <div><dt>Тип дома</dt><dd>{formatValue(characteristics?.houseType)}</dd></div>
        <div><dt>Состояние</dt><dd>{formatValue(characteristics?.condition)}</dd></div>
        <div><dt>Год постройки</dt><dd>{formatValue(characteristics ? characteristics.yearBuilt : house.yearBuilt)}</dd></div>
        <div><dt>Общая площадь</dt><dd>{area(characteristics?.totalArea ?? house.totalArea)}</dd></div>
        <div><dt>Жилая площадь</dt><dd>{area(characteristics?.livingArea ?? house.livingArea)}</dd></div>
        <div><dt>Жилых помещений</dt><dd>{formatValue(characteristics?.residentialPremises ?? house.apartments)}</dd></div>
        <div><dt>Этажей</dt><dd>{formatValue(characteristics?.floors ?? house.floors)}</dd></div>
        <div><dt>Подъездов</dt><dd>{formatValue(characteristics?.entrances ?? house.entrances)}</dd></div>
        <div><dt>Материал стен</dt><dd>{formatValue(characteristics?.wallMaterial)}</dd></div>
        <div><dt>Физический износ</dt><dd>{percent(characteristics?.deteriorationPercent)}</dd></div>
      </dl></section>
      <details className="panel house-more"><summary>Все характеристики</summary><dl className="house-facts house-more-facts">
        <div><dt>Статус</dt><dd>{formatValue(characteristics?.status)}</dd></div>
        <div><dt>Стадия жизненного цикла</dt><dd>{formatValue(characteristics?.lifecycleStage)}</dd></div>
        <div><dt>Серия / проект</dt><dd>{formatValue(characteristics?.projectSeries)}</dd></div>
        <div><dt>Год ввода в эксплуатацию</dt><dd>{formatValue(characteristics?.operationYear)}</dd></div>
        <div><dt>Год реконструкции</dt><dd>{formatValue(characteristics?.reconstructionYear)}</dd></div>
        <div><dt>Площадь жилых помещений</dt><dd>{area(characteristics?.residentialPremisesArea)}</dd></div>
        <div><dt>Жилых помещений, связанных с ЕГРН</dt><dd>{formatValue(characteristics?.residentialPremisesWithRealty)}</dd></div>
        <div><dt>Площадь жилых помещений, связанных с ЕГРН</dt><dd>{area(characteristics?.residentialPremisesWithRealtyArea)}</dd></div>
        <div><dt>Нежилая площадь</dt><dd>{area(characteristics?.nonResidentialArea)}</dd></div>
        <div><dt>Нежилых помещений</dt><dd>{formatValue(characteristics?.nonResidentialPremises)}</dd></div>
        <div><dt>Площадь нежилых помещений</dt><dd>{area(characteristics?.nonResidentialPremisesArea)}</dd></div>
        <div><dt>Нежилых помещений без общего имущества</dt><dd>{formatValue(characteristics?.nonResidentialPremisesNotCommon)}</dd></div>
        <div><dt>Их площадь</dt><dd>{area(characteristics?.nonResidentialPremisesNotCommonArea)}</dd></div>
        <div><dt>Собственников / долей</dt><dd>{formatValue(characteristics?.ownersOrShares)}</dd></div>
        <div><dt>Дата оценки износа</dt><dd>{formatDate(characteristics?.deteriorationDate)}</dd></div>
        <div><dt>Класс энергоэффективности</dt><dd>{formatValue(characteristics?.energyEfficiency)}</dd></div>
      </dl></details>
      <section className="panel"><div className="eyebrow">КТО УПРАВЛЯЕТ ДОМОМ</div><h2>{formatValue(management?.shortName ?? management?.fullName ?? house.organization)}</h2><dl className="house-facts">
        <div><dt>Способ управления</dt><dd>{formatValue(management?.method)}</dd></div>
        <div><dt>Полное название</dt><dd>{formatValue(management?.fullName)}</dd></div>
        <div><dt>Адрес организации</dt><dd>{formatValue(management?.address)}</dd></div>
        <div><dt>Руководитель</dt><dd>{formatValue(management?.chief ?? house.manager)}</dd></div>
        <div><dt>Телефон</dt><dd>{phoneHref ? <a href={phoneHref}>{phoneLabel}</a> : phoneLabel}</dd></div>
        <div><dt>Сайт</dt><dd>{safeWebsite ? <a href={safeWebsite} target="_blank" rel="noreferrer">{websiteLabel}</a> : formatValue(websiteLabel)}</dd></div>
        <div><dt>ИНН</dt><dd>{formatValue(management?.inn)}</dd></div>
        <div><dt>ОГРН</dt><dd>{formatValue(management?.ogrn)}</dd></div>
        <div><dt>Тип организации</dt><dd>{formatValue(management?.organizationType)}</dd></div>
        <div><dt>Начало управления</dt><dd>{formatDate(management?.contractStart)}</dd></div>
        <div><dt>Окончание управления</dt><dd>{formatDate(management?.contractEnd)}</dd></div>
        <div><dt>GUID организации</dt><dd>{formatValue(management?.organizationGuid)}</dd></div>
        <div><dt>GUID в реестре</dt><dd>{formatValue(management?.registryOrganizationGuid)}</dd></div>
      </dl><Link className="button primary full" to="/requests/new">Создать заявку <span>→</span></Link></section>
      <section className="panel"><h2>Источники данных</h2><dl className="house-facts">
        {sources.map((source) => <div key={source.name}><dt>{formatValue(source.name)}</dt><dd>{source.available ? "данные получены" : "недоступен при обновлении"}</dd></div>)}
      </dl></section>
      <p className="demo-note">Обновлено: {formatDateTime(house.dataUpdatedAt)}.</p>
    </> : null}
  </>;
}
