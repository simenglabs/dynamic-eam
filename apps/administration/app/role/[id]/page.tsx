import { notFound } from "next/navigation";
import { Alert, ApiError, PageHeader, api, can, getMe } from "@eam/ui";
import { RoleForm } from "../RoleForm";
import { loadTree } from "../tree";
import type { Role } from "../types";

export default async function EditRole({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const me = await getMe();
  let role: Role;
  try {
    role = await api<Role>("administration", `/roles/${encodeURIComponent(id)}`);
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) notFound();
    if (e instanceof ApiError && e.status === 403)
      return (<><PageHeader crumbs={["Administration", "Role"]} title="Role" /><div className="px-8 py-4"><Alert>Anda tidak punya akses.</Alert></div></>);
    throw e;
  }
  return (
    <>
      <PageHeader crumbs={["Administration", "Role", role.name]} title={role.name} />
      <RoleForm role={role} tree={await loadTree()} canSave={can(me, "administration/role", "update")} canDelete={can(me, "administration/role", "delete")} />
    </>
  );
}
