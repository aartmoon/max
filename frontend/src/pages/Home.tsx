import { Link } from "react-router-dom";
export default function Home() {
  return (
    <>
      <div className="eyebrow">ДОМА ВСЁ ПОД КОНТРОЛЕМ</div>
      <section className="hero">
        <div className="house-art" aria-hidden="true">
          <span>⌂</span>
          <i className="window one" />
          <i className="window two" />
          <i className="window three" />
          <i className="window four" />
          <b>✓</b>
        </div>
        <h1>
          Хороший дом
          <br />
          начинается с вас.
        </h1>
        <p>
          Решайте проблемы дома без путаницы.
          <br />
          Расскажите, что случилось — поможем
          <br className="desktop-break" /> направить обращение ответственным.
        </p>
        <Link className="button primary" to="/requests/new">
          Сообщить о проблеме <span>↗</span>
        </Link>
      </section>
      <div className="action-grid">
        <Link className="action-card" to="/requests/new?kind=APPLICATION">
          <span className="icon">▤</span>
          <h2>Заявка в УК</h2>
          <p>Обратиться в управляющую компанию</p>
          <span className="arrow">↗</span>
        </Link>
        <Link className="action-card" to="/requests/new?kind=QUESTION">
          <span className="icon">?</span>
          <h2>Задать вопрос</h2>
          <p>Уточнить, как решаются вопросы дома</p>
          <span className="arrow">↗</span>
        </Link>
      </div>
      <Link className="my-requests" to="/requests">
        <span>
          <strong>Мои обращения</strong>
          <small>Статусы и история — в одном месте</small>
        </span>
        <span>→</span>
      </Link>
      <div className="how">
        <h2>От проблемы к решению</h2>
        <div>
          <span>01</span>
          <p>
            Опишите ситуацию
            <br />
            <small>Можно добавить фото</small>
          </p>
        </div>
        <div>
          <span>02</span>
          <p>
            Узнайте ответственного
            <br />
            <small>Определим категорию и организацию</small>
          </p>
        </div>
        <div>
          <span>03</span>
          <p>
            Следите за обращением
            <br />
            <small>Сохраним каждый статус</small>
          </p>
        </div>
      </div>
      <p className="demo-note">
        Демо для хакатона «Умный город» · Интеграции работают в тестовом режиме
      </p>
    </>
  );
}
