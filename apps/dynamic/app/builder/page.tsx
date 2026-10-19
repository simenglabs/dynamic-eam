import Link from "next/link";
import { Plus } from "lucide-react";
import { Badge, Button, Card, MENU, PageHeader, Table, Td, Th, Tr, api, getMe } from "@eam/ui";
import { Forbidden } from "../_lib/Forbidden";
import type { ContentType } from "../_lib/types";

const zoneLabel = (z: string) => MENU.find((m) => m.zone === z)?.label ?? z;

export default async function Builder() {
  const crumbs = ["Dynamic", "Content-Type Builder"];
  if (!(await getMe())?.access.superAdmin) return <Forbidden crumbs={crumbs} />;
  const cts = await api<ContentType[]>("dynamic", "/content-types");
  return (
    <>
      <PageHeader crumbs={crumbs} title="Content-Type Builder" />
      <div className="px-8 py-4">
        <Card>
          <div className="flex items-center justify-between border-b pb-4">
            <div>
              <h2 className="text-lg font-bold">Content Types</h2>
              <p className="text-muted-foreground text-sm">Access rights for each menu are managed in Administration › Role.</p>
            </div>
            <Link href="/builder/new"><Button tabIndex={-1}><Plus /> New Content Type</Button></Link>
          </div>
          <Table>
            <thead>
              <tr>{["Name", "UID", "Menu", "Fields", ""].map((h) => <Th key={h}>{h}</Th>)}</tr>
            </thead>
            <tbody>
              {cts.map((ct) => (
                <Tr key={ct.uid}>
                  <Td>
                    <Link href={`/builder/${ct.uid}`} className="text-primary underline underline-offset-3">{ct.label}</Link>
                    {ct.hidden && <Badge className="ml-2">hidden</Badge>}
                  </Td>
                  <Td>{ct.uid}</Td>
                  <Td>{zoneLabel(ct.parent)}</Td>
                  <Td>{ct.fields.length}</Td>
                  <Td><Link href={`/c/${ct.uid}`} className="text-primary">View records →</Link></Td>
                </Tr>
              ))}
              {cts.length === 0 && <tr><td colSpan={5} className="text-muted-foreground p-6 text-center">No content types yet.</td></tr>}
            </tbody>
          </Table>
        </Card>
      </div>
    </>
  );
}
