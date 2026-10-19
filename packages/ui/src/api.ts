import { cookies } from "next/headers";
import { cache } from "react";
import { findZone } from "./menu";

export const TOKEN_COOKIE = "eam_token";

export class ApiError extends Error {
  constructor(public status: number, message: string, public data?: Record<string, unknown>) {
    super(message);
  }
}

// Server only: panggil service Go zone langsung, token user diteruskan sebagai Bearer.
// Env <ZONE>_API_URL (mis. WORK_ORDER_API_URL) untuk prod.
export async function api<T>(zone: string, path: string, init: { method?: string; body?: unknown } = {}): Promise<T> {
  const base =
    process.env[`${zone.toUpperCase().replace(/-/g, "_")}_API_URL`] ??
    `http://localhost:${findZone(zone).apiPort}`;
  const token = (await cookies()).get(TOKEN_COOKIE)?.value;
  const res = await fetch(`${base}/api/${zone}${path}`, {
    method: init.method ?? "GET",
    headers: { "Content-Type": "application/json", ...(token && { Authorization: `Bearer ${token}` }) },
    body: init.body === undefined ? undefined : JSON.stringify(init.body),
    cache: "no-store",
  });
  const data = res.status === 204 ? null : await res.json().catch(() => null);
  if (!res.ok) throw new ApiError(res.status, data?.error ?? `HTTP ${res.status}`, data);
  return data as T;
}

export type Access = { userId: string; roles: string[]; superAdmin: boolean; permissions: Record<string, string[]> };
export type Me = { id: string; username: string; name: string; access: Access };

// Profil + izin efektif user (RBAC), sekali per request. null = belum login / sesi tidak valid.
export const getMe = cache(async (): Promise<Me | null> => {
  if (!(await cookies()).get(TOKEN_COOKIE)) return null;
  return api<Me>("administration", "/auth/me").catch(() => null);
});

// Untuk menyembunyikan menu/tombol. Penegakan sebenarnya tetap di service Go.
export function can(me: Me | null, key: string, action: "read" | "create" | "update" | "delete" | "download"): boolean {
  if (!me) return false;
  const p = me.access.permissions;
  return me.access.superAdmin || !!p[key]?.includes(action) || !!p["*"]?.includes(action);
}
