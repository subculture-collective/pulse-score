import api from "./api";

export interface IntercomStatus {
  status: string;
  external_account_id?: string;
  last_sync_at?: string;
  last_sync_error?: string;
  connected_at?: string;
  conversation_count?: number;
  contact_count?: number;
}

export const intercomApi = {
  getConnectUrl: () =>
    api.get<{ url: string }>("/integrations/intercom/connect"),

  getStatus: () => api.get<IntercomStatus>("/integrations/intercom/status"),

  disconnect: () => api.delete("/integrations/intercom"),

  triggerSync: () =>
    api.post<{ message: string }>("/integrations/intercom/sync"),

  callback: (code: string, state: string) =>
    api.get("/integrations/intercom/callback", { params: { code, state } }),
};
