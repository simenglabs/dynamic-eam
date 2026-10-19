"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { ApiError, api } from "@eam/ui";

export type FormState = { error?: string };

export async function saveUser(id: string | undefined, _: FormState, form: FormData): Promise<FormState> {
  const body = {
    username: String(form.get("username") ?? ""),
    displayName: String(form.get("displayName") ?? ""),
    email: String(form.get("email") ?? ""),
    roles: form.getAll("roles").map(String),
    password: String(form.get("password") ?? ""),
    isActive: form.get("isActive") === "on",
    ...(id && { isLocked: form.get("isLocked") === "on" }),
  };
  try {
    await api("administration", id ? `/users/${encodeURIComponent(id)}` : "/users", { method: id ? "PUT" : "POST", body });
  } catch (e) {
    if (e instanceof ApiError && e.status < 500) return { error: e.message };
    throw e;
  }
  revalidatePath("/user-management");
  redirect("/user-management");
}

export async function deleteUser(id: string): Promise<FormState> {
  try {
    await api("administration", `/users/${encodeURIComponent(id)}`, { method: "DELETE" });
  } catch (e) {
    if (e instanceof ApiError && e.status < 500) return { error: e.message };
    throw e;
  }
  revalidatePath("/user-management");
  redirect("/user-management");
}
