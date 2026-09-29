import React from "react";
import ReactDOM from "react-dom/client";
import {
  BrowserRouter,
  Routes,
  Route,
  NavLink,
  Link,
  Navigate,
  useLocation,
} from "react-router-dom";
import { useEffect } from "react";
import MyHouse from "./pages/MyHouse";
import Admin from "./pages/Admin";
import AdminSettings from "./pages/AdminSettings";
import Home from "./pages/Home";
import NewRequest from "./pages/NewRequest";
import Requests from "./pages/Requests";
import RequestDetail from "./pages/RequestDetail";
import { MaxIntegration } from "./integration/MaxIntegration";
import { AuthProvider, useAuth } from "./auth";
import { appBasename } from "./paths";
import "./styles.css";
function App() {
  const location = useLocation();
  const { user, logout } = useAuth();
  const canAdmin = user.roles?.some((role) => role === "manager" || role === "admin");
  useEffect(() => {
    window.scrollTo(0, 0);
  }, [location.pathname]);
  return (
    <>
      <header className="site-header">
        <Link className="brand" to="/">
          <span>⌂</span> Твой дом<span className="brand-dot">.</span>
        </Link>
        <div className="header-actions">
          {canAdmin && (
            <Link
              className="demo-label"
              to={location.pathname.startsWith("/admin") ? "/" : "/admin"}
            >
              {location.pathname.startsWith("/admin") ? "ЖИТЕЛЮ" : "КАБИНЕТ УК"}
            </Link>
          )}
          <button className="logout-button" type="button" onClick={logout}>
            Выйти
          </button>
        </div>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/house" element={<MyHouse />} />
          <Route path="/admin" element={<RequireOrganizationRole><Admin /></RequireOrganizationRole>} />
          <Route path="/admin/settings" element={<Navigate to="/admin/settings/users" replace />} />
          <Route path="/admin/settings/users" element={<RequireAdmin><AdminSettings section="users" /></RequireAdmin>} />
          <Route path="/admin/settings/organizations" element={<RequireAdmin><AdminSettings section="organizations" /></RequireAdmin>} />
          <Route path="/admin/settings/routing" element={<RequireAdmin><AdminSettings section="routing" /></RequireAdmin>} />
          <Route path="/admin/requests/:id" element={<RequireOrganizationRole><RequestDetail admin /></RequireOrganizationRole>} />
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
function RequireOrganizationRole({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  return user.roles?.some((role) => role === "manager" || role === "admin") ? children : <Navigate to="/" replace />;
}
function RequireAdmin({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  return user.roles?.includes("admin") ? children : <Navigate to="/admin" replace />;
}
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter basename={appBasename}>
      <AuthProvider>
        <MaxIntegration>
          <App />
        </MaxIntegration>
      </AuthProvider>
    </BrowserRouter>
  </React.StrictMode>,
);
