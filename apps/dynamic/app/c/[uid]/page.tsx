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
  if (f.type === "boolean") return v ? <Badge color="green">Ya</Badge> : <Badge>Tidak</Badge>;
  if (f.type === "select") return <Badge color="blue">{String(v)}</Badge>;
  if (f.type === "domain") return <Badge color="blue">{label(f, v, lk)}</Badge>;
  return label(f, v, lk);
}

function label(f: Field, v: unknown, lk: Labels): string {
  if (f.type === "relation") return lk.options[f.target!]?.find((o) => o.id === v)?.label ?? "…"; // ponytail: hanya 100 opsi pertama per target
  if (f.type === "domain") return lk.domains[f.domain!]?.find((d) => d.valueId === v)?.valueDescription ?? String(v);
  if (f.type === "date") return new Date(String(v)).toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" });
  if (f.type === "number" && typeof v === "number") return v.toLocaleString("id-ID");
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

  // Kolom: label (display field / template) dulu, lalu maksimal 4 field skalar lain.
  // field yang sudah tampil di kolom label (display field / template "{kode} - {nama}") tidak diulang
  const refs = [...ct.displayField.matchAll(/\{(\w+)\}/g)].map((m) => m[1]);
  const inTitle = new Set(refs.length ? refs : [ct.displayField]);
  const titleCol: Field = { name: "__display", label: inTitle.size === 1 ? (ct.fields.find((f) => inTitle.has(f.name))?.label ?? "Data") : "Data", type: "text", required: false };
  // Kolom: label dulu, lalu 5 field pertama yang bisa ditampilkan (relasi & domain pakai label, bukan ID).
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
            <h2 className="text-lg font-bold">{title} <span className="text-muted-foreground text-sm font-normal">· {res.total} data</span></h2>
            <div className="flex gap-2">
              {!v && can(me, `ct/${uid}`, "download") && (
                // <a> biasa: file dari service Go lewat rewrite shell, bukan halaman Next.
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
              <Input type="search" name="q" defaultValue={q} placeholder="Cari lalu Enter…" aria-label={`Cari ${ct.label}`} className="w-60 pl-8" />
            </form>
            {/* filter status (bukan di view: view sudah memfilter status) */}
            {ct.workflow && !v && (
              <nav aria-label="Filter status" className="flex flex-wrap gap-1">
                {[{ key: "", label: "Semua" }, ...ct.workflow.states].map((st) => (
                  <Link key={st.key} href={link({ status: st.key, page: "" })} aria-current={status === st.key ? "page" : undefined}
                    className={"rounded-full border px-3 py-1 text-sm " + (status === st.key ? "border-primary bg-primary text-white" : "text-slate-600 hover:bg-slate-100")}>
                    {st.label}
                  </Link>
                ))}
              </nav>
            )}
            {(q || status) && <Link href={link({ q: "", status: "" })} className="text-primary text-sm">Reset filter</Link>}
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
                    {q || status ? "Tidak ada data yang cocok dengan filter." : "Belum ada data."}
                    {canCreate && !q && !status && <> <Link href={`/c/${uid}/new`} className="text-primary underline">Tambah {ct.label} pertama</Link></>}
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
