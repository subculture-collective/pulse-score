import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { AuthProvider } from "@/contexts/AuthContext";
import { ThemeProvider } from "@/contexts/ThemeContext";
import { ToastProvider } from "@/contexts/ToastContext";
import ProtectedRoute from "@/components/ProtectedRoute";
import ErrorBoundary from "@/components/ErrorBoundary";
import AppLayout from "@/layouts/AppLayout";
import LandingPage from "@/pages/LandingPage";
import PricingPage from "@/pages/PricingPage";
import LoginPage from "@/pages/auth/LoginPage";
import RegisterPage from "@/pages/auth/RegisterPage";
import DashboardPage from "@/pages/DashboardPage";
import CustomersPage from "@/pages/CustomersPage";
import CustomerDetailPage from "@/pages/CustomerDetailPage";
import SettingsPage from "@/pages/settings/SettingsPage";
import OnboardingPage from "@/pages/OnboardingPage";
import PrivacyPage from "@/pages/PrivacyPage";
import TermsPage from "@/pages/TermsPage";
import NotFoundPage from "@/pages/NotFoundPage";

function App() {
  return (
    <BrowserRouter>
      <ThemeProvider>
        <AuthProvider>
          <ToastProvider>
            <ErrorBoundary>
              <Routes>
                {/* Public marketing + auth routes */}
                <Route path="/" element={<LandingPage />} />
                <Route path="/pricing" element={<PricingPage />} />
                <Route path="/login" element={<LoginPage />} />
                <Route path="/register" element={<RegisterPage />} />
                <Route path="/privacy" element={<PrivacyPage />} />
                <Route path="/terms" element={<TermsPage />} />

                {/* Backward-compatible auth aliases */}
                <Route path="/auth/login" element={<Navigate to="/login" replace />} />
                <Route path="/auth/register" element={<Navigate to="/register" replace />} />

                {/* Protected app routes */}
                <Route
                  element={
                    <ProtectedRoute>
                      <AppLayout />
                    </ProtectedRoute>
                  }
                >
                  <Route path="/dashboard" element={<DashboardPage />} />
                  <Route path="/onboarding" element={<OnboardingPage />} />
                  <Route path="/customers" element={<CustomersPage />} />
                  <Route
                    path="/customers/:id"
                    element={<CustomerDetailPage />}
                  />
                  <Route path="/settings/*" element={<SettingsPage />} />
                </Route>

                {/* Catch-all 404 */}
                <Route path="*" element={<NotFoundPage />} />
              </Routes>
            </ErrorBoundary>
          </ToastProvider>
        </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  );
}

export default App;
