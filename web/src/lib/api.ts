import axios from "axios";

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1",
  headers: {
    "Content-Type": "application/json",
  },
});

export interface AuthUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
}

export interface AuthOrg {
  id: string;
  name: string;
  slug: string;
  role: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface AuthResponse {
  user: AuthUser;
  organization: AuthOrg;
  tokens: TokenPair;
}

export interface RegisterPayload {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  org_name: string;
}

export interface LoginPayload {
  email: string;
  password: string;
}

export const authApi = {
  register: (data: RegisterPayload) =>
    api.post<AuthResponse>("/auth/register", data),

  login: (data: LoginPayload) => api.post<AuthResponse>("/auth/login", data),

  refresh: (refreshToken: string) =>
    api.post<AuthResponse>("/auth/refresh", { refresh_token: refreshToken }),

  requestPasswordReset: (email: string) =>
    api.post("/auth/password-reset/request", { email }),

  completePasswordReset: (token: string, newPassword: string) =>
    api.post("/auth/password-reset/complete", {
      token,
      new_password: newPassword,
    }),
};

export default api;
