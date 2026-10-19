import { notFound, redirect } from "next/navigation";
import { Alert, Card, PageHeader } from "./ui";
import { findZone, links, permKey } from "./menu";
import { can, getMe } from "./api";

// Placeholder for submenus that don't have their own page yet.
// Create app/<path>/page.tsx in the zone to override (static routes win over the catch-all).
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
        <div className="px-8 py-4"><Alert>You don't have access to this page.</Alert></div>
      </>
    );
  return (
    <>
      <PageHeader crumbs={[z.label, item.label]} title={item.label} />
      <div className="px-8 py-4">
        <Card>
          <p className="text-muted-foreground">The {item.label} page hasn't been built yet.</p>
        </Card>
      </div>
    </>
  );
}
