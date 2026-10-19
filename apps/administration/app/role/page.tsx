import Link from "next/link";
import { Plus } from "lucide-react";
import { Alert, ApiError, Badge, Button, Card, PageHeader, Table, Td, Th, Tr, api, can, getMe } from "@eam/ui";
import type { Role } from "./types";

const PERM = "administration/role";

export default async function Roles() {
  const crumbs = ["Administration", "Role"];
  const me = await getMe();
  let roles: Role[];
  try {
    roles = await api<Role[]>("administration", "/roles");
  } catch (e) {
    if (e instanceof ApiError && e.status === 403)
      return (<><PageHeader crumbs={crumbs} title="Role" /><div className="px-8 py-4"><Alert>Anda tidak punya akses ke halaman ini.</Alert></div></>);
    throw e;
  }
  return (
    <>
      <PageHeader crumbs={crumbs} title="Role" />
      <div className="px-8 py-4">
        <Card>
          <div className="flex items-center justify-between border-b pb-4">
            <h2 className="text-lg font-bold">Roles</h2>
            {can(me, PERM, "create") && <Link href="/role/new"><Button tabIndex={-1}><Plus /> Add New Role</Button></Link>}
          </div>
          <Table>
            <thead><tr>{["Nama", "ID", "Deskripsi", "Menu"].map((h) => <Th key={h}>{h}</Th>)}</tr></thead>
            <tbody>
              {roles.map((r) => (
                <Tr key={r.id}>
                  <Td>
                    <Link href={`/role/${r.id}`} className="text-primary underline underline-offset-3">{r.name}</Link>
                    {r.system && <Badge color="yellow" className="ml-2">system</Badge>}
                  </Td>
                  <Td>{r.id}</Td>
                  <Td>{r.description}</Td>
                  <Td>{r.system ? "Semua" : r.permissions["*"] ? `Semua menu (${r.permissions["*"].join(", ")})` : `${Object.keys(r.permissions).length} menu`}</Td>
                </Tr>
              ))}
            </tbody>
          </Table>
        </Card>
      </div>
    </>
  );
}
