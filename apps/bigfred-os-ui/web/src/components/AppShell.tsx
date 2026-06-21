import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

export default function AppShell() {
  const { user, signOut } = useAuth();

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="app-brand">BigFred Hub OS</div>
        <nav className="app-nav">
          <NavLink to="/supervisord" className={({ isActive }) => (isActive ? "active" : "")}>
            Supervisord
          </NavLink>
          <NavLink to="/services" className={({ isActive }) => (isActive ? "active" : "")}>
            Services
          </NavLink>
          <NavLink to="/logs" className={({ isActive }) => (isActive ? "active" : "")}>
            Logs
          </NavLink>
          <NavLink to="/redis" className={({ isActive }) => (isActive ? "active" : "")}>
            Redis
          </NavLink>
          <NavLink to="/config" className={({ isActive }) => (isActive ? "active" : "")}>
            Config
          </NavLink>
        </nav>
        <div className="app-user">
          <span>{user?.username}</span>
          <button type="button" className="btn-ghost" onClick={() => void signOut()}>
            Sign out
          </button>
        </div>
      </header>
      <main className="app-main">
        <Outlet />
      </main>
    </div>
  );
}
