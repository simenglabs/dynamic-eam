import Link from "next/link";
import { Plus } from "lucide-react";
import { Button, Card, PageHeader, Table, Td, Th, Tr, api, getMe } from "@eam/ui";
import { Forbidden } from "../_lib/Forbidden";
import type { Component } from "../_lib/types";

export default async function Components() {
  const crumbs = ["Dynamic", "Components"];
  if (!(await getMe())?.access.superAdmin) return <Forbidden crumbs={crumbs} />;
  const comps = await api<Component[]>("dynamic", "/components");
  return (
    <>
      <PageHeader crumbs={crumbs} title="Components" />
      <div className="px-8 py-4">
        <Card>
          <div className="flex items-center justify-between border-b pb-4">
            <div>
              <h2 className="text-lg font-bold">Components</h2>
              <p className="text-muted-foreground text-sm">Kumpulan field yang bisa dipakai ulang di content type (boleh repeatable).</p>
            </div>
            <Link href="/components/new"><Button tabIndex={-1}><Plus /> New Component</Button></Link>
          </div>
          <Table>
            <thead><tr>{["Nama", "UID", "Fields"].map((h) => <Th key={h}>{h}</Th>)}</tr></thead>
            <tbody>
              {comps.map((c) => (
                <Tr key={c.uid}>
                  <Td><Link href={`/components/${c.uid}`} className="text-primary underline underline-offset-3">{c.label}</Link></Td>
                  <Td>{c.uid}</Td>
                  <Td>{c.fields.map((f) => f.label).join(", ")}</Td>
                </Tr>
              ))}
              {comps.length === 0 && <tr><td colSpan={3} className="text-muted-foreground p-6 text-center">Belum ada component.</td></tr>}
            </tbody>
          </Table>
        </Card>
      </div>
    </>
  );
}
