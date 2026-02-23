import api from "./api";

export interface StripeStatus {
  status: string;
  account_id?: string;
  last_sync_at?: string;
  last_sync_error?: string;
  customer_count?: number;
}

export const stripeApi = {
  getConnectUrl: () =>
    api.get<{ url: string }>("/integrations/stripe/connect"),

  getStatus: () => api.get<StripeStatus>("/integrations/stripe/status"),

  disconnect: () => api.delete("/integrations/stripe"),

  triggerSync: () =>
    api.post<{ message: string }>("/integrations/stripe/sync"),

  callback: (code: string, state: string) =>
    api.get("/integrations/stripe/callback", { params: { code, state } }),
};
