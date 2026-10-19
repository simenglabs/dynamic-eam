"use client";

import Link from "next/link";
import { useState, useTransition } from "react";
import { flushSync } from "react-dom";
import { ArrowDown, ArrowLeft, ArrowUp, ChevronDown, ChevronRight, Plus, Save, Search, Trash2, X } from "lucide-react";
import { MENU } from "@eam/ui/menu";
import { Alert, Badge, Button, Card, Field as FormField, Input } from "@eam/ui/ui";
import { confirmDialog } from "@eam/ui/dialog";
import { Combobox } from "@eam/ui/combobox";
import { Checkbox } from "@eam/ui/checkbox";
import { deleteSchema, saveSchema } from "./actions";
import { FIELD_TYPES, type Component, type Condition, type ContentType, type DomainCode, type Field, type FieldType, type Layout, type Workflow } from "./types";
import { LayoutEditor } from "./LayoutEditor";
import { WorkflowEditor } from "./WorkflowEditor";

const TYPE_LABEL: Record<FieldType, string> = {
  text: "Text", longtext: "Long text", number: "Number", boolean: "Boolean", date: "Date",
  select: "Select (enum)", domain: "Domain (master)", relation: "Relation", media: "Media / File", component: "Component",
  attributes: "Atribut classification",
};
let keySeq = 0;
const nextKey = () => String(++keySeq);

const slug = (s: string) => s.toLowerCase().trim().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 48);
const camel = (s: string) =>
  s.trim().replace(/[^a-zA-Z0-9]+(.)?/g, (_, c: string | undefined) => (c ? c.toUpperCase() : "")).replace(/^[^a-zA-Z]+/, "").replace(/^./, (c) => c.toLowerCase()).slice(0, 48);

type Props =
  | { kind: "content-types"; initial?: ContentType; contentTypes: CTInfo[]; components: Component[]; domains: DomainCode[] }
  | { kind: "components"; initial?: Component; contentTypes: CTInfo[]; components: Component[]; domains: DomainCode[] };

type CTInfo = { uid: string; label: string; fields?: Field[] };
type Tab = "general" | "fields" | "layout" | "workflow";

export function SchemaForm(props: Props) {
  const { kind, initial, contentTypes, components, domains } = props;
  const isCT = kind === "content-types";
  const ct = isCT ? (initial as ContentType | undefined) : undefined;
  const editing = !!initial;

  const [label, setLabel] = useState(initial?.label ?? "");
  const [uid, setUid] = useState(initial?.uid ?? "");
  const [uidTouched, setUidTouched] = useState(editing);
  const [parent, setParent] = useState(ct?.parent ?? "dynamic");
  const [hidden, setHidden] = useState(!!ct?.hidden);
  const [scopeField, setScopeField] = useState(ct?.scopeField ?? "");
  // _key = id stabil untuk React (dibuang cleanField sebelum dikirim).
  const [fields, setFields] = useState<(Field & { _key: string })[]>(() => (initial?.fields ?? []).map((f) => ({ ...f, _key: nextKey() })));
  const [displayField, setDisplayField] = useState(ct?.displayField ?? "");
  const [layout, setLayout] = useState<Layout | null>(ct?.layout ?? null);
  const [workflow, setWorkflow] = useState<Workflow | null>(ct?.workflow ?? null);
  const [error, setError] = useState<string>();
  const [pending, start] = useTransition();
  const [tab, setTab] = useState<Tab>("general");
  const [open, setOpen] = useState<Set<string>>(() => new Set(editing ? [] : fields.map((f) => f._key))); // baris field yang dibuka
  const [q, setQ] = useState("");
  const toggle = (k: string) => setOpen((s) => { const n = new Set(s); if (n.has(k)) n.delete(k); else n.add(k); return n; });
  const addField = () => {
    const k = nextKey();
    setFields((fs) => [...fs, { name: "", label: "", type: "text", required: false, _key: k }]);
    setOpen((s) => new Set(s).add(k));
    setQ("");
  };
  const shown = fields.map((f, i) => ({ f, i })).filter(({ f }) => !q || `${f.label} ${f.name} ${f.type}`.toLowerCase().includes(q.toLowerCase()));
  const tabs: { key: Tab; label: string; hint: string }[] = [
    { key: "general", label: "Umum", hint: "nama, menu, label data" },
    { key: "fields", label: `Fields (${fields.length})`, hint: "isi form" },
    ...(isCT ? [
      { key: "layout" as Tab, label: "Layout", hint: "susunan form" },
      { key: "workflow" as Tab, label: workflow ? `Workflow (${workflow.states.length} status)` : "Workflow", hint: "status & approval" },
    ] : []),
  ];

  const update = (i: number, patch: Partial<Field>) => setFields((fs) => fs.map((f, j) => (j === i ? { ...f, ...patch } : f)));
  const move = (i: number, d: number) =>
    setFields((fs) => {
      const next = [...fs];
      [next[i], next[i + d]] = [next[i + d], next[i]];
      return next;
    });


  const submit = () =>
    start(async () => {
      const body = isCT
        ? { uid, label, parent, hidden, scopeField: fields.some((f) => f.name === scopeField) ? scopeField : "", displayField: displayField.trim(), fields: fields.map(cleanField), layout: cleanLayout(layout, fields),
            workflow: workflow && { ...workflow, transitions: workflow.transitions.map((t) => ({ ...t, fields: t.fields?.length ? t.fields.map(cleanField) : undefined })) } }
        : { uid, label, fields: fields.map(cleanField) };
      const res = await saveSchema(kind, initial?.uid, JSON.stringify(body));
      setError(res?.error);
    });

  const remove = async () => {
    if (!initial) return;
    const ok = await confirmDialog({
      title: `Hapus ${initial.label}?`,
      message: isCT ? "Semua data entri dan index pencarian ikut terhapus. Tidak bisa dikembalikan." : "Component yang masih dipakai tidak bisa dihapus.",
      confirmText: "Hapus",
      danger: true,
    });
    if (!ok) return;
    start(async () => setError((await deleteSchema(kind, initial.uid))?.error));
  };

  const back = isCT ? "/builder" : "/components";
  return (
    <form
      action={submit}
      // isian wajib yang kosong ada di tab/baris tertutup: buka dulu supaya browser bisa menunjuk ke field-nya
      onInvalidCapture={(e) => {
        const el = e.target as HTMLElement;
        const t = el.closest<HTMLElement>("[data-tab]")?.dataset.tab as Tab | undefined;
        if (t && t !== tab) flushSync(() => setTab(t));
      }}
    >
      <div className="sticky top-16 z-10 mx-8 mt-4 flex items-center justify-between rounded-2xl border border-slate-200/80 bg-white/90 px-5 py-3 backdrop-blur">
        <Link href={back} className="flex items-center gap-2"><ArrowLeft className="size-5" /> Back</Link>
        <div className="flex gap-2">
          {editing && <Button type="button" variant="ghost" className="text-destructive" onClick={remove} disabled={pending}><Trash2 /> Delete</Button>}
          <Button disabled={pending}><Save /> {pending ? "Saving..." : "Save"}</Button>
        </div>
      </div>

      <div className="flex flex-col gap-4 px-8 py-4">
        {error && <Alert>{error}</Alert>}

        <div role="tablist" aria-label="Pengaturan" className="flex flex-wrap gap-1 rounded-xl border border-slate-200 bg-white p-1">
          {tabs.map((t) => (
            <button key={t.key} type="button" role="tab" aria-selected={tab === t.key} onClick={() => setTab(t.key)}
              className={"flex flex-col rounded-lg px-4 py-2 text-left text-sm transition " + (tab === t.key ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-100")}>
              <span className="font-semibold">{t.label}</span>
              <span className={"text-xs " + (tab === t.key ? "text-white/80" : "text-slate-400")}>{t.hint}</span>
            </button>
          ))}
        </div>

        <div data-tab="general" hidden={tab !== "general"}>
        <Card>
          <h2 className="border-b pb-4 text-lg font-bold">{isCT ? "Content Type" : "Component"}</h2>
          <div className="grid gap-x-4 gap-y-5 md:grid-cols-2">
            <FormField label="Nama" required>
              <Input value={label} required onChange={(e) => { setLabel(e.target.value); if (!uidTouched) setUid(slug(e.target.value)); }} placeholder={isCT ? "mis. Kendaraan" : "mis. Alamat"} />
            </FormField>
            <FormField label="UID" required>
              {editing ? (
                <p className="border-b px-3 py-2 font-normal">{uid}</p>
              ) : (
                <Input value={uid} required pattern="[a-z][a-z0-9-]{1,47}" title="huruf kecil, angka, '-'" onChange={(e) => { setUid(e.target.value); setUidTouched(true); }} />
              )}
            </FormField>
            {isCT && (
              <>
                <FormField label="Tampil di menu" required>
                  <Combobox value={parent} required onChange={setParent} aria-label="Tampil di menu" options={MENU.map((z) => ({ value: z.zone, label: z.label }))} />
                  <Checkbox checked={hidden} onChange={(e) => setHidden(e.target.checked)} className="mt-1 font-normal"
                    label="Sembunyikan dari menu (mis. profil user; izin tetap diatur di Role)" />
                </FormField>
                <FormField label="Batasi data per site" optional>
                  <Combobox value={scopeField} placeholder="— tidak dibatasi —" aria-label="Batasi data per site" onChange={setScopeField}
                    options={fields.filter((f) => f.name && f.type === "relation" && f.target === "master-site" && !f.multiple).map((f) => ({ value: f.name, label: f.label || f.name, hint: f.name }))} />
                  <span className="text-xs font-normal text-slate-500">User hanya melihat & mengubah data di site aksesnya (Profil user). Superadmin tidak dibatasi.</span>
                </FormField>
                <FormField label="Display field" required>
                  <Input
                    value={displayField}
                    required
                    list="display-fields"
                    placeholder="mis. nomor  atau  {nomor} - {nama}"
                    onChange={(e) => setDisplayField(e.target.value)}
                  />
                  <datalist id="display-fields">
                    {fields.filter((f) => f.name && f.type !== "component" && f.type !== "media").map((f) => (
                      <option key={f.name} value={`{${f.name}}`}>{f.label}</option>
                    ))}
                  </datalist>
                  <span className="text-muted-foreground text-xs font-normal">Nama field atau template, contoh <code>{"{nomor} - {nama}"}</code>.</span>
                </FormField>
              </>
            )}
          </div>
          {isCT && (
            <p className="text-muted-foreground text-sm">
              Setelah disimpan, menu ini muncul di sidebar untuk superadmin. Beri akses role lain (read/create/update/delete/download) di Administration › Role.
            </p>
          )}
        </Card>
        </div>

        <div data-tab="fields" hidden={tab !== "fields"}>
        <Card>
          <div className="flex flex-wrap items-center gap-2 border-b pb-4">
            <h2 className="mr-auto text-lg font-bold">Fields</h2>
            <label className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-slate-400" />
              <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Cari field…" aria-label="Cari field" className="w-56 pl-8" />
            </label>
            <Button type="button" variant="ghost" onClick={() => setOpen(new Set(fields.map((f) => f._key)))}>Buka semua</Button>
            <Button type="button" variant="ghost" onClick={() => setOpen(new Set())}>Tutup semua</Button>
            <Button type="button" variant="outline" onClick={addField}><Plus /> Add Field</Button>
          </div>
          {fields.length === 0 && <p className="text-muted-foreground">Belum ada field.</p>}
          {fields.length > 0 && shown.length === 0 && <p className="text-muted-foreground">Tidak ada field yang cocok dengan “{q}”.</p>}
          {shown.map(({ f, i }) => (
            <FieldRow
              key={f._key}
              open={open.has(f._key)}
              onToggle={() => toggle(f._key)}
              f={f}
              first={i === 0}
              last={i === fields.length - 1}
              inComponent={!isCT}
              contentTypes={isCT && !editing && uid ? [...contentTypes, { uid, label: `${label || uid} (ini)` }] : contentTypes}
              components={components.filter((c) => c.uid !== initial?.uid)}
              domains={domains}
              siblings={isCT ? fields : []}
              onChange={(p) => update(i, p)}
              onMove={(d) => move(i, d)}
              onRemove={() => setFields((fs) => fs.filter((_, j) => j !== i))}
            />
          ))}
          {fields.length > 3 && <Button type="button" variant="outline" className="w-fit" onClick={addField}><Plus /> Add Field</Button>}
        </Card>
        </div>

        {isCT && <div data-tab="layout" hidden={tab !== "layout"}><LayoutEditor fields={fields} value={layout} onChange={setLayout} /></div>}
        {isCT && (
          <div data-tab="workflow" hidden={tab !== "workflow"}>
          <WorkflowEditor value={workflow} onChange={setWorkflow}
            renderFields={(tf, set) => (
              <>
                {tf.map((f, i) => (
                  <FieldRow key={i} f={f} first={i === 0} last={i === tf.length - 1} inComponent contentTypes={contentTypes} components={components} domains={domains} siblings={[]}
                    onChange={(p) => set(tf.map((x, j) => (j === i ? { ...x, ...p } : x)))}
                    onMove={(d) => { const n = [...tf]; [n[i], n[i + d]] = [n[i + d], n[i]]; set(n); }}
                    onRemove={() => set(tf.filter((_, j) => j !== i))} />
                ))}
                <Button type="button" variant="outline" className="w-fit" onClick={() => set([...tf, { name: "", label: "", type: "text", required: true }])}><Plus /> Field isian</Button>
              </>
            )} />
          </div>
        )}
      </div>
    </form>
  );
}

// Layout: buang rujukan ke field yang sudah dihapus/diganti nama dan slot kosong di ujung baris.
function cleanLayout(l: Layout | null, fields: Field[]): Layout | null {
  if (!l) return null;
  const names = new Set(fields.map((f) => f.name));
  return {
    sections: l.sections.map((s) => ({
      ...s,
      rows: s.rows.map((r) => {
        const cells = r.fields.map((f) => (names.has(f) ? f : ""));
        while (cells.length && cells[cells.length - 1] === "") cells.pop();
        return { columns: r.columns, fields: cells };
      }),
    })),
  };
}

// Buang properti yang tidak relevan dengan tipe field, konversi default sesuai tipe.
function cleanField(f: Field): Field {
  const out: Field = { name: f.name, label: f.label, type: f.type, required: f.required };
  if ((f.type === "text" || f.type === "number") && f.unique) out.unique = true;
  if (f.type === "select") out.options = f.options;
  if (f.type === "relation") Object.assign(out, { target: f.target, ...(f.where?.length && { where: f.where.filter((c) => c.field && c.value) }) });
  if (f.type === "attributes") out.from = f.from;
  if (f.type === "relation" || f.type === "media") out.multiple = !!f.multiple;
  if (f.type === "component") Object.assign(out, { component: f.component, repeatable: !!f.repeatable, ...(f.repeatable && f.uniqueBy && { uniqueBy: f.uniqueBy }) });
  if (f.type === "domain") out.domain = f.domain?.trim();
  if (f.autoFill?.source) out.autoFill = f.autoFill;
  if (f.type !== "domain" && f.default !== undefined && f.default !== "") {
    out.default = f.type === "number" ? Number(f.default) : f.type === "boolean" ? f.default === "true" || f.default === true : f.default;
  }
  return out;
}

function FieldRow({ f, first, last, inComponent, contentTypes, components, domains, siblings, onChange, onMove, onRemove, open: openProp, onToggle }: {
  open?: boolean; // dikontrol parent (Buka/Tutup semua); tanpa ini dikelola sendiri
  onToggle?: () => void;
  domains: DomainCode[];
  siblings: Field[]; // field lain di content type (sumber isi otomatis dari relasi)
  f: Field;
  first: boolean;
  last: boolean;
  inComponent: boolean;
  contentTypes: CTInfo[];
  components: Component[];
  onChange: (p: Partial<Field>) => void;
  onMove: (d: number) => void;
  onRemove: () => void;
}) {
  const [nameTouched, setNameTouched] = useState(!!f.name);
  const [ownOpen, setOwnOpen] = useState(!f.name);
  const open = openProp ?? ownOpen;
  const toggle = onToggle ?? (() => setOwnOpen((o) => !o));
  const hasDefault = ["text", "longtext", "number", "boolean", "date", "select"].includes(f.type); // domain: default dari master
  const check = (key: "required" | "unique" | "multiple" | "repeatable", text: string) => (
    <label className="flex items-center gap-2 text-sm">
      <Checkbox checked={!!f[key]} onChange={(e) => onChange({ [key]: e.target.checked })} label={text} />
    </label>
  );

  // ringkasan pengaturan supaya baris tertutup tetap informatif
  const tags = [
    f.required && "Wajib", f.unique && "Unik", f.multiple && "Banyak", f.repeatable && "Berulang",
    f.type === "relation" && f.target && `→ ${contentTypes.find((c) => c.uid === f.target)?.label ?? f.target}`,
    f.type === "domain" && f.domain, f.type === "attributes" && f.from && `dari ${f.from}`,
    f.where?.length && `filter ${f.where.length}`, f.autoFill && `otomatis: ${AUTO_LABEL[f.autoFill.source]}`,
  ].filter(Boolean) as string[];
  const actions = (
    <div className="flex shrink-0 items-center gap-0.5">
      <Button type="button" variant="ghost" aria-label="Naikkan" disabled={first} onClick={() => onMove(-1)}><ArrowUp /></Button>
      <Button type="button" variant="ghost" aria-label="Turunkan" disabled={last} onClick={() => onMove(1)}><ArrowDown /></Button>
      <Button type="button" variant="ghost" aria-label="Hapus field" className="text-destructive" onClick={onRemove}><X /></Button>
    </div>
  );
  if (!open)
    return (
      <div className="flex items-center gap-2 rounded-lg border px-3 py-2 hover:border-slate-300">
        <button type="button" onClick={toggle} aria-expanded={false} className="flex min-w-0 flex-1 items-center gap-3 text-left">
          <ChevronRight className="size-4 shrink-0 text-slate-400" />
          <span className="truncate font-semibold">{f.label || <span className="text-slate-400">(tanpa label)</span>}</span>
          <code className="hidden truncate text-xs text-slate-400 sm:inline">{f.name}</code>
          <Badge color="blue" className="shrink-0">{TYPE_LABEL[f.type]}</Badge>
          <span className="hidden min-w-0 flex-wrap gap-1 md:flex">{tags.map((t) => <Badge key={t} className="shrink-0">{t}</Badge>)}</span>
        </button>
        {actions}
      </div>
    );

  return (
    <fieldset className="grid gap-3 rounded-lg border border-primary/40 p-4 md:grid-cols-[1fr_1fr_1fr_auto]">
      <button type="button" onClick={toggle} aria-expanded className="flex items-center gap-2 text-left text-sm font-semibold text-slate-700 md:col-span-full">
        <ChevronDown className="size-4 text-slate-400" /> {f.label || "Field baru"} <span className="font-normal text-slate-400">— klik untuk menutup</span>
      </button>
      <FormField label="Label" required>
        <Input value={f.label} required onChange={(e) => onChange({ label: e.target.value, ...(!nameTouched && { name: camel(e.target.value) }) })} />
      </FormField>
      <FormField label="Nama (API)" required>
        <Input value={f.name} required pattern="[a-z][a-zA-Z0-9]{0,47}" title="camelCase, diawali huruf kecil" onChange={(e) => { onChange({ name: e.target.value }); setNameTouched(true); }} />
      </FormField>
      <FormField label="Tipe" required>
        <Combobox value={f.type} required aria-label="Tipe field" onChange={(v) => onChange({ type: v as FieldType, default: undefined })}
          options={FIELD_TYPES.filter((t) => !inComponent || t !== "attributes").map((t) => ({ value: t, label: TYPE_LABEL[t] }))} />
      </FormField>
      <div className="pt-6">{actions}</div>

      {f.type === "select" && (
        <FormField label="Opsi (pisahkan dengan koma)" required>
          <Input value={(f.options ?? []).join(", ")} required onChange={(e) => onChange({ options: e.target.value.split(",").map((s) => s.trim()).filter(Boolean) })} />
        </FormField>
      )}
      {f.type === "relation" && (
        <FormField label="Relasi ke" required>
          <Combobox value={f.target ?? ""} required aria-label="Relasi ke" onChange={(v) => onChange({ target: v })}
            options={contentTypes.map((c) => ({ value: c.uid, label: c.label, hint: c.uid }))} />
        </FormField>
      )}
      {f.type === "relation" && (
        <WhereEditor f={f} siblings={inComponent ? [] : siblings} targetFields={contentTypes.find((c) => c.uid === f.target)?.fields ?? []} onChange={onChange} />
      )}
      {f.type === "attributes" && (
        <FormField label="Dari classification" required>
          <Combobox value={f.from ?? ""} required aria-label="Field classification" onChange={(v) => onChange({ from: v })}
            options={siblings.filter((x) => x.name && x.type === "relation" && !x.multiple && x.target === "master-classification").map((x) => ({ value: x.name, label: x.label || x.name }))} />
          <span className="text-xs font-normal text-slate-500">Field yang muncul di form = atribut classification terpilih (Master › Classification › Attributes).</span>
        </FormField>
      )}
      {f.type === "domain" && (
        <FormField label="Kode domain" required>
          <Input value={f.domain ?? ""} required list="domain-codes" placeholder="mis. PRIORITY" onChange={(e) => onChange({ domain: e.target.value.toUpperCase() })} />
          <datalist id="domain-codes">
            {domains.map((d) => <option key={d.code} value={d.code}>{d.description}</option>)}
          </datalist>
          <span className="text-xs font-normal text-slate-500">Opsi & nilai default diambil dari Master › Domain.</span>
        </FormField>
      )}
      {f.type === "component" && f.repeatable && (
        <FormField label="Unik berdasarkan" optional>
          <Combobox value={f.uniqueBy ?? ""} placeholder="— tidak ada —" aria-label="Unik berdasarkan" onChange={(v) => onChange({ uniqueBy: v || undefined })}
            options={(components.find((c) => c.uid === f.component)?.fields ?? []).map((cf) => ({ value: cf.name, label: cf.label, hint: cf.name }))} />
        </FormField>
      )}
      {f.type === "component" && (
        <FormField label="Component" required>
          <Combobox value={f.component ?? ""} required aria-label="Component" onChange={(v) => onChange({ component: v })}
            options={components.map((c) => ({ value: c.uid, label: c.label, hint: c.uid }))} />
        </FormField>
      )}
      {hasDefault && (
        <FormField label="Default" optional>
          {f.type === "boolean" ? (
            <Combobox value={f.default === undefined ? "" : String(f.default)} placeholder="-- none --" aria-label="Default" onChange={(v) => onChange({ default: v || undefined })}
              options={[{ value: "true", label: "true" }, { value: "false", label: "false" }]} />
          ) : f.type === "select" ? (
            <Combobox value={String(f.default ?? "")} placeholder="-- none --" aria-label="Default" onChange={(v) => onChange({ default: v || undefined })}
              options={(f.options ?? []).map((o) => ({ value: o, label: o }))} />
          ) : (
            <Input type={f.type === "number" ? "number" : f.type === "date" ? "date" : "text"} value={String(f.default ?? "")} onChange={(e) => onChange({ default: e.target.value })} />
          )}
        </FormField>
      )}
      {!inComponent && f.type !== "component" && f.type !== "media" && (
        <AutoFillEditor f={f} siblings={siblings} contentTypes={contentTypes} onChange={onChange} />
      )}
      <div className="flex flex-wrap items-end gap-4 pb-2 md:col-span-full">
        {check("required", "Required")}
        {!inComponent && (f.type === "text" || f.type === "number") && check("unique", "Unique")}
        {(f.type === "relation" || f.type === "media") && check("multiple", f.type === "relation" ? "Banyak (many)" : "Banyak file")}
        {f.type === "component" && check("repeatable", "Repeatable")}
      </div>
    </fieldset>
  );
}

// Filter pilihan relasi: field target = nilai tetap atau {field form}.
function WhereEditor({ f, siblings, targetFields, onChange }: { f: Field; siblings: Field[]; targetFields: Field[]; onChange: (p: Partial<Field>) => void }) {
  const where = f.where ?? [];
  const set = (i: number, p: Partial<Condition>) => onChange({ where: where.map((c, j) => (j === i ? { ...c, ...p } : c)) });
  const refs = siblings.filter((x) => x.name && x.name !== f.name).map((x) => `{${x.name}}`);
  return (
    <div className="grid gap-2 rounded-md bg-slate-50 p-3 md:col-span-full">
      <div className="text-sm font-medium">Filter pilihan (where) <span className="font-normal text-slate-500">— semua kondisi harus cocok; nilai {"{field}"} diambil dari form</span></div>
      {where.map((c, i) => (
        <div key={i} className="grid items-center gap-2 md:grid-cols-[1fr_auto_1fr_auto]">
          <Combobox value={c.field} aria-label="Field target" placeholder="field di target"
            onChange={(v) => set(i, { field: v })}
            options={[...targetFields.map((x) => ({ value: x.name, label: x.label, hint: x.name })), { value: "workflowStatus", label: "Status workflow", hint: "workflowStatus" }]} />
          <span className="text-sm text-slate-500">=</span>
          <Input value={c.value} aria-label="Nilai filter" list={`where-refs-${f.name}`} placeholder="nilai atau {field}" onChange={(e) => set(i, { value: e.target.value })} />
          <Button type="button" variant="ghost" aria-label="Hapus kondisi" className="text-destructive" onClick={() => onChange({ where: where.filter((_, j) => j !== i) })}><X /></Button>
        </div>
      ))}
      <datalist id={`where-refs-${f.name}`}>{refs.map((r) => <option key={r} value={r} />)}</datalist>
      <div><Button type="button" variant="outline" onClick={() => onChange({ where: [...where, { field: "", value: "" }] })}>+ Kondisi</Button></div>
    </div>
  );
}

const AUTO_LABEL = { relation: "Dari relasi", profile: "Dari profil user", today: "Tanggal hari ini", user: "User login (nama / pemilik)", pattern: "Kode otomatis (pola)" } as const;

// Pengaturan isi otomatis satu field.
function AutoFillEditor({ f, siblings, contentTypes, onChange }: { f: Field; siblings: Field[]; contentTypes: CTInfo[]; onChange: (p: Partial<Field>) => void }) {
  const af = f.autoFill;
  const set = (p: Partial<NonNullable<Field["autoFill"]>>) => onChange({ autoFill: { ...(af ?? { source: "relation" }), ...p } });
  const relations = siblings.filter((x) => x.name && x.name !== f.name && x.type === "relation" && !x.multiple);
  const from = relations.find((x) => x.name === af?.from);
  const targetFields = contentTypes.find((c) => c.uid === from?.target)?.fields ?? [];
  const profileFields = contentTypes.find((c) => c.uid === "user-profile")?.fields ?? [];
  // Default: relasi tunggal tepat sebelum field ini (mis. Classification -> Class Description) dan field "description" di targetnya.
  const descOf = (rel?: Field) => {
    const fs = contentTypes.find((c) => c.uid === rel?.target)?.fields ?? [];
    return (fs.find((x) => x.name === "description") ?? fs.find((x) => /desc/i.test(x.name)))?.name ?? "";
  };
  const before = siblings.slice(0, siblings.indexOf(f)).reverse().find((x) => relations.includes(x)) ?? relations[0];
  return (
    <div className="grid gap-3 rounded-md bg-slate-50 p-3 md:col-span-full md:grid-cols-4">
      <FormField label="Isi otomatis" optional>
        <Combobox value={af?.source ?? ""} placeholder="— tidak —" aria-label="Sumber isi otomatis"
          onChange={(v) => onChange({ autoFill: !v ? undefined : v === "pattern" ? { source: v, pattern: af?.pattern ?? "{seq:5}", lock: true } : v === "relation" ? { source: v, from: before?.name, field: descOf(before), lock: af?.lock ?? true } : { source: v as "relation", lock: af?.lock } })}
          options={Object.entries(AUTO_LABEL).filter(([v]) => v !== "pattern" || f.type === "text").map(([value, label]) => ({ value, label }))} />
      </FormField>
      {af?.source === "relation" && (
        <>
          <FormField label="Field relasi" required>
            <Combobox value={af.from ?? ""} required aria-label="Field relasi" onChange={(v) => set({ from: v, field: descOf(relations.find((x) => x.name === v)) })}
              options={relations.map((x) => ({ value: x.name, label: x.label || x.name, hint: x.target }))} />
          </FormField>
          <FormField label="Ambil field" required>
            <Combobox value={af.field ?? ""} required aria-label="Field yang diambil" onChange={(v) => set({ field: v })}
              options={targetFields.map((x) => ({ value: x.name, label: x.label, hint: x.name }))} />
          </FormField>
        </>
      )}
      {af?.source === "pattern" && (
        <div className="md:col-span-2">
          <FormField label="Pola kode" required>
            <Input value={af.pattern ?? ""} required aria-label="Pola kode" placeholder="AST-{YYYY}-{seq:5}" onChange={(e) => set({ pattern: e.target.value })} />
          </FormField>
          {/* token siap pakai: klik untuk menambahkan ke pola */}
          <div className="mt-2 flex flex-wrap gap-1.5">
            {["{seq:4}", "{YYYY}", "{YY}", "{MM}", "{DD}",
              ...relations.map((r) => `{${r.name}.${contentTypes.find((c) => c.uid === r.target)?.fields?.[0]?.name ?? "id"}}`),
              ...siblings.filter((x) => x.name && x.name !== f.name && ["text", "select", "domain"].includes(x.type)).map((x) => `{${x.name}}`),
            ].map((tok) => (
              <button key={tok} type="button" className="rounded border bg-white px-1.5 py-0.5 font-mono text-xs text-slate-600 hover:border-primary hover:text-primary"
                onClick={() => set({ pattern: (af.pattern ?? "") + tok })}>{tok}</button>
            ))}
          </div>
          <p className="mt-1 text-xs text-slate-500">{"{seq:N}"} nomor urut N digit, urutan terpisah per hasil pola lain (mis. per kode classification).</p>
        </div>
      )}
      {af?.source === "profile" && (
        <FormField label="Field profil" required>
          <Combobox value={af.field ?? ""} required aria-label="Field profil" onChange={(v) => set({ field: v })}
            options={profileFields.map((x) => ({ value: x.name, label: x.label, hint: x.name }))} />
        </FormField>
      )}
      {af && (
        <div className="flex items-end pb-2">
          <Checkbox checked={!!af.lock} onChange={(e) => set({ lock: e.target.checked })} label="Kunci (selalu otomatis, tidak bisa diedit)" />
        </div>
      )}
    </div>
  );
}
