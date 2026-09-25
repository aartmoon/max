import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apartmentApi, houseApi } from "../api";
import type { AddressSuggestion, House, UserApartment } from "../types";
import { AddressAutocomplete } from "../components/AddressAutocomplete";
import { Back, ErrorMessage, Loading } from "../components/UI";

const missing = "Не опубликовано в ГИС ЖКХ";

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

  const text = (value: string | number | null | undefined) => value ?? missing;
  const area = (value: number | null | undefined) => value == null ? missing : `${new Intl.NumberFormat("ru-RU").format(value)} м²`;
  const percent = (value: number | null | undefined) => value == null ? missing : `${new Intl.NumberFormat("ru-RU").format(value)} %`;
  const factDate = (value: string | null | undefined) => value == null ? missing : new Intl.DateTimeFormat("ru-RU").format(new Date(value));
  const characteristics = house?.characteristics;
  const management = house?.management;
  const sources = house?.dataSources ?? (house?.dataSource ? [{ name: house.dataSource, available: true, updatedAt: house.dataUpdatedAt ?? "", stale: house.stale }] : []);
  const website = management?.website;
  const safeWebsite = website && /^https?:\/\//i.test(website) ? website : null;
  const websiteLabel = safeWebsite ? safeWebsite.replace(/^https?:\/\//i, "").replace(/\/$/, "") : website;
  const phone = management?.phone ?? house?.contact;
  const phoneHref = phone ? `tel:${phone.replace(/[^+\d]/g, "")}` : null;

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
      <section className="house-banner"><span className="house-symbol" aria-hidden="true">⌂</span><div><small>{current?.label || current?.address}</small><h2>{house.address}</h2><p>{text(characteristics ? characteristics.yearBuilt : house.yearBuilt)} год постройки · {text(characteristics?.floors ?? house.floors)} этажей</p></div></section>
      <section className="panel"><h2>Паспорт дома</h2><dl className="house-facts">
        <div><dt>Кадастровый номер</dt><dd>{text(house.cadastralNumber)}</dd></div>
        <div><dt>Тип дома</dt><dd>{text(characteristics?.houseType)}</dd></div>
        <div><dt>Статус</dt><dd>{text(characteristics?.status)}</dd></div>
        <div><dt>Состояние</dt><dd>{text(characteristics?.condition)}</dd></div>
        <div><dt>Стадия жизненного цикла</dt><dd>{text(characteristics?.lifecycleStage)}</dd></div>
        <div><dt>Серия / проект</dt><dd>{text(characteristics?.projectSeries)}</dd></div>
        <div><dt>Год постройки</dt><dd>{text(characteristics ? characteristics.yearBuilt : house.yearBuilt)}</dd></div>
        <div><dt>Год ввода в эксплуатацию</dt><dd>{text(characteristics?.operationYear)}</dd></div>
        <div><dt>Год реконструкции</dt><dd>{text(characteristics?.reconstructionYear)}</dd></div>
        <div><dt>Общая площадь</dt><dd>{area(characteristics?.totalArea ?? house.totalArea)}</dd></div>
        <div><dt>Жилая площадь</dt><dd>{area(characteristics?.livingArea ?? house.livingArea)}</dd></div>
        <div><dt>Нежилая площадь</dt><dd>{area(characteristics?.nonResidentialArea)}</dd></div>
        <div><dt>Жилых помещений</dt><dd>{text(characteristics?.residentialPremises ?? house.apartments)}</dd></div>
        <div><dt>Площадь жилых помещений</dt><dd>{area(characteristics?.residentialPremisesArea)}</dd></div>
        <div><dt>Жилых помещений, связанных с ЕГРН</dt><dd>{text(characteristics?.residentialPremisesWithRealty)}</dd></div>
        <div><dt>Нежилых помещений</dt><dd>{text(characteristics?.nonResidentialPremises)}</dd></div>
        <div><dt>Площадь нежилых помещений</dt><dd>{area(characteristics?.nonResidentialPremisesArea)}</dd></div>
        <div><dt>Нежилых помещений без общего имущества</dt><dd>{text(characteristics?.nonResidentialPremisesNotCommon)}</dd></div>
        <div><dt>Их площадь</dt><dd>{area(characteristics?.nonResidentialPremisesNotCommonArea)}</dd></div>
        <div><dt>Этажей</dt><dd>{text(characteristics?.floors ?? house.floors)}</dd></div>
        <div><dt>Подъездов</dt><dd>{text(characteristics?.entrances ?? house.entrances)}</dd></div>
        <div><dt>Собственников / долей</dt><dd>{text(characteristics?.ownersOrShares)}</dd></div>
        <div><dt>Физический износ</dt><dd>{percent(characteristics?.deteriorationPercent)}</dd></div>
        <div><dt>Дата оценки износа</dt><dd>{factDate(characteristics?.deteriorationDate)}</dd></div>
        <div><dt>Материал стен</dt><dd>{text(characteristics?.wallMaterial)}</dd></div>
        <div><dt>Класс энергоэффективности</dt><dd>{text(characteristics?.energyEfficiency)}</dd></div>
      </dl></section>
      <section className="panel"><div className="eyebrow">КТО УПРАВЛЯЕТ ДОМОМ</div><h2>{text(management?.shortName ?? management?.fullName ?? house.organization)}</h2><dl className="house-facts">
        <div><dt>Способ управления</dt><dd>{text(management?.method)}</dd></div>
        <div><dt>Полное название</dt><dd>{text(management?.fullName)}</dd></div>
        <div><dt>Адрес организации</dt><dd>{text(management?.address)}</dd></div>
        <div><dt>Руководитель</dt><dd>{text(management?.chief ?? house.manager)}</dd></div>
        <div><dt>Телефон</dt><dd>{phoneHref ? <a href={phoneHref}>{phone}</a> : missing}</dd></div>
        <div><dt>Сайт</dt><dd>{safeWebsite ? <a href={safeWebsite} target="_blank" rel="noreferrer">{websiteLabel}</a> : text(websiteLabel)}</dd></div>
        <div><dt>ИНН</dt><dd>{text(management?.inn)}</dd></div>
        <div><dt>ОГРН</dt><dd>{text(management?.ogrn)}</dd></div>
        <div><dt>Тип организации</dt><dd>{text(management?.organizationType)}</dd></div>
        <div><dt>Начало управления</dt><dd>{factDate(management?.contractStart)}</dd></div>
        <div><dt>Окончание управления</dt><dd>{factDate(management?.contractEnd)}</dd></div>
        <div><dt>GUID организации</dt><dd>{text(management?.organizationGuid)}</dd></div>
        <div><dt>GUID в реестре</dt><dd>{text(management?.registryOrganizationGuid)}</dd></div>
      </dl><Link className="button primary full" to="/requests/new">Создать заявку <span>→</span></Link></section>
      <section className="panel"><h2>Источники данных</h2><dl className="house-facts">
        {sources.map((source) => <div key={source.name}><dt>{source.name}</dt><dd>{source.available ? "данные получены" : "недоступен при обновлении"}</dd></div>)}
      </dl></section>
      <p className="demo-note">Обновлено: {house.dataUpdatedAt ? new Date(house.dataUpdatedAt).toLocaleString("ru-RU") : missing}.</p>
    </> : null}
  </>;
}
