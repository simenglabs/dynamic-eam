"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { ApiError, api } from "@eam/ui";

export type ActionState = { error?: string };

async function call(path: string, method: string, body?: unknown): Promise<ActionState | null> {
  try {
    await api("dynamic", path, { method, body });
    return null;
  } catch (e) {
    if (e instanceof ApiError && e.status < 500) return { error: e.message };
    if (e instanceof ApiError && e.status === 502) return { error: e.message }; // tersimpan, index gagal
    throw e;
  }
}

const base = (kind: "content-types" | "components") => (kind === "content-types" ? "/builder" : "/components");

// json = schema hasil SchemaForm. uid kosong = buat baru.
export async function saveSchema(kind: "content-types" | "components", uid: string | undefined, json: string): Promise<ActionState> {
  const err = await call(uid ? `/${kind}/${encodeURIComponent(uid)}` : `/${kind}`, uid ? "PUT" : "POST", JSON.parse(json));
  if (err) return err;
  revalidatePath("/", "layout"); // sidebar ikut berubah
  redirect(base(kind));
}

export async function deleteSchema(kind: "content-types" | "components", uid: string): Promise<ActionState> {
  const err = await call(`/${kind}/${encodeURIComponent(uid)}`, "DELETE");
  if (err) return err;
  revalidatePath("/", "layout");
  redirect(base(kind));
}

export async function saveEntry(uid: string, id: string | undefined, json: string): Promise<ActionState> {
  const path = `/content-types/${encodeURIComponent(uid)}/entries${id ? `/${encodeURIComponent(id)}` : ""}`;
  const err = await call(path, id ? "PUT" : "POST", JSON.parse(json));
  if (err) return err;
  redirect(`/c/${uid}`);
}

export async function deleteEntry(uid: string, id: string): Promise<ActionState> {
  const err = await call(`/content-types/${encodeURIComponent(uid)}/entries/${encodeURIComponent(id)}`, "DELETE");
  if (err) return err;
  redirect(`/c/${uid}`);
}

// Profil user: upsert (dibuat bila belum ada), lalu muat ulang halaman profil.
export async function saveProfile(userId: string, json: string): Promise<ActionState> {
  const err = await call(`/profiles/${encodeURIComponent(userId)}`, "PUT", JSON.parse(json));
  if (err) return err;
  redirect(`/profile/${encodeURIComponent(userId)}?saved=1`);
}

export async function runTransition(uid: string, id: string, key: string, note: string, view?: string, data?: Record<string, unknown>): Promise<ActionState> {
  const err = await call(`/content-types/${encodeURIComponent(uid)}/entries/${encodeURIComponent(id)}/transitions/${encodeURIComponent(key)}`, "POST", { note, data });
  if (err) return err;
  redirect(view ? `/c/${uid}?view=${encodeURIComponent(view)}` : `/c/${uid}/${id}`);
}
