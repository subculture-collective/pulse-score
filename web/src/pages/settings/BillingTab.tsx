import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";

import SubscriptionManager from "@/components/billing/SubscriptionManager";

export default function BillingTab() {
  const [searchParams] = useSearchParams();

  const checkoutState = useMemo(() => {
    const value = searchParams.get("checkout");
    if (value === "success" || value === "cancelled") {
      return value;
    }
    return null;
  }, [searchParams]);

  return <SubscriptionManager checkoutState={checkoutState} />;
}
