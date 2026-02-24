import { useCallback, useState } from "react";
import { AxiosError } from "axios";

import { billingApi, type CheckoutPayload } from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";

export function useCheckout() {
  const [loading, setLoading] = useState(false);
  const toast = useToast();

  const startCheckout = useCallback(
    async (payload: CheckoutPayload) => {
      setLoading(true);
      try {
        const { data } = await billingApi.createCheckout(payload);
        if (!data.url) {
          throw new Error("missing checkout url");
        }
        window.location.href = data.url;
      } catch (error) {
        if (error instanceof AxiosError && error.response?.data?.error) {
          toast.error(String(error.response.data.error));
        } else {
          toast.error("Unable to start checkout right now.");
        }
      } finally {
        setLoading(false);
      }
    },
    [toast],
  );

  return {
    loading,
    startCheckout,
  };
}
