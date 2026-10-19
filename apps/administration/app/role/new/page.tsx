import { Alert, PageHeader, can, getMe } from "@eam/ui";
import { RoleForm } from "../RoleForm";
import { loadTree } from "../tree";

export default async function NewRole() {
  const crumbs = ["Administration", "Role", "New"];
  if (!can(await getMe(), "administration/role", "create"))
    return (<><PageHeader crumbs={crumbs} title="Add New Role" /><div className="px-8 py-4"><Alert>You don't have access.</Alert></div></>);
  return (
    <>
      <PageHeader crumbs={crumbs} title="Add New Role" />
      <RoleForm tree={await loadTree()} canSave canDelete={false} />
    </>
  );
}
