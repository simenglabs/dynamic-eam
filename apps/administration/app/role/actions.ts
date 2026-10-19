"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { ApiError, api } from "@eam/ui";

export type FormState = { error?: string };

async function call(path: string, method: string, body?: unknown): Promise<FormState | null> {
  try {
    await api("administration", path, { method, body });
    return null;
  } catch (e) {
    if (e instanceof ApiError && e.status < 500) return { error: e.message };
    throw e;
  }
}

export async function saveRole(id: string | undefined, json: string): Promise<FormState> {
  const err = await call(id ? `/roles/${encodeURIComponent(id)}` : "/roles", id ? "PUT" : "POST", JSON.parse(json));
  if (err) return err;
  revalidatePath("/", "layout");
  redirect("/role");
}

export async function deleteRole(id: string): Promise<FormState> {
  const err = await call(`/roles/${encodeURIComponent(id)}`, "DELETE");
  if (err) return err;
  revalidatePath("/", "layout");
  redirect("/role");
}
