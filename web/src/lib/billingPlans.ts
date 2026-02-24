export type BillingCycle = "monthly" | "annual";
export type PlanTier = "free" | "growth" | "scale";

export interface BillingPlanDefinition {
  tier: PlanTier;
  name: string;
  description: string;
  monthlyPrice: number;
  annualPrice: number;
  limits: {
    customers: string;
    integrations: string;
    alerts: string;
    support: string;
  };
  featured?: boolean;
}

export const billingPlans: BillingPlanDefinition[] = [
  {
    tier: "free",
    name: "Free",
    monthlyPrice: 0,
    annualPrice: 0,
    description: "Best for evaluating PulseScore with a small portfolio.",
    limits: {
      customers: "Up to 10 customers",
      integrations: "1 integration",
      alerts: "Basic alerts",
      support: "Community support",
    },
  },
  {
    tier: "growth",
    name: "Growth",
    monthlyPrice: 49,
    annualPrice: 490,
    description: "For fast-moving teams managing churn at scale.",
    featured: true,
    limits: {
      customers: "Up to 250 customers",
      integrations: "Up to 3 integrations",
      alerts: "Advanced alert rules",
      support: "Priority email support",
    },
  },
  {
    tier: "scale",
    name: "Scale",
    monthlyPrice: 149,
    annualPrice: 1490,
    description: "For mature revenue teams with complex customer motion.",
    limits: {
      customers: "Unlimited customers",
      integrations: "Unlimited integrations",
      alerts: "Advanced workflows",
      support: "Dedicated success partner",
    },
  },
];

export function savingsBadge(plan: BillingPlanDefinition): string {
  const monthlyAnnualized = plan.monthlyPrice * 12;
  const delta = monthlyAnnualized - plan.annualPrice;
  return delta > 0 ? `Save $${delta}/yr` : "Annual billing";
}
