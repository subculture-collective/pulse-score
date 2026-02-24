import WizardStep from "@/components/wizard/WizardStep";

export interface ScoreBucket {
  range: string;
  count: number;
}

export interface AtRiskCustomerPreview {
  id: string;
  name: string;
  health_score: number;
}

interface ScorePreviewStepProps {
  connectedProviders: string[];
  syncStatus: Record<string, string>;
  loading: boolean;
  distribution: ScoreBucket[];
  atRiskCustomers: AtRiskCustomerPreview[];
}

export default function ScorePreviewStep({
  connectedProviders,
  syncStatus,
  loading,
  distribution,
  atRiskCustomers,
}: ScorePreviewStepProps) {
  return (
    <WizardStep
      title="Generate first score preview"
      description="We’ll run an initial sync and show a quick snapshot of customer health."
    >
      {connectedProviders.length === 0 ? (
        <div className="rounded-lg border border-yellow-300 bg-yellow-50 p-4 text-sm text-yellow-800">
          No integrations connected yet. You can finish onboarding now and connect
          data sources later from Settings.
        </div>
      ) : (
        <>
          <div className="rounded-lg border border-gray-200 bg-gray-50 p-4">
            <h3 className="text-sm font-semibold text-gray-800">Sync status</h3>
            <ul className="mt-2 space-y-1 text-sm text-gray-700">
              {connectedProviders.map((provider) => (
                <li key={provider} className="flex items-center justify-between">
                  <span className="capitalize">{provider}</span>
                  <span className="font-medium">{syncStatus[provider] ?? "pending"}</span>
                </li>
              ))}
            </ul>
          </div>

          {loading && (
            <p className="mt-3 text-sm text-gray-600">
              Fetching preview data... this can take a minute.
            </p>
          )}

          {!loading && distribution.length > 0 && (
            <div className="mt-5 rounded-lg border border-gray-200 p-4">
              <h4 className="text-sm font-semibold text-gray-800">
                Score distribution preview
              </h4>
              <div className="mt-3 space-y-2">
                {distribution.map((bucket) => (
                  <div key={bucket.range} className="flex items-center gap-3">
                    <span className="w-16 text-xs text-gray-500">{bucket.range}</span>
                    <div className="h-2 flex-1 rounded bg-gray-100">
                      <div
                        className="h-2 rounded bg-indigo-500"
                        style={{
                          width: `${Math.min(100, bucket.count * 12)}%`,
                        }}
                      />
                    </div>
                    <span className="w-8 text-right text-xs text-gray-600">
                      {bucket.count}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {!loading && atRiskCustomers.length > 0 && (
            <div className="mt-5 rounded-lg border border-red-200 bg-red-50 p-4">
              <h4 className="text-sm font-semibold text-red-800">
                Top at-risk customers
              </h4>
              <ul className="mt-2 space-y-1 text-sm text-red-700">
                {atRiskCustomers.map((customer) => (
                  <li key={customer.id} className="flex items-center justify-between">
                    <span>{customer.name}</span>
                    <span className="font-semibold">{customer.health_score}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </WizardStep>
  );
}
