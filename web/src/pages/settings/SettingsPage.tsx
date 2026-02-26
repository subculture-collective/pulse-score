import {
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import OrganizationTab from "@/pages/settings/OrganizationTab";
import ProfileTab from "@/pages/settings/ProfileTab";
import IntegrationsTab from "@/pages/settings/IntegrationsTab";
import ScoringTab from "@/pages/settings/ScoringTab";
import BillingTab from "@/pages/settings/BillingTab";
import TeamTab from "@/pages/settings/TeamTab";
import AlertsTab from "@/pages/settings/AlertsTab";
import NotificationsTab from "@/pages/settings/NotificationsTab";
import StripeCallbackPage from "@/pages/settings/StripeCallbackPage";
import HubSpotCallbackPage from "@/pages/settings/HubSpotCallbackPage";
import IntercomCallbackPage from "@/pages/settings/IntercomCallbackPage";

const tabs = [
  { path: "organization", label: "Organization" },
  { path: "profile", label: "Profile" },
  { path: "integrations", label: "Integrations" },
  { path: "scoring", label: "Scoring" },
  { path: "billing", label: "Billing" },
  { path: "team", label: "Team" },
  { path: "alerts", label: "Alerts" },
  { path: "notifications", label: "Notifications" },
];

export default function SettingsPage() {
  const location = useLocation();
  const navigate = useNavigate();

  const currentTab =
    tabs.find((t) => location.pathname.includes(`/settings/${t.path}`))?.path ??
    "organization";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--galdr-fg)]">Settings</h1>
        <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
          Manage your organization, profile, and integrations.
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-[var(--galdr-border)]">
        <nav className="-mb-px flex gap-4 overflow-x-auto">
          {tabs.map((tab) => (
            <button
              key={tab.path}
              onClick={() => navigate(`/settings/${tab.path}`)}
              className={`whitespace-nowrap border-b-2 px-1 pb-3 text-sm font-medium transition-colors ${
                currentTab === tab.path
                  ? "border-[var(--galdr-accent)] text-[var(--galdr-accent)]"
                  : "border-transparent text-[var(--galdr-fg-muted)] hover:text-[var(--galdr-fg)]"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab content */}
      <Routes>
        <Route path="organization" element={<OrganizationTab />} />
        <Route path="profile" element={<ProfileTab />} />
        <Route path="integrations" element={<IntegrationsTab />} />
        <Route path="scoring" element={<ScoringTab />} />
        <Route path="billing" element={<BillingTab />} />
        <Route path="team" element={<TeamTab />} />
        <Route path="alerts" element={<AlertsTab />} />
        <Route path="notifications" element={<NotificationsTab />} />
        <Route path="stripe/callback" element={<StripeCallbackPage />} />
        <Route path="hubspot/callback" element={<HubSpotCallbackPage />} />
        <Route path="intercom/callback" element={<IntercomCallbackPage />} />
        <Route index element={<Navigate to="organization" replace />} />
      </Routes>
    </div>
  );
}
