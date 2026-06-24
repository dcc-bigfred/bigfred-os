import { FormEvent, useState } from "react";
import { ApiError, changePassword } from "../api/client";
import { useAuth } from "../auth/AuthContext";

export default function AccountPage() {
  const { user } = useAuth();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [pending, setPending] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(false);

    if (newPassword !== confirmPassword) {
      setError("New passwords do not match.");
      return;
    }
    if (newPassword.length < 4) {
      setError("New password must be at least 4 characters.");
      return;
    }

    setPending(true);
    try {
      await changePassword(currentPassword, newPassword);
      setSuccess(true);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (err) {
      if (err instanceof ApiError) {
        switch (err.code) {
          case "invalid_credentials":
            setError("Current password is incorrect.");
            break;
          case "password_change_failed":
            setError("Password change rejected by the system.");
            break;
          default:
            setError(err.detail ?? err.code);
        }
      } else {
        setError("Could not change password.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="account-page">
      <h2>Account</h2>
      <p className="account-lead">
        Signed in as <strong>{user?.username}</strong>. Change the Linux password used for SSH
        and this panel (PAM).
      </p>

      <form className="account-card" onSubmit={(e) => void onSubmit(e)}>
        {error ? <div className="login-error">{error}</div> : null}
        {success ? (
          <div className="account-success">Password updated successfully.</div>
        ) : null}

        <label htmlFor="current-password">Current password</label>
        <input
          id="current-password"
          name="current-password"
          type="password"
          autoComplete="current-password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          required
        />

        <label htmlFor="new-password">New password</label>
        <input
          id="new-password"
          name="new-password"
          type="password"
          autoComplete="new-password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          required
        />

        <label htmlFor="confirm-password">Confirm new password</label>
        <input
          id="confirm-password"
          name="confirm-password"
          type="password"
          autoComplete="new-password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          required
        />

        <button type="submit" disabled={pending}>
          {pending ? "Updating…" : "Change password"}
        </button>
      </form>
    </div>
  );
}
