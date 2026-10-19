import { Suspense } from "react";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { DialogHost } from "./Dialog";
import { api, getMe } from "./api";
import { buildMenu, type DynamicMenu } from "./menu";

export function RootHtml({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        {children}
        <DialogHost />
      </body>
    </html>
  );
}

// Sidebar + content area. zone = the zone app currently rendering (undefined = shell).
export async function Chrome({ zone, children }: { zone?: string; children: React.ReactNode }) {
  const me = await getMe();
  // A down dynamic service must not break other pages: dynamic menus are simply hidden.
  const dynamic = me ? await api<DynamicMenu[]>("dynamic", "/content-types").catch(() => []) : [];
  const menu = buildMenu(dynamic, me?.access ?? null);
  return (
    <>
      {/* Suspense: Sidebar reads ?view= (useSearchParams) */}
      <Suspense>
        <Sidebar zone={zone} menu={menu} />
      </Suspense>
      <div className="min-h-screen pl-[var(--sidebar-width)]">
        <Topbar menu={menu} user={me && { name: me.name, username: me.username, roles: me.access.roles }} />
        <main className="pb-10">{children}</main>
      </div>
    </>
  );
}

// Root layout for a zone.
export function AppLayout({ zone, children }: { zone: string; children: React.ReactNode }) {
  return (
    <RootHtml>
      <Chrome zone={zone}>{children}</Chrome>
    </RootHtml>
  );
}
