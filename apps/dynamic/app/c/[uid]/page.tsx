import Link from "next/link";
import { Download, Plus, Search } from "lucide-react";
import { Badge, Button, Card, Input, MENU, PageHeader, Pagination, Table, Td, Th, Tr, api, can, getMe } from "@eam/ui";
import { guard, isElement } from "../../_lib/guard";
import { displayLabel, stateOf, type ContentType, type DomainValue, type Entry, type Field, type Option } from "../../_lib/types";

const LISTABLE = ["text", "number", "boolean", "date", "select", "domain", "relation"];

type Labels = { options: Record<string, Option[]>; domains: Record<string, DomainValue[]> };

function cell(f: Field, v: unknown, lk: Labels) {
  if (v === undefined || v === null || v === "" || (Array.isArray(v) && v.length === 0)) return <span className="text-muted-foreground">–</span>;
  if (Array.isArray(v)) return v.map((x) => label(f, x, lk)).join(", ");
  if (f.type === "boolean") return v ? <Badge color="green">Yes</Badge> : <Badge>No</Badge>;
  if (f.type === "select") return <Badge color="blue">{String(v)}</Badge>;
  if (f.type === "domain") return <Badge color="blue">{label(f, v, lk)}</Badge>;
  return label(f, v, lk);
}

function label(f: Field, v: unknown, lk: Labels): string {
  if (f.type === "relation") return lk.options[f.target!]?.find((o) => o.id === v)?.label ?? "…"; // ponytail: only the first 100 options per target
  if (f.type === "domain") return lk.domains[f.domain!]?.find((d) => d.valueId === v)?.valueDescription ?? String(v);
  if (f.type === "date") return new Date(String(v)).toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" });
  if (f.type === "number" && typeof v === "number") return v.toLocaleString("en-GB");
  return String(v);
}

export default async function Entries({ params, searchParams }: {
  params: Promise<{ uid: string }>;
  searchParams: Promise<{ q?: string; page?: string; view?: string; status?: string }>;
}) {
  const [{ uid }, { q = "", page = "1", view = "", status = "" }, me] = await Promise.all([params, searchParams, getMe()]);
  const ct = await guard(api<ContentType>("dynamic", `/content-types/${encodeURIComponent(uid)}`), ["Dynamic", uid]);
  if (isElement(ct)) return ct;
  const v = ct.workflow?.views.find((x) => x.key === view);
  const title = v?.label ?? ct.label;
  const crumbs = [MENU.find((m) => m.zone === (v?.parent ?? ct.parent))?.label ?? ct.parent, title];
  const qs = new URLSearchParams({ q, page, size: "10", ...(v && { view }), ...(!v && status && { status }) });
  const res = await guard(api<{ total: number; page: number; size: number; items: Entry[] }>("dynamic", `/content-types/${encodeURIComponent(uid)}/entries?${qs}`), crumbs);
  if (isElement(res)) return res;
  const detail = (id: string) => `/c/${uid}/${id}${v ? `?view=${view}` : ""}`;

  // Fields already shown in the label column (display field / template "{code} - {name}") are not repeated.
  const refs = [...ct.displayField.matchAll(/\{(\w+)\}/g)].map((m) => m[1]);
  const inTitle = new Set(refs.length ? refs : [ct.displayField]);
  const titleCol: Field = { name: "__display", label: inTitle.size === 1 ? (ct.fields.find((f) => inTitle.has(f.name))?.label ?? "Record") : "Record", type: "text", required: false };
  // Columns: label first, then the first 5 listable fields (relations & domains show labels, not IDs).
  const cols = [titleCol, ...ct.fields.filter((f) => !inTitle.has(f.name) && LISTABLE.includes(f.type)).slice(0, 5)];
  const targets = [...new Set(cols.filter((f) => f.type === "relation").map((f) => f.target!))];
  const codes = [...new Set(cols.filter((f) => f.type === "domain").map((f) => f.domain!))];
  const [opts, vals] = await Promise.all([
    Promise.all(targets.map((t) => api<Option[]>("dynamic", `/content-types/${encodeURIComponent(t)}/options`).catch(() => [] as Option[]))),
    Promise.all(codes.map((c) => api<DomainValue[]>("dynamic", `/domains/${encodeURIComponent(c)}/values`).catch(() => [] as DomainValue[]))),
  ]);
  const lk: Labels = { options: Object.fromEntries(targets.map((t, i) => [t, opts[i]])), domains: Object.fromEntries(codes.map((c, i) => [c, vals[i]])) };
  const canCreate = !v && can(me, `ct/${uid}`, "create");
  const link = (p: Record<string, string>) => `/c/${uid}?${new URLSearchParams(Object.entries({ q, status, ...(v && { view }), ...p }).filter(([, x]) => x))}`;

  return (
    <>
      <PageHeader crumbs={crumbs} title={title} />
      <div className="px-8 py-4">
        <Card>
          <div className="flex items-center justify-between border-b pb-4">
            <h2 className="text-lg font-bold">{title} <span className="text-muted-foreground text-sm font-normal">· {res.total} records</span></h2>
            <div className="flex gap-2">
              {!v && can(me, `ct/${uid}`, "download") && (
                // plain <a>: file comes from the Go service via the shell rewrite, not a Next page.
                <a href={`/api/dynamic/content-types/${uid}/export`} download>
                  <Button variant="outline" tabIndex={-1}><Download /> Download CSV</Button>
                </a>
              )}
              {canCreate && (
                <Link href={`/c/${uid}/new`}><Button tabIndex={-1}><Plus /> Add New {ct.label}</Button></Link>
              )}
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <form className="relative w-fit">
              {v && <input type="hidden" name="view" value={view} />}
              {status && <input type="hidden" name="status" value={status} />}
              <Search className="text-muted-foreground absolute top-3 left-2.5 size-4" />
              <Input type="search" name="q" defaultValue={q} placeholder="Search and press Enter…" aria-label={`Search ${ct.label}`} className="w-60 pl-8" />
            </form>
            {/* status filter (not in views: a view already filters by status) */}
            {ct.workflow && !v && (
              <nav aria-label="Filter status" className="flex flex-wrap gap-1">
                {[{ key: "", label: "All" }, ...ct.workflow.states].map((st) => (
                  <Link key={st.key} href={link({ status: st.key, page: "" })} aria-current={status === st.key ? "page" : undefined}
                    className={"rounded-full border px-3 py-1 text-sm " + (status === st.key ? "border-primary bg-primary text-white" : "text-slate-600 hover:bg-slate-100")}>
                    {st.label}
                  </Link>
                ))}
              </nav>
            )}
            {(q || status) && <Link href={link({ q: "", status: "" })} className="text-primary text-sm">Reset filters</Link>}
          </div>
          <Table>
            <thead><tr>{cols.map((f) => <Th key={f.name}>{f.label}</Th>)}{ct.workflow && <Th>Status</Th>}</tr></thead>
            <tbody>
              {res.items.map((e) => (
                <Tr key={e.id}>
                  {cols.map((f, i) => (
                    <Td key={f.name}>
                      {i === 0 ? (
                        <Link href={detail(e.id)} className="text-primary underline underline-offset-3">{displayLabel(ct.displayField, e) || e.id}</Link>
                      ) : (
                        cell(f, e[f.name], lk)
                      )}
                    </Td>
                  ))}
                  {ct.workflow && (
                    <Td>
                      <Badge color={stateOf(ct, e.workflowStatus)?.color}>{stateOf(ct, e.workflowStatus)?.label ?? e.workflowStatus ?? "–"}</Badge>
                    </Td>
                  )}
                </Tr>
              ))}
              {res.items.length === 0 && (
                <tr>
                  <td colSpan={cols.length + (ct.workflow ? 1 : 0)} className="text-muted-foreground p-8 text-center">
                    {q || status ? "No records match the filters." : "No records yet."}
                    {canCreate && !q && !status && <> <Link href={`/c/${uid}/new`} className="text-primary underline">Add the first {ct.label}</Link></>}
                  </td>
                </tr>
              )}
            </tbody>
          </Table>
          <Pagination page={res.page} size={res.size} total={res.total} href={(p) => link({ page: String(p) })} />
        </Card>
      </div>
    </>
  );
}
