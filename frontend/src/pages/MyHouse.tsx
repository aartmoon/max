import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { houseApi } from "../api";
import type { House } from "../types";
import { Back, ErrorMessage, Loading } from "../components/UI";
export default function MyHouse() {
  const [house, setHouse] = useState<House | null>(null),
    [error, setError] = useState(""),
    [retry, setRetry] = useState(0);
  useEffect(() => {
    const c = new AbortController();
    setError("");
    houseApi
      .get(c.signal)
      .then(setHouse)
      .catch((e) => {
        if (!c.signal.aborted) setError(e.message);
      });
    return () => c.abort();
  }, [retry]);
  const area = (n: number) => new Intl.NumberFormat("ru-RU").format(n) + " м²";
  return (
    <>
      <Back />
      <div className="eyebrow">ПАСПОРТ ДОМА</div>
      <h1>Мой дом</h1>
      <p className="intro">Всё важное о доме и тех, кто о нём заботится.</p>
      {error ? (
        <>
          <ErrorMessage message={error} />
          <button className="button" onClick={() => setRetry(retry + 1)}>
            Повторить
          </button>
        </>
      ) : !house ? (
        <Loading />
      ) : (
        <>
          <section className="house-banner">
            <span className="house-symbol" aria-hidden="true">
              ⌂
            </span>
            <div>
              <small>ВАШ ТЕСТОВЫЙ ДОМ</small>
              <h2>{house.address}</h2>
              <p>
                {house.yearBuilt} год постройки · {house.floors} этажей
              </p>
            </div>
          </section>
          <section className="panel">
            <h2>Характеристики дома</h2>
            <dl className="house-facts">
              <div>
                <dt>Общая площадь</dt>
                <dd>{area(house.totalArea)}</dd>
              </div>
              <div>
                <dt>Жилая площадь</dt>
                <dd>{area(house.livingArea)}</dd>
              </div>
              <div>
                <dt>Этажей</dt>
                <dd>{house.floors}</dd>
              </div>
              <div>
                <dt>Подъездов</dt>
                <dd>{house.entrances}</dd>
              </div>
              <div>
                <dt>Квартир</dt>
                <dd>{house.apartments}</dd>
              </div>
              <div>
                <dt>Год постройки</dt>
                <dd>{house.yearBuilt}</dd>
              </div>
            </dl>
          </section>
          <section className="panel">
            <div className="eyebrow">ОБСЛУЖИВАНИЕ ДОМА</div>
            <h2>{house.organization}</h2>
            <dl>
              <dt>Ответственный за дом</dt>
              <dd>{house.manager}</dd>
              <dt>Диспетчерская</dt>
              <dd>{house.contact}</dd>
            </dl>
            <Link className="button primary full" to="/requests/new">
              Создать заявку <span>→</span>
            </Link>
          </section>
          <p className="demo-note">
            Демонстрационный паспорт дома. Площади, характеристики и
            ответственный — тестовые данные, не сведения из реестра. Дом пока
            закреплён за общим демо-пользователем.
          </p>
        </>
      )}
    </>
  );
}
