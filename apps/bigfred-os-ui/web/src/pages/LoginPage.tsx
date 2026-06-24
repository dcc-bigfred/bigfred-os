import { FormEvent, useState } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { ApiError, login } from "../api/client";
import { useAuth } from "../auth/AuthContext";

interface LocationState {
  from?: { pathname?: string };
}

export default function LoginPage() {
  const { user, setUser } = useAuth();
  const location = useLocation();
  const [username, setUsername] = useState("root");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  if (user) {
    const dest = (location.state as LocationState | undefined)?.from?.pathname ?? "/logs";
    return <Navigate to={dest} replace />;
  }

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setPending(true);
    try {
      const me = await login(username.trim(), password);
      setUser(me);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.code === "invalid_credentials" ? "Invalid username or password." : err.code);
      } else {
        setError("Could not connect to the server.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="login-page">
      <form className="login-card" onSubmit={(e) => void onSubmit(e)}>
        <h1>BigFred Hub OS</h1>
        <p>Hub administration panel</p>
        {error ? <div className="login-error">{error}</div> : null}
        <label htmlFor="username">Username</label>
        <input
          id="username"
          name="username"
          autoComplete="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
        />
        <label htmlFor="password">Password</label>
        <input
          id="password"
          name="password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        <button type="submit" disabled={pending}>
          {pending ? "Signing in…" : "Sign in"}
        </button>
      </form>
    </div>
  );
}
