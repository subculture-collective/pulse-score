import IntegrationStatusBadge from "@/components/IntegrationStatusBadge";
import { RefreshCw, Unplug } from "lucide-react";

interface IntegrationCardProps {
  provider: string;
  status: "connected" | "syncing" | "error" | "disconnected";
  lastSyncAt?: string;
  customerCount?: number;
  onSync?: () => void;
  onDisconnect?: () => void;
}

function relativeTime(dateStr?: string): string {
  if (!dateStr) return "Never";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "Just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export default function IntegrationCard({
  provider,
  status,
  lastSyncAt,
  customerCount,
  onSync,
  onDisconnect,
}: IntegrationCardProps) {
  const isActive = status === "connected" || status === "syncing";

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold capitalize text-gray-900 dark:text-gray-100">
            {provider}
          </h3>
          <IntegrationStatusBadge status={status} />
        </div>
      </div>

      {isActive && (
        <div className="mt-4 space-y-1 text-xs text-gray-500 dark:text-gray-400">
          <p>Last sync: {relativeTime(lastSyncAt)}</p>
          {customerCount !== undefined && (
            <p>
              {customerCount} customer{customerCount !== 1 ? "s" : ""}
            </p>
          )}
        </div>
      )}

      {isActive && (
        <div className="mt-4 flex gap-2">
          {onSync && (
            <button
              onClick={onSync}
              disabled={status === "syncing"}
              className="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
            >
              <RefreshCw className="h-3 w-3" />
              Sync Now
            </button>
          )}
          {onDisconnect && (
            <button
              onClick={onDisconnect}
              className="inline-flex items-center gap-1 rounded-lg border border-red-300 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 dark:border-red-700 dark:text-red-400 dark:hover:bg-red-950"
            >
              <Unplug className="h-3 w-3" />
              Disconnect
            </button>
          )}
        </div>
      )}
    </div>
  );
}
