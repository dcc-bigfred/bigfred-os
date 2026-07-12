import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import AppShell from "./components/AppShell";
import ProtectedRoute from "./components/ProtectedRoute";
import LoginPage from "./pages/LoginPage";
import LogsPage from "./pages/LogsPage";
import SupervisordPage from "./pages/SupervisordPage";
import ServicesPage from "./pages/ServicesPage";
import RedisPage from "./pages/RedisPage";
import ConfigPage from "./pages/ConfigPage";
import AccountPage from "./pages/AccountPage";
import TerminalPage from "./pages/TerminalPage";

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<ProtectedRoute />}>
            <Route element={<AppShell />}>
              <Route index element={<Navigate to="/logs" replace />} />
              <Route path="supervisord" element={<SupervisordPage />} />
              <Route path="services" element={<ServicesPage />} />
              <Route path="redis" element={<RedisPage />} />
              <Route path="config" element={<ConfigPage />} />
              <Route path="account" element={<AccountPage />} />
              <Route path="logs" element={<LogsPage />} />
              <Route path="terminal" element={<TerminalPage />} />
            </Route>
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
