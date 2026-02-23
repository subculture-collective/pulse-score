import BaseLayout from "@/components/BaseLayout";
import StripeConnectionCard from "@/components/integrations/StripeConnectionCard";

export default function SettingsPage() {
  return (
    <BaseLayout>
      <div className="space-y-6">
        <div>
          <h2 className="text-2xl font-semibold text-gray-900">Settings</h2>
          <p className="mt-1 text-sm text-gray-600">
            Manage your integrations and account settings.
          </p>
        </div>

        <section>
          <h3 className="mb-4 text-lg font-medium text-gray-900">
            Integrations
          </h3>
          <div className="space-y-4">
            <StripeConnectionCard />
          </div>
        </section>
      </div>
    </BaseLayout>
  );
}
