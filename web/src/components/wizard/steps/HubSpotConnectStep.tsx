import WizardStep from "@/components/wizard/WizardStep";

interface HubSpotConnectStepProps {
  connected: boolean;
  loading: boolean;
  statusText: string;
  accountId?: string;
  error?: string;
  onConnect: () => void;
  onRefresh: () => void;
}

export default function HubSpotConnectStep({
  connected,
  loading,
  statusText,
  accountId,
  error,
  onConnect,
  onRefresh,
}: HubSpotConnectStepProps) {
  return (
    <WizardStep
      title="Connect HubSpot (optional)"
      description="HubSpot data enriches customer context with contacts, deals, and company attributes."
    >
      <div className="rounded-lg border border-gray-200 bg-gray-50 p-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-sm font-medium text-gray-800">Status: {statusText}</p>
            {accountId && (
              <p className="mt-1 text-xs text-gray-500">Portal ID: {accountId}</p>
            )}
          </div>
          <span
            className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
              connected
                ? "bg-green-100 text-green-700"
                : "bg-gray-200 text-gray-700"
            }`}
          >
            {connected ? "Connected" : "Not connected"}
          </span>
        </div>
      </div>

      {error && (
        <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="mt-5 flex gap-2">
        {!connected && (
          <button
            onClick={onConnect}
            disabled={loading}
            className="rounded-md bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700 disabled:opacity-50"
          >
            {loading ? "Connecting..." : "Connect HubSpot"}
          </button>
        )}
        <button
          onClick={onRefresh}
          disabled={loading}
          className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        >
          Refresh status
        </button>
      </div>
    </WizardStep>
  );
}
