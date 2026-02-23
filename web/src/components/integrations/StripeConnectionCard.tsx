import { useCallback, useEffect, useState } from "react";
import { stripeApi, type StripeStatus } from "@/lib/stripe";

export default function StripeConnectionCard() {
  const [status, setStatus] = useState<StripeStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const fetchStatus = useCallback(async () => {
    try {
      const { data } = await stripeApi.getStatus();
      setStatus(data);
      setError("");
    } catch {
      setStatus(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  async function handleConnect() {
    setActionLoading(true);
    setError("");
    try {
      const { data } = await stripeApi.getConnectUrl();
      window.location.href = data.url;
    } catch {
      setError("Failed to start Stripe connection.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleDisconnect() {
    if (!confirm("Are you sure you want to disconnect Stripe?")) return;
    setActionLoading(true);
    setError("");
    try {
      await stripeApi.disconnect();
      setStatus(null);
      setMessage("Stripe disconnected.");
    } catch {
      setError("Failed to disconnect Stripe.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleSync() {
    setActionLoading(true);
    setError("");
    setMessage("");
    try {
      await stripeApi.triggerSync();
      setMessage("Sync started. This may take a few minutes.");
    } catch {
      setError("Failed to start sync.");
    } finally {
      setActionLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="rounded-lg border border-gray-200 bg-white p-6">
        <p className="text-sm text-gray-500">Loading Stripe status...</p>
      </div>
    );
  }

  const isConnected = status?.status === "active" || status?.status === "syncing";

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-100">
            <svg
              className="h-6 w-6 text-indigo-600"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M2.25 8.25h19.5M2.25 9h19.5m-16.5 5.25h6m-6 2.25h3m-3.75 3h15a2.25 2.25 0 002.25-2.25V6.75A2.25 2.25 0 0019.5 4.5h-15a2.25 2.25 0 00-2.25 2.25v10.5A2.25 2.25 0 004.5 19.5z"
              />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-gray-900">Stripe</h3>
            <p className="text-sm text-gray-500">
              Payment and subscription data
            </p>
          </div>
        </div>

        <StatusBadge status={status?.status} />
      </div>

      {error && (
        <div className="mt-4 rounded-md bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}
      {message && (
        <div className="mt-4 rounded-md bg-green-50 p-3 text-sm text-green-700">
          {message}
        </div>
      )}

      {isConnected && status && (
        <div className="mt-4 space-y-2 text-sm text-gray-600">
          {status.account_id && (
            <p>
              Account: <span className="font-mono">{status.account_id}</span>
            </p>
          )}
          {status.last_sync_at && (
            <p>
              Last sync:{" "}
              {new Date(status.last_sync_at).toLocaleString()}
            </p>
          )}
          {status.customer_count !== undefined && (
            <p>Customers synced: {status.customer_count}</p>
          )}
          {status.last_sync_error && (
            <p className="text-red-600">
              Last error: {status.last_sync_error}
            </p>
          )}
        </div>
      )}

      <div className="mt-6 flex gap-3">
        {isConnected ? (
          <>
            <button
              onClick={handleSync}
              disabled={actionLoading}
              className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {actionLoading ? "..." : "Sync Now"}
            </button>
            <button
              onClick={handleDisconnect}
              disabled={actionLoading}
              className="rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50"
            >
              Disconnect
            </button>
          </>
        ) : (
          <button
            onClick={handleConnect}
            disabled={actionLoading}
            className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {actionLoading ? "Connecting..." : "Connect Stripe"}
          </button>
        )}
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status?: string }) {
  if (!status || status === "disconnected") {
    return (
      <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-700">
        Not connected
      </span>
    );
  }
  if (status === "active") {
    return (
      <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-700">
        Connected
      </span>
    );
  }
  if (status === "syncing") {
    return (
      <span className="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-700">
        Syncing
      </span>
    );
  }
  if (status === "error") {
    return (
      <span className="inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-700">
        Error
      </span>
    );
  }
  return (
    <span className="inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs font-medium text-yellow-700">
      {status}
    </span>
  );
}
