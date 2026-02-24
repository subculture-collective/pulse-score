import { useEffect, useState } from "react";
import {
  notificationPreferencesApi,
  alertsApi,
  type NotificationPreference,
  type AlertRule,
} from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import { Loader2, Bell, Mail, BookOpen } from "lucide-react";

export default function NotificationsTab() {
  const [prefs, setPrefs] = useState<NotificationPreference | null>(null);
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const toast = useToast();

  useEffect(() => {
    async function load() {
      try {
        const [prefRes, rulesRes] = await Promise.all([
          notificationPreferencesApi.get(),
          alertsApi.listRules(),
        ]);
        setPrefs(prefRes.data);
        setRules(rulesRes.data.rules ?? []);
      } catch {
        toast.error("Failed to load notification preferences");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function toggleField(field: keyof Pick<NotificationPreference, "email_enabled" | "in_app_enabled" | "digest_enabled">) {
    if (!prefs) return;
    setSaving(true);
    try {
      const { data } = await notificationPreferencesApi.update({
        [field]: !prefs[field],
      });
      setPrefs(data);
      toast.success("Preferences updated");
    } catch {
      toast.error("Failed to update preferences");
    } finally {
      setSaving(false);
    }
  }

  async function updateDigestFrequency(freq: string) {
    setSaving(true);
    try {
      const { data } = await notificationPreferencesApi.update({
        digest_frequency: freq,
      });
      setPrefs(data);
      toast.success("Digest frequency updated");
    } catch {
      toast.error("Failed to update preferences");
    } finally {
      setSaving(false);
    }
  }

  async function toggleMuteRule(ruleId: string) {
    if (!prefs) return;
    setSaving(true);
    const muted = prefs.muted_rule_ids ?? [];
    const newMuted = muted.includes(ruleId)
      ? muted.filter((id) => id !== ruleId)
      : [...muted, ruleId];
    try {
      const { data } = await notificationPreferencesApi.update({
        muted_rule_ids: newMuted,
      });
      setPrefs(data);
      toast.success(
        muted.includes(ruleId) ? "Rule unmuted" : "Rule muted"
      );
    } catch {
      toast.error("Failed to update preferences");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-indigo-600" />
      </div>
    );
  }

  if (!prefs) return null;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Notification Preferences
        </h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Control how and when you receive alert notifications.
        </p>
      </div>

      {/* Channel toggles */}
      <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
          Channels
        </h3>
        <div className="space-y-4">
          <Toggle
            icon={<Mail className="h-5 w-5" />}
            label="Email notifications"
            description="Receive alert notifications via email"
            enabled={prefs.email_enabled}
            saving={saving}
            onToggle={() => toggleField("email_enabled")}
          />
          <Toggle
            icon={<Bell className="h-5 w-5" />}
            label="In-app notifications"
            description="Show notifications in the app"
            enabled={prefs.in_app_enabled}
            saving={saving}
            onToggle={() => toggleField("in_app_enabled")}
          />
        </div>
      </div>

      {/* Digest settings */}
      <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
          Digest
        </h3>
        <div className="space-y-4">
          <Toggle
            icon={<BookOpen className="h-5 w-5" />}
            label="Email digest"
            description="Receive a periodic summary of alerts"
            enabled={prefs.digest_enabled}
            saving={saving}
            onToggle={() => toggleField("digest_enabled")}
          />
          {prefs.digest_enabled && (
            <div className="ml-10">
              <label className="text-sm text-gray-600 dark:text-gray-300">
                Frequency
              </label>
              <select
                value={prefs.digest_frequency}
                onChange={(e) => updateDigestFrequency(e.target.value)}
                disabled={saving}
                className="mt-1 block w-40 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 sm:text-sm"
              >
                <option value="daily">Daily</option>
                <option value="weekly">Weekly</option>
              </select>
            </div>
          )}
        </div>
      </div>

      {/* Muted rules */}
      {rules.length > 0 && (
        <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
            Mute individual rules
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">
            Muted rules will not trigger notifications for you.
          </p>
          <div className="space-y-3">
            {rules.map((rule) => {
              const isMuted = (prefs.muted_rule_ids ?? []).includes(rule.id);
              return (
                <div
                  key={rule.id}
                  className="flex items-center justify-between"
                >
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {rule.name}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {rule.trigger_type}
                    </p>
                  </div>
                  <button
                    onClick={() => toggleMuteRule(rule.id)}
                    disabled={saving}
                    className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                      isMuted
                        ? "bg-gray-200 text-gray-700 hover:bg-gray-300 dark:bg-gray-600 dark:text-gray-300 dark:hover:bg-gray-500"
                        : "bg-red-100 text-red-700 hover:bg-red-200 dark:bg-red-900/30 dark:text-red-400 dark:hover:bg-red-900/50"
                    }`}
                  >
                    {isMuted ? "Unmute" : "Mute"}
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

function Toggle({
  icon,
  label,
  description,
  enabled,
  saving,
  onToggle,
}: {
  icon: React.ReactNode;
  label: string;
  description: string;
  enabled: boolean;
  saving: boolean;
  onToggle: () => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <span className="text-gray-400 dark:text-gray-500">{icon}</span>
        <div>
          <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
            {label}
          </p>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            {description}
          </p>
        </div>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={enabled}
        disabled={saving}
        onClick={onToggle}
        className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50 ${
          enabled ? "bg-indigo-600" : "bg-gray-200 dark:bg-gray-600"
        }`}
      >
        <span
          className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
            enabled ? "translate-x-5" : "translate-x-0"
          }`}
        />
      </button>
    </div>
  );
}
