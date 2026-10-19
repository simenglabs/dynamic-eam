import { Alert, PageHeader } from "@eam/ui";

export function Forbidden({ crumbs }: { crumbs: string[] }) {
  return (
    <>
      <PageHeader crumbs={crumbs} title={crumbs.at(-1)!} />
      <div className="px-8 py-4"><Alert>You do not have access to this page (superadmin only).</Alert></div>
    </>
  );
}
