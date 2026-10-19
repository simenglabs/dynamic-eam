import { Suspense } from "react";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { DialogHost } from "./Dialog";
import { api, getMe } from "./api";
import { buildMenu, type DynamicMenu } from "./menu";

export function RootHtml({ children }: { children: React.ReactNode }) {
  return (
    <html lang="id">
      <body>
        {children}
        <DialogHost />
      </body>
    </html>
  );
}

// Sidebar + area konten. zone = zone app yang sedang render (undefined = shell).
export async function Chrome({ zone, children }: { zone?: string; children: React.ReactNode }) {
  const me = await getMe();
  // Service dynamic mati tidak boleh merusak halaman lain: menu dinamis cukup tidak tampil.
  const dynamic = me ? await api<DynamicMenu[]>("dynamic", "/content-types").catch(() => []) : [];
  const menu = buildMenu(dynamic, me?.access ?? null);
  return (
    <>
      {/* Suspense: Sidebar membaca ?view= (useSearchParams) */}
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

// Root layout untuk zone.
export function AppLayout({ zone, children }: { zone: string; children: React.ReactNode }) {
  return (
    <RootHtml>
      <Chrome zone={zone}>{children}</Chrome>
    </RootHtml>
  );
}
