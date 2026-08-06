import { useEffect, useRef, useState } from "react";
import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";

function MenuIcon({ open }: { open: boolean }) {
  if (open) {
    return (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path
          d="M6 6l12 12M18 6L6 18"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
        />
      </svg>
    );
  }
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M4 7h16M4 12h16M4 17h16"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  );
}

export default function AppShell() {
  const { user, signOut } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const headerRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node | null;
      if (headerRef.current && target && !headerRef.current.contains(target)) {
        setMenuOpen(false);
      }
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setMenuOpen(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [menuOpen]);

  const closeMenu = () => setMenuOpen(false);

  return (
    <div className="app-shell">
      <header className="app-header" ref={headerRef}>
        <div className="app-brand">BigFred Hub OS</div>
        <button
          type="button"
          className="app-menu-toggle"
          aria-label={menuOpen ? "Close menu" : "Open menu"}
          aria-expanded={menuOpen}
          aria-controls="app-nav"
          onClick={() => setMenuOpen((v) => !v)}
        >
          <MenuIcon open={menuOpen} />
        </button>
        <nav id="app-nav" className={`app-nav${menuOpen ? " open" : ""}`}>
          <NavLink
            to="/services"
            className={({ isActive }) => (isActive ? "active" : "")}
            onClick={closeMenu}
          >
            Services
          </NavLink>
          <NavLink
            to="/logs"
            className={({ isActive }) => (isActive ? "active" : "")}
            onClick={closeMenu}
          >
            Logs
          </NavLink>
          <NavLink
            to="/terminal"
            className={({ isActive }) => (isActive ? "active" : "")}
            onClick={closeMenu}
          >
            Terminal
          </NavLink>
          <NavLink
            to="/redis"
            className={({ isActive }) => (isActive ? "active" : "")}
            onClick={closeMenu}
          >
            Redis
          </NavLink>
          <NavLink
            to="/config"
            className={({ isActive }) => (isActive ? "active" : "")}
            onClick={closeMenu}
          >
            Config
          </NavLink>
          <NavLink
            to="/time"
            className={({ isActive }) => (isActive ? "active" : "")}
            onClick={closeMenu}
          >
            Time
          </NavLink>
          <NavLink
            to="/account"
            className={({ isActive }) => (isActive ? "active" : "")}
            onClick={closeMenu}
          >
            Account
          </NavLink>
          <NavLink
            to="/update"
            className={({ isActive }) => (isActive ? "active" : "")}
            onClick={closeMenu}
          >
            Update
          </NavLink>
          <div className="app-nav-user">
            <span className="app-nav-username">{user?.username}</span>
            <button type="button" className="btn-ghost" onClick={() => void signOut()}>
              Sign out
            </button>
          </div>
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
