"use client";

import Link from "next/link";
import { useActionState, useState, useTransition } from "react";
import { ArrowLeft, MapPin, Save, Trash2 } from "lucide-react";
import { Alert, Button, Card, Field, Input } from "@eam/ui/ui";
import { confirmDialog } from "@eam/ui/dialog";
import { Checkbox } from "@eam/ui/checkbox";
import { deleteUser, saveUser, type FormState } from "./actions";
import type { User } from "./types";

export function UserForm({ user, roles, canSuper, canSave, canDelete }: {
  user?: User;
  roles: { id: string; name: string }[];
  canSuper: boolean;
  canSave: boolean;
  canDelete: boolean;
}) {
  const [state, action, pending] = useActionState<FormState, FormData>(saveUser.bind(null, user?.id), {});
  const [deleting, startDelete] = useTransition();
  const [delError, setDelError] = useState<string>();

  const onDelete = async () => {
    if (!user || !(await confirmDialog({ title: `Delete user ${user.username}?`, message: "The account will no longer be able to sign in and cannot be restored.", confirmText: "Delete", danger: true }))) return;
    startDelete(async () => setDelError((await deleteUser(user.id)).error));
  };

  return (
    <form action={action}>
      <div className="sticky top-16 z-10 mx-8 mt-4 flex items-center justify-between rounded-2xl border border-slate-200/80 bg-white/90 px-5 py-3 backdrop-blur">
        <Link href="/user-management" className="flex items-center gap-2"><ArrowLeft className="size-5" /> Back</Link>
        <div className="flex gap-2">
          {user && canDelete && (
            <Button type="button" variant="ghost" onClick={onDelete} disabled={deleting} className="text-destructive">
              <Trash2 /> Delete
            </Button>
          )}
          {user && (
            // Profile (site, organization, etc.) = dynamic content type user-profile; rendered by the dynamic zone.
            <a href={`/dynamic/profile/${encodeURIComponent(user.id)}`}>
              <Button type="button" variant="outline" tabIndex={-1}><MapPin /> Profile & placement</Button>
            </a>
          )}
          {canSave && <Button disabled={pending}><Save /> {pending ? "Saving..." : "Save"}</Button>}
        </div>
      </div>
      <div className="px-8 py-4">
        <Card>
          <h2 className="border-b pb-4 text-lg font-bold">User</h2>
          {(state.error || delError) && <Alert>{state.error ?? delError}</Alert>}
          <div className="grid gap-x-4 gap-y-5 md:grid-cols-2">
            <Field label="Username" required={!user}>
              {user ? (
                <p className="border-b px-3 py-2 font-normal">{user.username}</p>
              ) : (
                <Input name="username" required autoComplete="off" />
              )}
            </Field>
            <Field label="Display Name" required>
              <Input name="displayName" required defaultValue={user?.displayName} />
            </Field>
            <Field label="Email" required>
              <Input name="email" type="email" required defaultValue={user?.email} />
            </Field>
            <Field label="Role" required>
              <span className="flex flex-wrap gap-x-5 gap-y-2 pt-2 font-normal">
                {roles
                  .filter((r) => r.id !== "superadmin" || canSuper || user?.roles.includes("superadmin"))
                  .map((r) => (
                    <Checkbox key={r.id} name="roles" value={r.id} label={r.name}
                      defaultChecked={user ? user.roles.includes(r.id) : r.id === "user"} />
                  ))}
              </span>
            </Field>
            <Field label={user ? "Reset Password" : "Password"} required={!user} optional={!!user}>
              <Input name="password" type="password" minLength={8} maxLength={72} required={!user} autoComplete="new-password" placeholder={user ? "Leave blank to keep unchanged" : "At least 8 characters"} />
            </Field>
            <div className="flex items-end gap-8 pb-2">
              <label className="flex items-center gap-2 font-bold">
                <Checkbox name="isActive" defaultChecked={user?.isActive ?? true} label="Active" />
              </label>
              {user && (
                <label className="flex items-center gap-2 font-bold">
                  <Checkbox name="isLocked" defaultChecked={user.isLocked} label="Locked" />
                  {user.failedAttempts > 0 && <span className="text-muted-foreground font-normal">({user.failedAttempts} failed logins)</span>}
                </label>
              )}
            </div>
          </div>
        </Card>
      </div>
    </form>
  );
}
