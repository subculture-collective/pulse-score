import { useCallback, useEffect, useState } from "react";
import {
  alertsApi,
  type AlertRule,
  type AlertHistory,
  type CreateAlertRulePayload,
} from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import {
  Bell,
  Loader2,
  Plus,
  Trash2,
  Power,
  PowerOff,
  ChevronDown,
  ChevronUp,
} from "lucide-react";

const TRIGGER_TYPES = [
  { value: "score_below", label: "Score Below Threshold" },
  { value: "score_drop", label: "Score Drop" },
  { value: "risk_change", label: "Risk Level Change" },
  { value: "payment_failed", label: "Payment Failed" },
];

function triggerLabel(type: string): string {
  return TRIGGER_TYPES.find((t) => t.value === type)?.label ?? type;
}

function statusBadge(status: string) {
  const colors: Record<string, string> = {
    sent: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
    failed: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200",
    pending:
      "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200",
  };
  return (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${colors[status] ?? "bg-gray-100 text-gray-800"}`}
    >
      {status}
    </span>
  );
}

// ── Create/Edit Form ────────────────────────────────
interface RuleFormProps {
  onSave: (data: CreateAlertRulePayload) => Promise<void>;
  onCancel: () => void;
  initial?: AlertRule;
  saving: boolean;
}

function RuleForm({ onSave, onCancel, initial, saving }: RuleFormProps) {
  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [triggerType, setTriggerType] = useState(
    initial?.trigger_type ?? "score_below",
  );
  const [recipients, setRecipients] = useState(
    initial?.recipients?.join(", ") ?? "",
  );
  const [threshold, setThreshold] = useState(
    String((initial?.conditions?.threshold as number) ?? 40),
  );
  const [points, setPoints] = useState(
    String((initial?.conditions?.points as number) ?? 10),
  );
  const [days, setDays] = useState(
    String((initial?.conditions?.days as number) ?? 7),
  );

  function buildConditions(): Record<string, unknown> {
    switch (triggerType) {
      case "score_below":
        return { threshold: Number(threshold) };
      case "score_drop":
        return { points: Number(points), days: Number(days) };
      case "risk_change":
        return {};
      case "payment_failed":
        return {};
      default:
        return {};
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await onSave({
      name,
      description,
      trigger_type: triggerType,
      conditions: buildConditions(),
      channel: "email",
      recipients: recipients
        .split(",")
        .map((r) => r.trim())
        .filter(Boolean),
    });
  }

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500";
  const labelCls = "block text-sm font-medium text-gray-700 dark:text-gray-300";

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-4 rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800"
    >
      <div>
        <label className={labelCls}>Name</label>
        <input
          className={inputCls}
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div>
        <label className={labelCls}>Description</label>
        <input
          className={inputCls}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <div>
        <label className={labelCls}>Trigger Type</label>
        <select
          className={inputCls}
          value={triggerType}
          onChange={(e) => setTriggerType(e.target.value)}
        >
          {TRIGGER_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </div>

      {triggerType === "score_below" && (
        <div>
          <label className={labelCls}>Score Threshold</label>
          <input
            className={inputCls}
            type="number"
            min={1}
            max={100}
            required
            value={threshold}
            onChange={(e) => setThreshold(e.target.value)}
          />
          <p className="mt-1 text-xs text-gray-500">
            Alert when score drops below this value
          </p>
        </div>
      )}

      {triggerType === "score_drop" && (
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className={labelCls}>Points Drop</label>
            <input
              className={inputCls}
              type="number"
              min={1}
              required
              value={points}
              onChange={(e) => setPoints(e.target.value)}
            />
          </div>
          <div>
            <label className={labelCls}>Within Days</label>
            <input
              className={inputCls}
              type="number"
              min={1}
              required
              value={days}
              onChange={(e) => setDays(e.target.value)}
            />
          </div>
        </div>
      )}

      <div>
        <label className={labelCls}>Recipients (comma-separated emails)</label>
        <input
          className={inputCls}
          required
          value={recipients}
          onChange={(e) => setRecipients(e.target.value)}
          placeholder="admin@example.com, csm@example.com"
        />
      </div>

      <div className="flex gap-3">
        <button
          type="submit"
          disabled={saving}
          className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {saving ? "Saving…" : initial ? "Update Rule" : "Create Rule"}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
        >
          Cancel
        </button>
      </div>
    </form>
  );
}

// ── Main Tab ────────────────────────────────────────
export default function AlertsTab() {
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [history, setHistory] = useState<AlertHistory[]>([]);
  const [stats, setStats] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [saving, setSaving] = useState(false);
  const [expandedRule, setExpandedRule] = useState<string | null>(null);
  const [ruleHistory, setRuleHistory] = useState<AlertHistory[]>([]);
  const toast = useToast();

  const fetchData = useCallback(async () => {
    try {
      const [rulesRes, historyRes, statsRes] = await Promise.all([
        alertsApi.listRules(),
        alertsApi.listHistory({ limit: 10 }),
        alertsApi.getStats(),
      ]);
      setRules(rulesRes.data.rules ?? []);
      setHistory(historyRes.data.history ?? []);
      setStats(statsRes.data ?? {});
    } catch {
      toast.error("Failed to load alert data");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleCreate(data: CreateAlertRulePayload) {
    setSaving(true);
    try {
      await alertsApi.createRule(data);
      toast.success("Alert rule created");
      setShowForm(false);
      await fetchData();
    } catch {
      toast.error("Failed to create alert rule");
    } finally {
      setSaving(false);
    }
  }

  async function toggleActive(rule: AlertRule) {
    try {
      await alertsApi.updateRule(rule.id, { is_active: !rule.is_active });
      toast.success(rule.is_active ? "Rule disabled" : "Rule enabled");
      await fetchData();
    } catch {
      toast.error("Failed to update rule");
    }
  }

  async function deleteRule(id: string) {
    try {
      await alertsApi.deleteRule(id);
      toast.success("Alert rule deleted");
      await fetchData();
    } catch {
      toast.error("Failed to delete rule");
    }
  }

  async function toggleRuleHistory(ruleId: string) {
    if (expandedRule === ruleId) {
      setExpandedRule(null);
      setRuleHistory([]);
      return;
    }
    try {
      const { data } = await alertsApi.listRuleHistory(ruleId, { limit: 5 });
      setRuleHistory(data.history ?? []);
      setExpandedRule(ruleId);
    } catch {
      toast.error("Failed to load rule history");
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {[
          { label: "Total Rules", value: rules.length },
          { label: "Sent", value: stats.sent ?? 0 },
          { label: "Failed", value: stats.failed ?? 0 },
          { label: "Pending", value: stats.pending ?? 0 },
        ].map((s) => (
          <div
            key={s.label}
            className="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800"
          >
            <p className="text-xs font-medium text-gray-500 dark:text-gray-400">
              {s.label}
            </p>
            <p className="mt-1 text-2xl font-bold text-gray-900 dark:text-gray-100">
              {s.value}
            </p>
          </div>
        ))}
      </div>

      {/* Alert Rules */}
      <div>
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Alert Rules
          </h3>
          <button
            onClick={() => setShowForm(true)}
            className="inline-flex items-center gap-1.5 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            <Plus className="h-4 w-4" /> New Rule
          </button>
        </div>

        {showForm && (
          <RuleForm
            onSave={handleCreate}
            onCancel={() => setShowForm(false)}
            saving={saving}
          />
        )}

        {rules.length === 0 && !showForm ? (
          <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-12 dark:border-gray-600">
            <Bell className="h-8 w-8 text-gray-400" />
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
              No alert rules yet
            </p>
            <button
              onClick={() => setShowForm(true)}
              className="mt-3 text-sm font-medium text-indigo-600 hover:text-indigo-700 dark:text-indigo-400"
            >
              Create your first rule
            </button>
          </div>
        ) : (
          <div className="space-y-3">
            {rules.map((rule) => (
              <div
                key={rule.id}
                className="rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800"
              >
                <div className="flex items-center justify-between p-4">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <h4 className="font-medium text-gray-900 dark:text-gray-100">
                        {rule.name}
                      </h4>
                      <span
                        className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${rule.is_active ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200" : "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"}`}
                      >
                        {rule.is_active ? "Active" : "Disabled"}
                      </span>
                    </div>
                    <p className="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
                      {triggerLabel(rule.trigger_type)} ·{" "}
                      {rule.recipients.length} recipient
                      {rule.recipients.length !== 1 ? "s" : ""}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => toggleRuleHistory(rule.id)}
                      title="View history"
                      className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-200"
                    >
                      {expandedRule === rule.id ? (
                        <ChevronUp className="h-4 w-4" />
                      ) : (
                        <ChevronDown className="h-4 w-4" />
                      )}
                    </button>
                    <button
                      onClick={() => toggleActive(rule)}
                      title={rule.is_active ? "Disable" : "Enable"}
                      className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-200"
                    >
                      {rule.is_active ? (
                        <PowerOff className="h-4 w-4" />
                      ) : (
                        <Power className="h-4 w-4" />
                      )}
                    </button>
                    <button
                      onClick={() => deleteRule(rule.id)}
                      title="Delete"
                      className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
                {expandedRule === rule.id && (
                  <div className="border-t border-gray-200 px-4 py-3 dark:border-gray-700">
                    {ruleHistory.length === 0 ? (
                      <p className="text-sm text-gray-500">
                        No alerts sent for this rule yet.
                      </p>
                    ) : (
                      <div className="space-y-2">
                        {ruleHistory.map((h) => (
                          <div
                            key={h.id}
                            className="flex items-center justify-between text-sm"
                          >
                            <span className="text-gray-600 dark:text-gray-400">
                              {new Date(h.created_at).toLocaleString()}
                            </span>
                            {statusBadge(h.status)}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Recent Alert History */}
      <div>
        <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
          Recent Alerts
        </h3>
        {history.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            No alerts sent yet.
          </p>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-gray-200 bg-gray-50 text-xs uppercase text-gray-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400">
                <tr>
                  <th className="px-4 py-3">Date</th>
                  <th className="px-4 py-3">Channel</th>
                  <th className="px-4 py-3">Status</th>
                </tr>
              </thead>
              <tbody>
                {history.map((h) => (
                  <tr
                    key={h.id}
                    className="border-b border-gray-100 dark:border-gray-800"
                  >
                    <td className="px-4 py-3 text-gray-700 dark:text-gray-300">
                      {new Date(h.created_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-gray-500 dark:text-gray-400">
                      {h.channel}
                    </td>
                    <td className="px-4 py-3">{statusBadge(h.status)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
