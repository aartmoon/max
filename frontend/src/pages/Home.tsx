import { Link } from "react-router-dom";
export default function Home() {
  return (
    <>
      <div className="eyebrow">ДОМА ВСЁ ПОД КОНТРОЛЕМ</div>
      <section className="hero">
        <div className="house-art" aria-hidden="true">
          <span>⌂</span>
          <b>✓</b>
        </div>
        <h1>
          Хороший дом
          <br />
          начинается с вас.
        </h1>
        <p>
          Вопрос, запрос или проблема — всё начинается с заявки. Следите за
          решением и будьте на связи с управляющей компанией.
        </p>
        <Link className="button primary" to="/requests/new">
          Создать заявку <span>↗</span>
        </Link>
      </section>
      <div className="action-grid">
        <Link className="action-card" to="/requests">
          <span className="icon">▤</span>
          <h2>Мои заявки</h2>
          <p>Статусы, ответы УК и история решения</p>
          <span className="arrow">↗</span>
        </Link>
        <Link className="action-card" to="/house">
          <span className="icon">⌂</span>
          <h2>Мой дом</h2>
          <p>Управляющая компания и информация о доме</p>
          <span className="arrow">↗</span>
        </Link>
      </div>
      <section className="how">
        <h2>Одна заявка — любой вопрос</h2>
        <div>
          <span>01</span>
          <p>
            Выберите тип заявки
            <br />
            <small>Запрос, вопрос, экстренно, проблема или жалоба</small>
          </p>
        </div>
        <div>
          <span>02</span>
          <p>
            Расскажите подробнее
            <br />
            <small>Укажите адрес и при необходимости добавьте фото</small>
          </p>
        </div>
        <div>
          <span>03</span>
          <p>
            Следите за решением
            <br />
            <small>
              Изменения статуса и комментарии УК сохранятся в истории
            </small>
          </p>
        </div>
      </section>
      <Link className="my-requests" to="/admin">
        <span>
          <strong>Работаете в УК?</strong>
          <small>Открыть демонстрационный кабинет организации</small>
        </span>
        <span>→</span>
      </Link>
      <p className="demo-note">
        Демо для хакатона «Умный город» · Интеграции работают в тестовом режиме
      </p>
    </>
  );
}
