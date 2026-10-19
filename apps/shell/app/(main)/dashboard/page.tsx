import { Activity, Boxes, CircleAlert, CircleCheck, Clock3, type LucideIcon } from "lucide-react";
import { Badge, Card, PageHeader, Table, Td, Th, Tr, api, buildMenu, can, getMe, links, type DynamicMenu } from "@eam/ui";

type State = { key: string; label: string; color: "neutral" | "green" | "red" | "yellow" | "blue" };
type CT = { uid: string; label: string; displayField: string; workflow: { states: State[] } | null };
type Page = { total: number; items: (Record<string, unknown> & { id: string; workflowStatus?: string; updatedAt: string })[] };

const ASSET = "asset-tnd"; // dashboard summary from the dynamic Asset (T&D) menu
const ICON: Record<State["color"], LucideIcon> = { green: CircleCheck, yellow: Clock3, red: CircleAlert, blue: Activity, neutral: Activity };
const TILE: Record<State["color"], string> = {
  green: "bg-green-50 text-green-700", yellow: "bg-amber-50 text-amber-700", red: "bg-red-50 text-red-700",
  blue: "bg-blue-50 text-blue-700", neutral: "bg-slate-100 text-slate-600",
};
const BAR: Record<State["color"], string> = { green: "bg-green-600", yellow: "bg-amber-500", red: "bg-red-600", blue: "bg-blue-600", neutral: "bg-slate-400" };
const fmt = new Intl.NumberFormat("en-GB");

const label = (display: string, d: Record<string, unknown>) =>
  /[{}]/.test(display) ? display.replace(/\{([a-zA-Z][a-zA-Z0-9]*)\}/g, (_, k) => String(d[k] ?? "")) : String(d[display] ?? "");

export default async function Dashboard() {
  const me = await getMe();
  const dynamic = await api<DynamicMenu[]>("dynamic", "/content-types").catch(() => []);
  const menu = buildMenu(dynamic, me?.access ?? null);
  const canAssets = can(me, `ct/${ASSET}`, "read");

  // Per-status counts via ?status= (Elasticsearch). On failure show "–", the page still works.
  let ct: CT | null = null;
  let total: number | null = null;
  let counts: Record<string, number | null> = {};
  let recent: Page["items"] = [];
  if (canAssets) {
    ct = await api<CT>("dynamic", `/content-types/${ASSET}`).catch(() => null);
    const states = ct?.workflow?.states ?? [];
    const base = `/content-types/${ASSET}/entries`;
    const [all, ...per] = await Promise.all([
      api<Page>("dynamic", `${base}?size=5`).catch(() => null),
      ...states.map((s) => api<Page>("dynamic", `${base}?size=1&status=${s.key}`).catch(() => null)),
    ]);
    total = all?.total ?? null;
    recent = all?.items ?? [];
    counts = Object.fromEntries(states.map((s, i) => [s.key, per[i]?.total ?? null]));
  }
  const states = ct?.workflow?.states ?? [];
  const max = Math.max(1, ...Object.values(counts).map((n) => n ?? 0));

  return (
    <>
      <PageHeader crumbs={["Home"]} title="Dashboard" subtitle="Overview of asset condition and operational activity" />
      <div className="flex flex-col gap-5 px-8 py-4">
        {ct && (
          <>
            <section aria-label="Asset summary" className="grid gap-5 sm:grid-cols-2 xl:grid-cols-4">
              <Kpi icon={Boxes} tone="bg-primary-50 text-primary-700" label="Total Assets" value={total} href={`/dynamic/c/${ASSET}`} />
              {states.slice(0, 3).map((s) => (
                <Kpi key={s.key} icon={ICON[s.color]} tone={TILE[s.color]} label={`${s.label} Assets`} value={counts[s.key]}
                  hint={total && counts[s.key] != null ? `${((counts[s.key]! / total) * 100).toFixed(1)}% of total` : undefined} />
              ))}
            </section>

            <div className="grid gap-5 xl:grid-cols-[1fr_1.6fr]">
              <Card>
                <h2 className="font-semibold text-slate-900">Asset Status</h2>
                {total === null ? (
                  <Empty />
                ) : (
                  <ul className="flex flex-col gap-4">
                    {states.map((s) => {
                      const n = counts[s.key] ?? 0;
                      return (
                        <li key={s.key} className="grid grid-cols-[8.5rem_1fr_4.5rem] items-center gap-3 text-sm" title={`${s.label}: ${fmt.format(n)} assets`}>
                          <span className="truncate text-slate-600">{s.label}</span>
                          <span className="h-2.5 rounded-full bg-slate-100">
                            <span className={`block h-2.5 rounded-full ${BAR[s.color]}`} style={{ width: `${(n / max) * 100}%`, minWidth: n ? 6 : 0 }} />
                          </span>
                          <span className="text-right font-semibold text-slate-900 tabular-nums">{fmt.format(n)}</span>
                        </li>
                      );
                    })}
                  </ul>
                )}
              </Card>

              <Card className="gap-3">
                <div className="flex items-center justify-between">
                  <h2 className="font-semibold text-slate-900">Latest Assets</h2>
                  <a href={`/dynamic/c/${ASSET}`} className="text-primary-700 text-sm font-medium hover:underline">View all</a>
                </div>
                {total === null ? (
                  <Empty />
                ) : recent.length === 0 ? (
                  <p className="py-8 text-center text-sm text-slate-500">No assets yet.</p>
                ) : (
                  <Table>
                    <thead><tr>{["Asset", "Status", "Updated"].map((h) => <Th key={h}>{h}</Th>)}</tr></thead>
                    <tbody>
                      {recent.map((e) => {
                        const st = states.find((s) => s.key === e.workflowStatus);
                        return (
                          <Tr key={e.id}>
                            <Td><a href={`/dynamic/c/${ASSET}/${e.id}`} className="font-medium text-slate-900 hover:underline">{label(ct!.displayField, e) || e.id}</a></Td>
                            <Td><Badge color={st?.color}>{st?.label ?? e.workflowStatus ?? "–"}</Badge></Td>
                            <Td className="text-slate-500">{new Date(e.updatedAt).toLocaleString("en-GB", { dateStyle: "medium", timeStyle: "short" })}</Td>
                          </Tr>
                        );
                      })}
                    </tbody>
                  </Table>
                )}
              </Card>
            </div>
          </>
        )}

        <section aria-label="Menu" className="grid gap-5 sm:grid-cols-2 xl:grid-cols-4">
          {menu.map((z) => {
            const first = links(z)[0]; // first submenu the user may open
            return (
              <a key={z.zone} href={`/${first.zone ?? z.zone}${first.path}`} className="group rounded-2xl border border-slate-200/80 bg-white p-5 transition-colors hover:border-primary-200">
                <p className="font-semibold text-slate-900 group-hover:text-primary-700">{z.label}</p>
                <p className="mt-1 text-sm text-slate-500">{links(z).length} submenus</p>
              </a>
            );
          })}
          {menu.length === 0 && <p className="text-slate-500">No accessible menus yet. Contact your administrator.</p>}
        </section>
      </div>
    </>
  );
}

function Kpi({ icon: Icon, tone, label, value, hint, href }: { icon: LucideIcon; tone: string; label: string; value: number | null | undefined; hint?: string; href?: string }) {
  const body = (
    <Card className="h-full flex-row items-start gap-4 p-5">
      <span className={`flex size-11 shrink-0 items-center justify-center rounded-xl ${tone}`}><Icon className="size-5" /></span>
      <span className="flex flex-col">
        <span className="text-sm text-slate-500">{label}</span>
        <span className="text-[1.7rem] leading-tight font-bold tracking-tight text-slate-900 tabular-nums">{value == null ? "–" : fmt.format(value)}</span>
        {hint && <span className="mt-1 text-xs text-slate-500">{hint}</span>}
      </span>
    </Card>
  );
  return href ? <a href={href} className="rounded-2xl">{body}</a> : body;
}

function Empty() {
  return <p className="rounded-xl bg-slate-50 py-8 text-center text-sm text-slate-500">Search data is not available yet (Elasticsearch is unreachable).</p>;
}
