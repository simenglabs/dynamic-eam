import Link from "next/link";
import { Plus, Search } from "lucide-react";
import { ApiError, Alert, Badge, Button, Card, Input, PageHeader, Table, Td, Th, Tr, api, can, getMe } from "@eam/ui";
import type { User } from "./types";

export default async function UserManagement({ searchParams }: { searchParams: Promise<{ q?: string }> }) {
  const [{ q = "" }, me] = await Promise.all([searchParams, getMe()]);
  let users: User[];
  try {
    users = await api<User[]>("administration", "/users");
  } catch (e) {
    if (e instanceof ApiError && e.status === 403) return <Forbidden />;
    throw e;
  }
  const needle = q.toLowerCase();
  const rows = users.filter((u) => [u.username, u.displayName, u.email].some((v) => v.toLowerCase().includes(needle)));

  return (
    <>
      <PageHeader crumbs={["Administration", "Users Management"]} title="Users Management" />
      <div className="px-8 py-4">
        <Card>
          <div className="flex items-center justify-between border-b pb-4">
            <h2 className="text-lg font-bold">Users</h2>
            {can(me, "administration/user-management", "create") && (
              <Link href="/user-management/new">
                <Button tabIndex={-1}><Plus /> Add New User</Button>
              </Link>
            )}
          </div>
          <form className="relative w-fit">
            <Search className="text-muted-foreground absolute top-3 left-2.5 size-4" />
            <Input type="search" name="q" defaultValue={q} placeholder="Search..." aria-label="Search user" className="w-60 pl-8" />
          </form>
          <Table>
            <thead>
              <tr>
                {["Username", "Name", "Email", "Roles", "Status", "Updated"].map((h) => <Th key={h}>{h}</Th>)}
              </tr>
            </thead>
            <tbody>
              {rows.map((u) => (
                <Tr key={u.id}>
                  <Td>
                    <Link href={`/user-management/${u.id}`} className="text-primary underline underline-offset-3">{u.username}</Link>
                  </Td>
                  <Td>{u.displayName}</Td>
                  <Td>{u.email}</Td>
                  <Td>{u.roles.map((r) => <Badge key={r} color={r === "superadmin" ? "yellow" : "blue"} className="mr-1">{r}</Badge>)}</Td>
                  <Td>
                    {u.isLocked ? <Badge color="red">Locked</Badge> : u.isActive ? <Badge color="green">Active</Badge> : <Badge>Inactive</Badge>}
                  </Td>
                  <Td>{new Date(u.updatedAt).toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" })}</Td>
                </Tr>
              ))}
              {rows.length === 0 && (
                <tr><td colSpan={6} className="text-muted-foreground p-6 text-center">Tidak ada user.</td></tr>
              )}
            </tbody>
          </Table>
          <p className="text-sm">Showing {rows.length} of {users.length}.</p>
        </Card>
      </div>
    </>
  );
}

function Forbidden() {
  return (
    <>
      <PageHeader crumbs={["Administration", "Users Management"]} title="Users Management" />
      <div className="px-8 py-4"><Alert>Anda tidak punya akses ke halaman ini (khusus admin).</Alert></div>
    </>
  );
}
