import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import AppShell from "./components/AppShell";
import ProtectedRoute from "./components/ProtectedRoute";
import LoginPage from "./pages/LoginPage";
import LogsPage from "./pages/LogsPage";
import ServicesPage from "./pages/ServicesPage";
import PlaceholderPage from "./pages/PlaceholderPage";

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<ProtectedRoute />}>
            <Route element={<AppShell />}>
              <Route index element={<Navigate to="/logs" replace />} />
              <Route path="supervisord" element={<PlaceholderPage title="Supervisord" />} />
              <Route path="services" element={<ServicesPage />} />
              <Route path="logs" element={<LogsPage />} />
            </Route>
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
