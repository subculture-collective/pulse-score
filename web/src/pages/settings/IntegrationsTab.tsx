import { useCallback, useEffect, useState } from "react";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import StripeConnectionCard from "@/components/integrations/StripeConnectionCard";
import IntegrationCard from "@/components/IntegrationCard";
import { Loader2 } from "lucide-react";

interface IntegrationConnection {
  id: string;
  provider: string;
  status: string;
  last_sync_at?: string;
  customer_count?: number;
}

export default function IntegrationsTab() {
  const [integrations, setIntegrations] = useState<IntegrationConnection[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  const fetchIntegrations = useCallback(async () => {
    try {
      const { data } = await api.get<{ integrations: IntegrationConnection[] }>(
        "/integrations",
      );
      setIntegrations(data.integrations ?? []);
    } catch {
      toast.error("Failed to load integrations");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    fetchIntegrations();
  }, [fetchIntegrations]);

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
      </div>
    );
  }

  // Filter out Stripe from generic list since StripeConnectionCard has its own handling
  const otherIntegrations = integrations.filter((i) => i.provider !== "stripe");

  return (
    <div className="space-y-6">
      <div>
        <h3 className="mb-4 text-sm font-medium text-gray-900 dark:text-gray-100">
          Stripe
        </h3>
        <StripeConnectionCard />
      </div>

      {otherIntegrations.length > 0 && (
        <div>
          <h3 className="mb-4 text-sm font-medium text-gray-900 dark:text-gray-100">
            Other Integrations
          </h3>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            {otherIntegrations.map((integration) => (
              <IntegrationCard
                key={integration.id}
                provider={integration.provider}
                status={
                  integration.status as
                    | "connected"
                    | "syncing"
                    | "error"
                    | "disconnected"
                }
                lastSyncAt={integration.last_sync_at}
                customerCount={integration.customer_count}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
