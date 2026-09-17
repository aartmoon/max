import React from "react";
import ReactDOM from "react-dom/client";
import {
  BrowserRouter,
  Routes,
  Route,
  NavLink,
  Link,
  useLocation,
} from "react-router-dom";
import { useEffect } from "react";
import MyHouse from "./pages/MyHouse";
import Admin from "./pages/Admin";
import Home from "./pages/Home";
import NewRequest from "./pages/NewRequest";
import Requests from "./pages/Requests";
import RequestDetail from "./pages/RequestDetail";
import { MaxIntegration } from "./integration/MaxIntegration";
import { appBasename } from "./paths";
import "./styles.css";
function App() {
  const location = useLocation();
  useEffect(() => {
    window.scrollTo(0, 0);
  }, [location.pathname]);
  return (
    <>
      <header>
        <Link className="brand" to="/">
          <span>⌂</span> Твой дом<span className="brand-dot">.</span>
        </Link>
        <Link
          className="demo-label"
          to={location.pathname.startsWith("/admin") ? "/" : "/admin"}
        >
          {location.pathname.startsWith("/admin") ? "ЖИТЕЛЮ" : "КАБИНЕТ УК"}
        </Link>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/house" element={<MyHouse />} />
          <Route path="/admin" element={<Admin />} />
          <Route path="/admin/requests/:id" element={<RequestDetail admin />} />
          <Route path="/requests/new" element={<NewRequest />} />
          <Route path="/requests" element={<Requests />} />
          <Route path="/requests/:id" element={<RequestDetail />} />
          <Route
            path="*"
            element={
              <>
                <h1>Страница не найдена</h1>
                <Link to="/">На главную</Link>
              </>
            }
          />
        </Routes>
      </main>
      {!location.pathname.startsWith("/admin") && (
        <nav className="bottom-nav">
          <NavLink to="/" end>
            <span>⌂</span>Главная
          </NavLink>
          <NavLink to="/requests/new">
            <span>＋</span>Создать
          </NavLink>
          <NavLink to="/requests" end>
            <span>▤</span>Заявки
          </NavLink>
          <NavLink to="/house">
            <span>⌂</span>Мой дом
          </NavLink>
        </nav>
      )}
    </>
  );
}
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter basename={appBasename}>
      <MaxIntegration>
        <App />
      </MaxIntegration>
    </BrowserRouter>
  </React.StrictMode>,
);
