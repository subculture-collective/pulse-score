import PricingSection from "@/components/landing/PricingSection";
import SeoMeta from "@/components/SeoMeta";

export default function PricingPage() {
  return (
    <div className="min-h-screen bg-white dark:bg-gray-950">
      <SeoMeta
        title="PulseScore Pricing"
        description="Compare PulseScore Free, Growth, and Scale plans with monthly and annual billing options."
        path="/pricing"
      />
      <PricingSection showStandaloneHeader />
    </div>
  );
}
