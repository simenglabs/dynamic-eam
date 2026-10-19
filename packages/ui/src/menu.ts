// Satu menu besar = satu zone (Next.js app terpisah, basePath /<zone>) + satu service Go.
// Sub menu = route di dalam zone. Port di sini jadi sumber tunggal untuk shell rewrites & api().
// zone = zone pemilik halaman kalau beda dari menu induknya (mis. menu dinamis di bawah "Master"
// yang halamannya dirender zone "dynamic"). perm = kunci RBAC (default "<zone><path>", mis. "assets/tnd").
// superOnly = hanya superadmin (tidak diatur lewat role).
export type Link = { label: string; path: string; zone?: string; perm?: string; superOnly?: boolean };
export type Item = Link | { label: string; items: Link[] };
// webPort/apiPort kosong = zone tanpa app & service sendiri: hanya wadah menu dinamis (dirender zone "dynamic").
export type ZoneDef = { zone: string; label: string; webPort?: number; apiPort?: number; items: Item[] };

export const MENU: ZoneDef[] = [
  {
    zone: "renkon", label: "Renkon", webPort: 3001, apiPort: 8081,
    items: [
      { label: "PRK", items: [
        { label: "PRK Unit Induk", path: "/prk" },
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
        { label: "Template KHS", path: "/khs/template" },
        { label: "Unit KHS", path: "/khs/master-unit" },
      ] },
      { label: "Daftar Material", path: "/material" },
      { label: "Daftar Permohonan", path: "/permohonan" },
      { label: "Perencanaan", path: "/perencanaan" },
      { label: "Approval Perencanaan", path: "/perencanaan/approval" },
    ],
  },
  {
    // Semua sub menu Assets dibuat dari dynamic (seed services/dynamic/internal/seed/assets.json).
    zone: "assets", label: "Assets",
    items: [],
  },
  {
    zone: "sigv2", label: "SIG", webPort: 3003, apiPort: 8083,
    items: [
      { label: "Peta Aset CMMS", path: "/peta" },
      { label: "Aset Belum Bertempat", path: "/belum-bertempat" },
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
    // Semua sub menu Master dibuat dari dynamic (seed services/dynamic/internal/seed/master.json).
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

// Metadata content type dari service dynamic (cukup untuk menu & matriks RBAC).
export type DynamicMenu = {
  uid: string;
  label: string;
  parent: string;
  hidden?: boolean; // tidak tampil di sidebar (mis. profil user), tetap ada di matriks Role
  workflow?: { views: { key: string; label: string; parent: string }[]; transitions: { key: string; label: string }[] } | null;
};
type AccessLike = { superAdmin: boolean; permissions: Record<string, string[]> };

// Menu dinamis yang muncul di zone tsb: content type (parent) + view workflow (view.parent).
function dynamicLinks(dynamic: DynamicMenu[], zone: string, includeHidden = false): Link[] {
  return dynamic.flatMap((ct) => [
    ...(ct.parent === zone && (includeHidden || !ct.hidden) ? [{ label: ct.label, path: `/c/${ct.uid}`, zone: "dynamic", perm: `ct/${ct.uid}` }] : []),
    ...(ct.workflow?.views ?? [])
      .filter((v) => v.parent === zone)
      .map((v) => ({ label: v.label, path: `/c/${ct.uid}?view=${v.key}`, zone: "dynamic", perm: `ct/${ct.uid}/v/${v.key}` })),
  ]);
}

// Menu untuk sidebar: menu statis + menu dinamis, hanya yang boleh di-read.
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

// Baris matriks RBAC per menu besar: sub menu statis, menu dinamis, view workflow, dan transisi workflow.
// Transisi hanya memakai aksi "update" (= boleh menjalankan transisi).
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
        .flatMap((ct) => (ct.workflow?.transitions ?? []).map((t) => ({ key: `ct/${ct.uid}/t/${t.key}`, label: `${ct.label} › Transisi: ${t.label}`, transition: true }))),
    ],
  })).filter((z) => z.items.length > 0);
}

// isLinkActive: menu aktif bila zone & path cocok (termasuk halaman turunan seperti detail/form)
// DAN parameter ?view= sama persis — menu view workflow berbagi path dengan list content type-nya.
export function isLinkActive(link: { zone: string; path: string }, here: { zone?: string; pathname: string; view: string }): boolean {
  const [base, query = ""] = link.path.split("?");
  const view = new URLSearchParams(query).get("view") ?? "";
  return link.zone === here.zone && (here.pathname === base || here.pathname.startsWith(base + "/")) && view === here.view;
}
