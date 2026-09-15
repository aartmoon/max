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
import Home from "./pages/Home";
import NewRequest from "./pages/NewRequest";
import Requests from "./pages/Requests";
import RequestDetail from "./pages/RequestDetail";
import { maxBridge } from "./integration/max";
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
        <span className="demo-label">ДЕМО</span>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<Home />} />
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
      <nav className="bottom-nav">
        <NavLink to="/" end>
          <span>⌂</span>Главная
        </NavLink>
        <NavLink to="/requests/new">
          <span>＋</span>Создать
        </NavLink>
        <NavLink to="/requests" end>
          <span>▤</span>Обращения
        </NavLink>
      </nav>
    </>
  );
}
maxBridge.ready();
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
);
