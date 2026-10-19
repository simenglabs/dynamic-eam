"use server";

import { cookies } from "next/headers";
import { ApiError, TOKEN_COOKIE, api } from "@eam/ui";

export type LoginState = { error?: string; attemptsRemaining?: number; username?: string; redirectTo?: string };

export async function login(_: LoginState, form: FormData): Promise<LoginState> {
  const username = String(form.get("username") ?? "").trim();
  const password = String(form.get("password") ?? "");
  const next = String(form.get("next") ?? "");
  if (!username || !password) return { error: "Username and password are required", username };

  let res: { token: string; expiresIn: number };
  try {
    res = await api("administration", "/auth/login", { method: "POST", body: { username, password } });
  } catch (e) {
    if (e instanceof ApiError && e.status < 500)
      return { error: e.message, attemptsRemaining: e.data?.attemptsRemaining as number | undefined, username };
    return { error: "Login service is unavailable, please try again later", username };
  }

  (await cookies()).set(TOKEN_COOKIE, res.token, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: res.expiresIn,
  });
  // Internal paths only (prevents open redirects to other domains). The client navigates with a hard reload,
  // since the target may be in another zone (server action redirect() = soft nav, can't cross zones).
  return { redirectTo: next.startsWith("/") && !next.startsWith("//") && !next.startsWith("/\\") ? next : "/dashboard" };
}
