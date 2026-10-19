import { notFound } from "next/navigation";
import { ApiError, PageHeader, api, can, getMe } from "@eam/ui";
import { UserForm } from "../UserForm";
import type { User } from "../types";

export default async function EditUser({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const me = await getMe();
  const roles = await api<{ id: string; name: string }[]>("administration", "/roles").catch(() => []);
  const user = await api<User>("administration", `/users/${encodeURIComponent(id)}`).catch((e) => {
    if (e instanceof ApiError && e.status === 404) notFound();
    throw e;
  });
  return (
    <>
      <PageHeader crumbs={["Administration", "Users Management", user.username]} title={user.displayName} />
      <UserForm
        user={user}
        roles={roles}
        canSuper={!!me?.access.superAdmin}
        canSave={can(me, "administration/user-management", "update")}
        canDelete={can(me, "administration/user-management", "delete")}
      />
    </>
  );
}
