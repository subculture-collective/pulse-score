import { useMemo, useState } from "react";
import { Check, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";

type BillingCycle = "monthly" | "annual";

interface Plan {
  name: string;
  monthlyPrice: number;
  annualPrice: number;
  description: string;
  ctaLabel: string;
  ctaHref: string;
  featured?: boolean;
  limits: {
    customers: string;
    integrations: string;
    alerts: string;
    support: string;
  };
}

const plans: Plan[] = [
  {
    name: "Free",
    monthlyPrice: 0,
    annualPrice: 0,
    description: "Best for evaluating PulseScore with a small portfolio.",
    ctaLabel: "Get Started Free",
    ctaHref: "/register?plan=free",
    limits: {
      customers: "Up to 10 customers",
      integrations: "1 integration",
      alerts: "Basic alerts",
      support: "Community support",
    },
  },
  {
    name: "Growth",
    monthlyPrice: 49,
    annualPrice: 490,
    description: "For fast-moving teams managing churn at scale.",
    ctaLabel: "Start Growth Trial",
    ctaHref: "/register?plan=growth",
    featured: true,
    limits: {
      customers: "Up to 250 customers",
      integrations: "Up to 3 integrations",
      alerts: "Advanced alert rules",
      support: "Priority email support",
    },
  },
  {
    name: "Scale",
    monthlyPrice: 149,
    annualPrice: 1490,
    description: "For mature revenue teams with complex customer motion.",
    ctaLabel: "Start Scale Trial",
    ctaHref: "/register?plan=scale",
    limits: {
      customers: "Unlimited customers",
      integrations: "Unlimited integrations",
      alerts: "Advanced workflows",
      support: "Dedicated success partner",
    },
  },
];

export default function PricingSection() {
  const [cycle, setCycle] = useState<BillingCycle>("monthly");

  const annualSavingsText = useMemo(() => {
    const growthMonthlyAnnualized = plans[1].monthlyPrice * 12;
    const growthSavings = growthMonthlyAnnualized - plans[1].annualPrice;
    return `Save $${growthSavings}/yr`;
  }, []);

  return (
    <section id="pricing" className="bg-white px-6 py-16 dark:bg-gray-950 sm:px-10 lg:px-14 lg:py-24">
      <div className="mx-auto max-w-7xl">
        <div className="flex flex-col items-start justify-between gap-6 md:flex-row md:items-end">
          <div className="max-w-3xl">
            <h2 className="text-3xl font-bold tracking-tight text-gray-900 dark:text-gray-100 sm:text-4xl">
              Pricing that fits the gap between spreadsheets and $10K tools.
            </h2>
            <p className="mt-3 text-gray-600 dark:text-gray-300">
              Transparent tiers. No setup fees. Upgrade when your customer base
              grows.
            </p>
          </div>

          <div className="inline-flex items-center rounded-xl border border-gray-200 bg-gray-50 p-1 dark:border-gray-700 dark:bg-gray-900">
            <button
              onClick={() => setCycle("monthly")}
              className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
                cycle === "monthly"
                  ? "bg-white text-gray-900 shadow-sm dark:bg-gray-800 dark:text-gray-100"
                  : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              }`}
            >
              Monthly
            </button>
            <button
              onClick={() => setCycle("annual")}
              className={`rounded-lg px-3 py-1.5 text-sm font-medium transition ${
                cycle === "annual"
                  ? "bg-white text-gray-900 shadow-sm dark:bg-gray-800 dark:text-gray-100"
                  : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
              }`}
            >
              Annual
            </button>
            {cycle === "annual" && (
              <span className="ml-2 inline-flex items-center rounded-full bg-emerald-100 px-2 py-1 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                <Sparkles className="mr-1 h-3 w-3" />
                {annualSavingsText}
              </span>
            )}
          </div>
        </div>

        <div className="mt-10 grid grid-cols-1 gap-4 lg:grid-cols-3">
          {plans.map((plan) => {
            const isFree = plan.monthlyPrice === 0;
            const displayPrice = cycle === "monthly" ? plan.monthlyPrice : plan.annualPrice;
            const period = cycle === "monthly" ? "/mo" : "/yr";

            return (
              <article
                key={plan.name}
                className={`relative rounded-2xl border p-6 shadow-sm ${
                  plan.featured
                    ? "border-indigo-300 bg-indigo-50/50 dark:border-indigo-700 dark:bg-indigo-950/20"
                    : "border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900"
                }`}
              >
                {plan.featured && (
                  <span className="absolute -top-3 right-4 rounded-full bg-indigo-600 px-3 py-1 text-xs font-semibold text-white">
                    Most popular
                  </span>
                )}

                <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100">{plan.name}</h3>
                <p className="mt-2 text-sm text-gray-600 dark:text-gray-300">{plan.description}</p>

                <div className="mt-5 flex items-baseline gap-1">
                  <span className="text-4xl font-extrabold tracking-tight text-gray-900 dark:text-gray-100">
                    ${displayPrice}
                  </span>
                  <span className="text-sm text-gray-500 dark:text-gray-400">{period}</span>
                </div>
                {isFree && (
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">No credit card required</p>
                )}

                <Link
                  to={plan.ctaHref}
                  className={`mt-5 inline-flex w-full items-center justify-center rounded-xl px-4 py-2.5 text-sm font-semibold transition ${
                    plan.featured
                      ? "bg-indigo-600 text-white hover:bg-indigo-700"
                      : "border border-gray-300 bg-white text-gray-800 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-800"
                  }`}
                >
                  {plan.ctaLabel}
                </Link>

                <ul className="mt-5 space-y-2 text-sm text-gray-600 dark:text-gray-300">
                  {Object.values(plan.limits).map((item) => (
                    <li key={item} className="flex items-start gap-2">
                      <Check className="mt-0.5 h-4 w-4 shrink-0 text-indigo-500" />
                      {item}
                    </li>
                  ))}
                </ul>
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
}
