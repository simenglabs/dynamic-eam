import { Alert, PageHeader, api, can, getMe } from "@eam/ui";
import { UserForm } from "../UserForm";

export default async function NewUser() {
  const me = await getMe();
  const crumbs = ["Administration", "Users Management", "New"];
  if (!can(me, "administration/user-management", "create"))
    return (<><PageHeader crumbs={crumbs} title="Add New User" /><div className="px-8 py-4"><Alert>You don't have access.</Alert></div></>);
  const roles = await api<{ id: string; name: string }[]>("administration", "/roles");
  return (
    <>
      <PageHeader crumbs={crumbs} title="Add New User" />
      <UserForm roles={roles} canSuper={!!me?.access.superAdmin} canSave canDelete={false} />
    </>
  );
}
