import { notFound, redirect } from "next/navigation";
import { Alert, Card, PageHeader } from "./ui";
import { findZone, links, permKey } from "./menu";
import { can, getMe } from "./api";

// Placeholder untuk sub menu yang belum punya halaman sendiri.
// Buat app/<path>/page.tsx di zone untuk override (route statis menang atas catch-all).
export async function ZonePage({ zone, params }: { zone: string; params: Promise<{ slug?: string[] }> }) {
  const z = findZone(zone);
  const { slug } = await params;
  if (!slug) redirect(links(z)[0].path);
  const item = links(z).find((l) => l.path === "/" + slug.join("/"));
  if (!item) notFound();
  if (!can(await getMe(), permKey(zone, item), "read"))
    return (
      <>
        <PageHeader crumbs={[z.label, item.label]} title={item.label} />
        <div className="px-8 py-4"><Alert>Anda tidak punya akses ke halaman ini.</Alert></div>
      </>
    );
  return (
    <>
      <PageHeader crumbs={[z.label, item.label]} title={item.label} />
      <div className="px-8 py-4">
        <Card>
          <p className="text-muted-foreground">Halaman {item.label} belum dibuat.</p>
        </Card>
      </div>
    </>
  );
}
