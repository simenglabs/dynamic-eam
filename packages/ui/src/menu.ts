// One top-level menu = one zone (separate Next.js app, basePath /<zone>) + one Go service.
// Submenu = route inside the zone. Ports here are the single source for shell rewrites & api().
// zone = zone that owns the page if different from its parent menu (e.g. a dynamic menu under "Master"
// whose page is rendered by the "dynamic" zone). perm = RBAC key (default "<zone><path>", e.g. "assets/tnd").
// superOnly = superadmin only (not managed via roles).
export type Link = { label: string; path: string; zone?: string; perm?: string; superOnly?: boolean };
export type Item = Link | { label: string; items: Link[] };
// No webPort/apiPort = zone without its own app & service: just a container for dynamic menus (rendered by the "dynamic" zone).
export type ZoneDef = { zone: string; label: string; webPort?: number; apiPort?: number; items: Item[] };

export const MENU: ZoneDef[] = [
  {
    zone: "renkon", label: "Renkon", webPort: 3001, apiPort: 8081,
    items: [
      { label: "PRK", items: [
        { label: "PRK Parent Unit", path: "/prk" },
        { label: "PRK Unit", path: "/prk/unit" },
        { label: "Approval PRK", path: "/prk/approval" },
      ] },
      { label: "SKAI", items: [
        { label: "SKAI", path: "/skai" },
        { label: "Approval SKAI", path: "/skai/approval" },
      ] },
      { label: "SKKI", items: [{ label: "SKKI", path: "/skki" }] },
      { label: "KHS", items: [
        { label: "KHS", path: "/khs" },
        { label: "Approval KHS", path: "/khs/approval" },
        { label: "KHS Template", path: "/khs/template" },
        { label: "KHS Unit", path: "/khs/master-unit" },
      ] },
      { label: "Material List", path: "/material" },
      { label: "Request List", path: "/permohonan" },
      { label: "Planning", path: "/perencanaan" },
      { label: "Planning Approval", path: "/perencanaan/approval" },
    ],
  },
  {
    // All Assets submenus come from dynamic (seed services/dynamic/internal/seed/assets.json).
    zone: "assets", label: "Assets",
    items: [],
  },
  {
    zone: "sigv2", label: "SIG", webPort: 3003, apiPort: 8083,
    items: [
      { label: "CMMS Asset Map", path: "/peta" },
      { label: "Unplaced Assets", path: "/belum-bertempat" },
    ],
  },
  {
    zone: "work-order", label: "Work Order", webPort: 3004, apiPort: 8084,
    items: [
      { label: "Work Order Tracking (T&D)", path: "/tracking-tnd" },
      { label: "Run Report Work Order", path: "/run-report" },
    ],
  },
  {
    zone: "dynamic", label: "Dynamic", webPort: 3005, apiPort: 8088,
    items: [
      { label: "Content-Type Builder", path: "/builder", superOnly: true },
      { label: "Components", path: "/components", superOnly: true },
    ],
  },
  {
    // All Master submenus come from dynamic (seed services/dynamic/internal/seed/master.json).
    zone: "master", label: "Master",
    items: [],
  },
  {
    zone: "administration", label: "Administration", webPort: 3007, apiPort: 8087,
    items: [
      { label: "Users Management", path: "/user-management" },
      { label: "Role", path: "/role" },
    ],
  },
];

export const links = (z: ZoneDef): Link[] => z.items.flatMap((i) => ("items" in i ? i.items : [i]));
export const findZone = (zone: string) => MENU.find((z) => z.zone === zone)!;

export const permKey = (zone: string, l: Link) => l.perm ?? zone + l.path;

// Content type metadata from the dynamic service (enough for menus & the RBAC matrix).
export type DynamicMenu = {
  uid: string;
  label: string;
  parent: string;
  hidden?: boolean; // hidden from the sidebar (e.g. user profile), still in the Role matrix
  workflow?: { views: { key: string; label: string; parent: string }[]; transitions: { key: string; label: string }[] } | null;
};
type AccessLike = { superAdmin: boolean; permissions: Record<string, string[]> };

// Dynamic menus shown in the zone: content types (parent) + workflow views (view.parent).
function dynamicLinks(dynamic: DynamicMenu[], zone: string, includeHidden = false): Link[] {
  return dynamic.flatMap((ct) => [
    ...(ct.parent === zone && (includeHidden || !ct.hidden) ? [{ label: ct.label, path: `/c/${ct.uid}`, zone: "dynamic", perm: `ct/${ct.uid}` }] : []),
    ...(ct.workflow?.views ?? [])
      .filter((v) => v.parent === zone)
      .map((v) => ({ label: v.label, path: `/c/${ct.uid}?view=${v.key}`, zone: "dynamic", perm: `ct/${ct.uid}/v/${v.key}` })),
  ]);
}

// Sidebar menu: static + dynamic menus, only those the user can read.
export function buildMenu(dynamic: DynamicMenu[], access: AccessLike | null): ZoneDef[] {
  const canRead = (zone: string, l: Link) =>
    !!access && (access.superAdmin || (!l.superOnly && [permKey(zone, l), "*"].some((k) => access.permissions[k]?.includes("read"))));
  return MENU.map((z) => {
    const items = [...z.items, ...dynamicLinks(dynamic, z.zone)]
      .map((i) => ("items" in i ? { ...i, items: i.items.filter((l) => canRead(z.zone, l)) } : i))
      .filter((i) => ("items" in i ? i.items.length > 0 : canRead(z.zone, i)));
    return { ...z, items };
  }).filter((z) => z.items.length > 0);
}

// RBAC matrix rows per top-level menu: static submenus, dynamic menus, workflow views and workflow transitions.
// Transitions only use the "update" action (= allowed to run the transition).
export function permTree(dynamic: DynamicMenu[]) {
  return MENU.map((z) => ({
    zone: z.zone,
    label: z.label,
    items: [
      ...[...z.items, ...dynamicLinks(dynamic, z.zone, true)]
        .flatMap((i) => ("items" in i ? i.items.map((l) => ({ ...l, label: `${i.label} › ${l.label}` })) : [i]))
        .filter((l) => !l.superOnly)
        .map((l) => ({ key: permKey(z.zone, l), label: l.label, transition: false })),
      ...dynamic
        .filter((ct) => ct.parent === z.zone)
        .flatMap((ct) => (ct.workflow?.transitions ?? []).map((t) => ({ key: `ct/${ct.uid}/t/${t.key}`, label: `${ct.label} › Transition: ${t.label}`, transition: true }))),
    ],
  })).filter((z) => z.items.length > 0);
}

// isLinkActive: a menu is active when zone & path match (including child pages like detail/form)
// AND the ?view= param matches exactly — workflow view menus share the path of their content type list.
export function isLinkActive(link: { zone: string; path: string }, here: { zone?: string; pathname: string; view: string }): boolean {
  const [base, query = ""] = link.path.split("?");
  const view = new URLSearchParams(query).get("view") ?? "";
  return link.zone === here.zone && (here.pathname === base || here.pathname.startsWith(base + "/")) && view === here.view;
}
