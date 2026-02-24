import { useCallback, useEffect, useMemo, useState } from "react";
import { ExternalLink, Loader2 } from "lucide-react";

import { useToast } from "@/contexts/ToastContext";
import { useCheckout } from "@/hooks/useCheckout";
import { billingApi, type BillingSubscriptionResponse } from "@/lib/api";
import {
  billingPlans,
  type BillingCycle,
  type PlanTier,
} from "@/lib/billingPlans";

interface SubscriptionManagerProps {
  checkoutState?: "success" | "cancelled" | null;
}

function normalizeTier(value?: string): PlanTier {
  const normalized = value?.toLowerCase();
  if (normalized === "growth" || normalized === "scale") return normalized;
  return "free";
}

function formatRenewalDate(date: string | null): string {
  if (!date) return "—";
  const parsed = new Date(date);
  if (Number.isNaN(parsed.getTime())) return "—";
  return parsed.toLocaleDateString();
}

function formatLimit(limit: number): string {
  return limit < 0 ? "∞" : String(limit);
}

function usagePercent(used: number, limit: number): number {
  if (limit <= 0) return 0;
  return Math.min(100, Math.round((used / limit) * 100));
}

export default function SubscriptionManager({
  checkoutState,
}: SubscriptionManagerProps) {
  const [subscription, setSubscription] =
    useState<BillingSubscriptionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [cancelling, setCancelling] = useState(false);
  const [openingPortal, setOpeningPortal] = useState(false);

  const { loading: checkoutLoading, startCheckout } = useCheckout();
  const toast = useToast();

  const currentTier = normalizeTier(subscription?.tier);
  const cycle = (subscription?.billing_cycle ?? "monthly") as BillingCycle;

  const fetchSubscription = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await billingApi.getSubscription();
      setSubscription(data);
    } catch {
      toast.error("Failed to load subscription details");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void fetchSubscription();
  }, [fetchSubscription]);

  useEffect(() => {
    if (checkoutState === "success") {
      toast.success("Checkout complete. Subscription is refreshing...");
      void fetchSubscription();
    }
    if (checkoutState === "cancelled") {
      toast.info("Checkout cancelled. No changes were made.");
    }
  }, [checkoutState, fetchSubscription, toast]);

  const recommendedPlans = useMemo(
    () => billingPlans.filter((plan) => plan.tier !== "free"),
    [],
  );

  async function handleOpenPortal() {
    setOpeningPortal(true);
    try {
      const { data } = await billingApi.createPortalSession();
      window.location.href = data.url;
    } catch {
      toast.error("Unable to open Stripe customer portal right now.");
    } finally {
      setOpeningPortal(false);
    }
  }

  async function handleCancelAtPeriodEnd() {
    if (
      !window.confirm(
        "Cancel this subscription at the end of the current billing period?",
      )
    ) {
      return;
    }

    setCancelling(true);
    try {
      await billingApi.cancelAtPeriodEnd();
      toast.success("Your subscription will cancel at period end.");
      await fetchSubscription();
    } catch {
      toast.error("Failed to schedule cancellation. Please try again.");
    } finally {
      setCancelling(false);
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-10">
        <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!subscription) {
    return null;
  }

  return (
    <div className="space-y-6">
      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {currentTier[0].toUpperCase() + currentTier.slice(1)} plan
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Status:{" "}
              <span className="font-medium text-gray-700 dark:text-gray-200">
                {subscription.status}
              </span>
              {" · "}
              Cycle:{" "}
              <span className="font-medium text-gray-700 dark:text-gray-200">
                {cycle}
              </span>
              {" · "}
              Renewal:{" "}
              <span className="font-medium text-gray-700 dark:text-gray-200">
                {formatRenewalDate(subscription.renewal_date)}
              </span>
            </p>
            {subscription.cancel_at_period_end && (
              <p className="mt-2 text-xs font-medium text-amber-600 dark:text-amber-300">
                This subscription is scheduled to cancel at period end.
              </p>
            )}
          </div>

          <div className="flex flex-wrap gap-2">
            <button
              onClick={handleOpenPortal}
              disabled={openingPortal}
              className="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-60 dark:border-gray-700 dark:text-gray-200 dark:hover:bg-gray-800"
            >
              {openingPortal ? "Opening..." : "Open customer portal"}
              <ExternalLink className="h-4 w-4" />
            </button>
            {currentTier !== "free" && !subscription.cancel_at_period_end && (
              <button
                onClick={handleCancelAtPeriodEnd}
                disabled={cancelling}
                className="rounded-lg border border-rose-300 px-3 py-2 text-sm font-medium text-rose-700 hover:bg-rose-50 disabled:opacity-60 dark:border-rose-700 dark:text-rose-300 dark:hover:bg-rose-950/40"
              >
                {cancelling ? "Cancelling..." : "Cancel at period end"}
              </button>
            )}
          </div>
        </div>
      </section>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
        <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
          Usage
        </h4>
        <div className="mt-4 space-y-4">
          {[
            {
              label: "Customers",
              used: subscription.usage.customers.used,
              limit: subscription.usage.customers.limit,
            },
            {
              label: "Integrations",
              used: subscription.usage.integrations.used,
              limit: subscription.usage.integrations.limit,
            },
          ].map((item) => (
            <div key={item.label}>
              <div className="mb-1 flex items-center justify-between text-sm text-gray-600 dark:text-gray-300">
                <span>{item.label}</span>
                <span>
                  {item.used} / {formatLimit(item.limit)}
                </span>
              </div>
              <div className="h-2 rounded-full bg-gray-200 dark:bg-gray-700">
                <div
                  className="h-2 rounded-full bg-indigo-500"
                  style={{ width: `${usagePercent(item.used, item.limit)}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
        <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
          Change plan
        </h4>
        <div className="mt-3 grid gap-3 md:grid-cols-2">
          {recommendedPlans.map((plan) => {
            const isCurrent = plan.tier === currentTier;
            return (
              <div
                key={plan.tier}
                className={`rounded-lg border p-4 ${
                  isCurrent
                    ? "border-emerald-300 bg-emerald-50 dark:border-emerald-700 dark:bg-emerald-950/20"
                    : "border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800/50"
                }`}
              >
                <div className="flex items-start justify-between">
                  <div>
                    <p className="font-semibold text-gray-900 dark:text-gray-100">
                      {plan.name}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {plan.description}
                    </p>
                  </div>
                  <p className="text-sm font-medium text-gray-700 dark:text-gray-200">
                    $
                    {cycle === "monthly" ? plan.monthlyPrice : plan.annualPrice}
                    /{cycle === "monthly" ? "mo" : "yr"}
                  </p>
                </div>
                <button
                  disabled={checkoutLoading || isCurrent}
                  onClick={() => startCheckout({ tier: plan.tier, cycle })}
                  className="mt-3 w-full rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {isCurrent
                    ? "Current plan"
                    : checkoutLoading
                      ? "Redirecting..."
                      : `Switch to ${plan.name}`}
                </button>
              </div>
            );
          })}
        </div>
      </section>
    </div>
  );
}
