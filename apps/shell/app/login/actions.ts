"use server";

import { cookies } from "next/headers";
import { ApiError, TOKEN_COOKIE, api } from "@eam/ui";

export type LoginState = { error?: string; attemptsRemaining?: number; username?: string; redirectTo?: string };

export async function login(_: LoginState, form: FormData): Promise<LoginState> {
  const username = String(form.get("username") ?? "").trim();
  const password = String(form.get("password") ?? "");
  const next = String(form.get("next") ?? "");
  if (!username || !password) return { error: "Username dan password wajib diisi", username };

  let res: { token: string; expiresIn: number };
  try {
    res = await api("administration", "/auth/login", { method: "POST", body: { username, password } });
  } catch (e) {
    if (e instanceof ApiError && e.status < 500)
      return { error: e.message, attemptsRemaining: e.data?.attemptsRemaining as number | undefined, username };
    return { error: "Layanan login tidak tersedia, coba lagi nanti", username };
  }

  (await cookies()).set(TOKEN_COOKIE, res.token, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: res.expiresIn,
  });
  // Hanya path internal (cegah open redirect ke domain lain). Navigasi dilakukan client dengan hard reload,
  // karena tujuan bisa berada di zone lain (redirect() server action = soft nav, tidak bisa lintas zone).
  return { redirectTo: next.startsWith("/") && !next.startsWith("//") && !next.startsWith("/\\") ? next : "/dashboard" };
}
