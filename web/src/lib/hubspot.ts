import api from "./api";

export interface HubSpotStatus {
  status: string;
  external_account_id?: string;
  last_sync_at?: string;
  last_sync_error?: string;
  connected_at?: string;
  contact_count?: number;
  deal_count?: number;
  company_count?: number;
}

export const hubspotApi = {
  getConnectUrl: () =>
    api.get<{ url: string }>("/integrations/hubspot/connect"),

  getStatus: () => api.get<HubSpotStatus>("/integrations/hubspot/status"),

  disconnect: () => api.delete("/integrations/hubspot"),

  triggerSync: () =>
    api.post<{ message: string }>("/integrations/hubspot/sync"),

  callback: (code: string, state: string) =>
    api.get("/integrations/hubspot/callback", { params: { code, state } }),
};
