"use client";

import Link from "next/link";
import { useEffect, useRef, useState, useTransition } from "react";
import { flushSync } from "react-dom";
import { ArrowLeft, FileText, Plus, Save, Trash2, Upload, X } from "lucide-react";
import { Alert, Badge, Button, Card, Field as FormField, Input, Textarea } from "@eam/ui/ui";
import { confirmDialog, promptDialog } from "@eam/ui/dialog";
import { Combobox } from "@eam/ui/combobox";
import { Checkbox } from "@eam/ui/checkbox";
import { deleteEntry, runTransition, saveEntry } from "./actions";
import { describeChange } from "./history";
import { editable, effectiveLayout, stateOf, type Component, type ContentType, type Entry, type Field, type Media, type Option, type Revision, type DomainValue, type AttributeDef, type Transition, resolveWhere } from "./types";

type Data = Record<string, unknown>;
type Ctx = { components: Record<string, Component>; options: Record<string, Option[]>; domains: Record<string, DomainValue[]>; form: Data; me?: Option };

const META = ["id", "version", "createdAt", "updatedAt", "createdBy", "updatedBy", "workflowStatus", "workflowHistory"];

export function EntryForm({ ct, entry, history = [], components, options, domains = {}, canSave, canDelete, transitions, view, saveAction, back, placement }: {
  placement?: { defaultSite?: string; organization?: string; userId?: string; userName?: string; profile?: Record<string, unknown> }; // auto fill from user & profile
  saveAction?: (json: string) => Promise<{ error?: string }>; // default: saveEntry (create/update a regular entry)
  back?: { href: string; label: string }; // plain <a>: may point to another zone
  domains?: Record<string, DomainValue[]>;
  history?: Revision[];
  ct: ContentType;
  entry?: Entry;
  components: Record<string, Component>;
  options: Record<string, Option[]>;
  canSave: boolean;
  canDelete: boolean;
  transitions: { key: string; label: string }[]; // transitions allowed & valid from the current status
  view?: string;
}) {
  const [data, setData] = useState<Data>(() => {
    // New record: fill field defaults (the server does too; this makes them visible in the form).
    if (!entry)
      return Object.fromEntries([
        ...ct.fields.filter((f) => f.default !== undefined).map((f) => [f.name, f.default]),
        ...ct.fields
          .filter((f) => f.type === "relation" && !f.multiple && f.default === undefined)
          .map((f) => [f.name, f.target === "master-site" ? placement?.defaultSite : f.target === "master-organization" ? placement?.organization : undefined])
          .filter(([, v]) => v),
        ...ct.fields
          .filter((f) => f.autoFill && f.autoFill.source !== "relation" && f.autoFill.source !== "pattern")
          .map((f) => [f.name, single(f, f.autoFill!.source === "today" ? new Date().toISOString().slice(0, 10) : f.autoFill!.source === "user" ? (f.type === "relation" ? (f.multiple ? [placement?.userId] : placement?.userId) : placement?.userName) : placement?.profile?.[f.autoFill!.field!])])
          .filter(([, v]) => v !== undefined && v !== null && v !== ""),
        ...ct.fields
          .filter((f) => f.type === "domain")
          .map((f) => [f.name, domains[f.domain!]?.find((v) => v.isDefault)?.valueId])
          .filter(([, v]) => v !== undefined),
      ]);
    const d = { ...entry };
    META.forEach((k) => delete d[k]);
    return d;
  });
  const [error, setError] = useState<string>();
  const setField = (name: string, v: unknown) => setData((cur) => ({ ...cur, [name]: v }));
  const [pending, start] = useTransition();
  // logged-in user as a profile relation option (profile may not exist / be readable): label = their name
  const me = placement?.userId ? { id: placement.userId, label: placement.userName ?? placement.userId } : undefined;
  const ctx = { components, options, domains, form: data, me };

  const locked = !!entry && !editable(ct, entry.workflowStatus);
  const writable = canSave && !locked;
  const state = stateOf(ct, entry?.workflowStatus);

  // Live auto fill: on every input change the server computes auto fields (relation, profile, patterned code)
  // with the same rules as Save. Locked = always overwritten; unlocked = only when empty.
  const autoFields = ct.fields.filter((f) => f.autoFill);
  const input = JSON.stringify(Object.fromEntries(Object.entries(data).filter(([k]) => !autoFields.some((f) => f.name === k && f.autoFill!.lock))));
  useEffect(() => {
    if (!writable || autoFields.length === 0) return;
    const ac = new AbortController();
    const t = setTimeout(() => {
      fetch(`/api/dynamic/content-types/${ct.uid}/autofill`, {
        method: "POST", signal: ac.signal, headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: entry?.id, data: JSON.parse(input) }),
      })
        .then((r) => (r.ok ? r.json() : null))
        .then((res: Data | null) => {
          if (!res) return;
          setData((cur) => {
            const out = { ...cur };
            for (const f of autoFields) if (f.autoFill!.lock || out[f.name] == null || out[f.name] === "") out[f.name] = res[f.name] ?? undefined;
            return out;
          });
        })
        .catch(() => {}); // preview only; Save is still computed by the server
    }, 250);
    return () => { clearTimeout(t); ac.abort(); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [input, writable, ct.uid, entry?.id]);

  const submit = () =>
    start(async () => setError((await (saveAction ? saveAction(JSON.stringify(data)) : saveEntry(ct.uid, entry?.id, JSON.stringify(data))))?.error));
  const remove = async () => {
    if (!entry || !(await confirmDialog({ title: "Delete this record?", message: "Deleted records cannot be restored.", confirmText: "Delete", danger: true }))) return;
    start(async () => setError((await deleteEntry(ct.uid, entry.id))?.error));
  };
  const [asking, setAsking] = useState<Transition>(); // transition with inputs: open the modal form
  const transit = async (t: { key: string; label: string }) => {
    if (!entry) return;
    const full = ct.workflow?.transitions.find((x) => x.key === t.key);
    if (full?.fields?.length) return setAsking(full);
    const note = await promptDialog({ title: t.label, message: "The note (optional) will be recorded in the status history.", placeholder: "Note", multiline: true, confirmText: t.label });
    if (note === null) return;
    start(async () => setError((await runTransition(ct.uid, entry.id, t.key, note, view))?.error));
  };

  // Tabs: one per layout section + History. One section without history = no tabs.
  const sections = effectiveLayout(ct).sections;
  const tabs = [
    ...sections.map((x) => ({ key: `s:${x.id}`, label: x.title })),
    ...(entry ? [{ key: "history", label: `History (${history.length})` }] : []),
  ];
  const [tab, setTab] = useState(tabs[0]?.key);
  const tabbed = tabs.length > 1;

  return (
    <>
    <form
      action={submit}
      // empty required field in another tab: switch to it so the browser can point to it
      onInvalidCapture={(e) => {
        const t = (e.target as HTMLElement).closest<HTMLElement>("[data-tab]")?.dataset.tab;
        if (t && t !== tab) flushSync(() => setTab(t));
      }}
    >
      <div className="sticky top-16 z-10 mx-8 mt-4 flex items-center justify-between rounded-2xl border border-slate-200/80 bg-white/90 px-5 py-3 backdrop-blur">
        {back ? (
          <a href={back.href} className="flex items-center gap-2"><ArrowLeft className="size-5" /> {back.label}</a>
        ) : (
          <Link href={view ? `/c/${ct.uid}?view=${view}` : `/c/${ct.uid}`} className="flex items-center gap-2"><ArrowLeft className="size-5" /> Back</Link>
        )}
        <div className="flex items-center gap-2">
          {state && <Badge color={state.color} className="mr-2">{state.label}</Badge>}
          {transitions.map((t) => (
            <Button key={t.key} type="button" variant="outline" onClick={() => transit(t)} disabled={pending}>{t.label}</Button>
          ))}
          {entry && canDelete && !locked && <Button type="button" variant="ghost" className="text-destructive" onClick={remove} disabled={pending}><Trash2 /> Delete</Button>}
          {writable && <Button disabled={pending}><Save /> {pending ? "Saving..." : "Save"}</Button>}
        </div>
      </div>
      <div className="flex flex-col gap-4 px-8 py-4">
        {error && <Alert>{error}</Alert>}
        {locked && <Alert className="border-primary/30 bg-accent text-foreground">This record is {state?.label ?? entry?.workflowStatus} and locked.</Alert>}
        {tabbed && (
          <div role="tablist" aria-label="Form sections" className="flex flex-wrap gap-1 rounded-xl border border-slate-200 bg-white p-1">
            {tabs.map((t) => (
              <button key={t.key} type="button" role="tab" aria-selected={tab === t.key} onClick={() => setTab(t.key)}
                className={"rounded-lg px-4 py-2 text-sm font-semibold transition " + (tab === t.key ? "bg-primary text-white" : "text-slate-600 hover:bg-slate-100")}>
                {t.label}
              </button>
            ))}
          </div>
        )}
        <fieldset disabled={!writable} className="flex flex-col gap-4">
          {sections.map((s) => (
            <Card key={s.id} data-tab={`s:${s.id}`} hidden={tabbed && tab !== `s:${s.id}`}>
              {!tabbed && <h2 className="border-b pb-4 text-lg font-bold">{s.title}</h2>}
              {s.rows.map((r, ri) => (
                <div key={ri} className="grid gap-x-4 gap-y-5" style={{ gridTemplateColumns: `repeat(${r.columns}, minmax(0, 1fr))` }}>
                  {Array.from({ length: r.columns }, (_, ci) => {
                    const f = ct.fields.find((x) => x.name === r.fields[ci]);
                    return (
                      <div key={ci}>
                        {f &&
                          (f.autoFill?.lock ? (
                            // Locked: always computed by the server, display only.
                            <FormField label={`${f.label} (auto)`}>
                              <Input value={shown(f, data[f.name], ctx)} readOnly tabIndex={-1} className="bg-slate-50 text-slate-600" placeholder="filled automatically" />
                            </FormField>
                          ) : (
                            <FieldInput f={f.autoFill?.source === "pattern" ? { ...f, required: false } : f} value={data[f.name]} onChange={(v) => setField(f.name, v)} ctx={ctx} />
                          ))}
                      </div>
                    );
                  })}
                </div>
              ))}
            </Card>
          ))}
        </fieldset>
        {entry && <div hidden={tabbed && tab !== "history"}><HistoryCard ct={ct} entry={entry} history={history} ctx={ctx} /></div>}
      </div>
    </form>
    {asking && entry && (
      <TransitionDialog t={asking} ctx={ctx} pending={pending} error={error} onCancel={() => setAsking(undefined)}
        onSubmit={(note, input) => start(async () => {
          const res = await runTransition(ct.uid, entry.id, asking.key, note, view, input);
          setError(res?.error);
          if (!res?.error) setAsking(undefined);
        })} />
    )}
    </>
  );
}

function Fields({ fields, value, onChange, ctx }: { fields: Field[]; value: Data; onChange: (d: Data) => void; ctx: Ctx }) {
  return (
    <div className="grid gap-x-4 gap-y-5 md:grid-cols-2">
      {fields.map((f) => (
        <div key={f.name} className={f.type === "component" || f.type === "longtext" ? "md:col-span-2" : undefined}>
          <FieldInput f={f} value={value[f.name]} onChange={(v) => onChange({ ...value, [f.name]: v })} ctx={ctx} />
        </div>
      ))}
    </div>
  );
}

function FieldInput({ f, value, onChange, ctx }: { f: Field; value: unknown; onChange: (v: unknown) => void; ctx: Ctx }) {
  const str = value === undefined || value === null ? "" : String(value);
  const text = (e: { target: { value: string } }) => onChange(e.target.value === "" ? null : e.target.value);

  switch (f.type) {
    case "boolean":
      return (
        <label className="flex items-center gap-2 pt-8 font-bold">
          <Checkbox checked={value === true} onChange={(e) => onChange(e.target.checked)}
            label={<span className="text-sm font-medium text-slate-700">{f.label}{f.required && <span className="text-destructive ml-0.5">*</span>}</span>} />
        </label>
      );
    case "component":
      return <ComponentInput f={f} value={value} onChange={onChange} ctx={ctx} />;
    case "relation":
      return <RelationInput f={f} value={value} onChange={onChange} ctx={ctx} />;
    case "attributes":
      return <AttributesInput f={f} value={value} onChange={onChange} ctx={ctx} />;
    case "media":
      return (
        <FormField label={f.label} required={f.required} optional={!f.required}>
          <MediaInput multiple={!!f.multiple} value={value} onChange={onChange} />
        </FormField>
      );
  }

  return (
    <FormField label={f.label} required={f.required} optional={!f.required}>
      {f.type === "longtext" ? (
        <Textarea value={str} required={f.required} onChange={text} />
      ) : f.type === "number" ? (
        <Input type="number" step="any" value={str} required={f.required} onChange={(e) => onChange(e.target.value === "" ? null : Number(e.target.value))} />
      ) : f.type === "date" ? (
        <Input type="date" value={str} required={f.required} onChange={text} />
      ) : f.type === "select" ? (
        <Combobox value={str} required={f.required} onChange={(v) => onChange(v || null)} aria-label={f.label}
          options={(f.options ?? []).map((o) => ({ value: o, label: o }))} />
      ) : f.type === "domain" ? (
        <Combobox value={str} required={f.required} onChange={(v) => onChange(v || null)} aria-label={f.label}
          options={[
            // keep showing an old value deactivated in the domain so it does not silently disappear
            ...(str && !(ctx.domains[f.domain!] ?? []).some((v) => v.valueId === str) ? [{ value: str, label: `${str} (inactive)` }] : []),
            ...(ctx.domains[f.domain!] ?? []).map((v) => ({ value: v.valueId, label: v.valueDescription, hint: v.valueId })),
          ]} />
      ) : (
        <Input value={str} required={f.required} maxLength={255} onChange={text} />
      )}
    </FormField>
  );
}

// Relation; with a filter (where), options are refetched from the server whenever a referenced form field changes,
// and selections that no longer pass the filter are cleared.
/* ponytail: at most the first 100 options from the server; switch to server-side search if relation data grows large */
function RelationInput({ f, value, onChange, ctx }: { f: Field; value: unknown; onChange: (v: unknown) => void; ctx: Ctx }) {
  const where = JSON.stringify(resolveWhere(f.where, ctx.form));
  const [filtered, setFiltered] = useState<Option[]>();
  useEffect(() => {
    if (!f.where?.length) return;
    const qs = new URLSearchParams(Object.entries(JSON.parse(where) as Record<string, string>).map(([k, v]) => [`where[${k}]`, v]));
    const ac = new AbortController();
    fetch(`/api/dynamic/content-types/${f.target}/options?${qs}`, { signal: ac.signal })
      .then((r) => (r.ok ? r.json() : []))
      .then((opts: Option[]) => {
        setFiltered(opts);
        if (opts.length >= 100) return; // truncated list: do not clear selections that may be beyond the first 100
        const ok = (id: unknown) => opts.some((o) => o.id === id);
        if (f.multiple && Array.isArray(value) && value.some((v) => !ok(v))) onChange(value.filter(ok));
        else if (!f.multiple && value && !ok(value)) onChange(null);
      })
      .catch(() => {});
    return () => ac.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [where, f.target]);
  const base = filtered ?? ctx.options[f.target!] ?? [];
  const opts = [...(f.target === "user-profile" && ctx.me && !base.some((o) => o.id === ctx.me!.id) ? [ctx.me] : []), ...base].map((o) => ({ value: o.id, label: o.label }));
  return (
    <FormField label={f.label} required={f.required} optional={!f.required}>
      {f.multiple ? (
        <Combobox multiple value={(value as string[]) ?? []} required={f.required} onChange={(v) => onChange(v)} aria-label={f.label} options={opts} />
      ) : (
        <Combobox value={value == null ? "" : String(value)} required={f.required} onChange={(v) => onChange(v || null)} aria-label={f.label} options={opts} />
      )}
    </FormField>
  );
}

// Classification attributes: inputs follow the selected classification (including inherited from parents).
function AttributesInput({ f, value, onChange, ctx }: { f: Field; value: unknown; onChange: (v: unknown) => void; ctx: Ctx }) {
  const classId = ctx.form[f.from!] as string | undefined;
  const [defs, setDefs] = useState<AttributeDef[]>();
  useEffect(() => {
    if (!classId) return setDefs(undefined);
    const ac = new AbortController();
    fetch(`/api/dynamic/classifications/${encodeURIComponent(classId)}/attributes`, { signal: ac.signal })
      .then((r) => (r.ok ? r.json() : []))
      .then(setDefs)
      .catch(() => {});
    return () => ac.abort();
  }, [classId]);
  const vals = (value ?? {}) as Record<string, unknown>;
  const set = (id: string, v: unknown) => onChange({ ...vals, [id]: v === "" || v === null ? undefined : v });
  return (
    <fieldset className="grid gap-3">
      <legend className="mb-2 text-sm font-semibold text-slate-700">{f.label}</legend>
      {!classId ? (
        <p className="text-sm text-slate-500">Select a classification first to show its attributes.</p>
      ) : !defs ? (
        <p className="text-sm text-slate-500">Loading attributes…</p>
      ) : defs.length === 0 ? (
        <p className="text-sm text-slate-500">This classification has no attributes yet.</p>
      ) : (
        <div className="grid gap-x-4 gap-y-5 md:grid-cols-3">
          {defs.map((d) => {
            const label = `${d.label}${d.unit ? ` (${d.unit})` : ""}`;
            const v = vals[d.id];
            if (d.dataType === "BOOLEAN")
              return (
                <label key={d.id} className="flex items-center gap-2 pt-8">
                  <Checkbox checked={v === true} onChange={(e) => set(d.id, e.target.checked)} label={<span className="text-sm font-medium text-slate-700">{label}</span>} />
                </label>
              );
            return (
              <FormField key={d.id} label={label} required={d.mandatory} optional={!d.mandatory}>
                {d.values ? (
                  <Combobox value={v == null ? "" : String(v)} required={d.mandatory} aria-label={d.label} onChange={(x) => set(d.id, x || null)}
                    options={d.values.map((o) => ({ value: o.valueId, label: o.valueDescription, hint: o.valueId }))} />
                ) : d.dataType === "NUMERIC" ? (
                  <Input type="number" step="any" value={v == null ? "" : String(v)} required={d.mandatory} onChange={(e) => set(d.id, e.target.value === "" ? null : Number(e.target.value))} />
                ) : d.dataType === "DATE" ? (
                  <Input type="date" value={v == null ? "" : String(v)} required={d.mandatory} onChange={(e) => set(d.id, e.target.value)} />
                ) : (
                  <Input value={v == null ? "" : String(v)} required={d.mandatory} maxLength={d.length || 255} onChange={(e) => set(d.id, e.target.value)} />
                )}
              </FormField>
            );
          })}
        </div>
      )}
    </fieldset>
  );
}

// Transition input modal (e.g. Submit requires an attachment upload + its type). Validated again on the server.
function TransitionDialog({ t, ctx, pending, error, onCancel, onSubmit }: {
  t: Transition; ctx: Ctx; pending: boolean; error?: string; onCancel: () => void; onSubmit: (note: string, data: Data) => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const [input, setInput] = useState<Data>({});
  const [note, setNote] = useState("");
  useEffect(() => { ref.current?.showModal(); }, []);
  return (
    <dialog ref={ref} aria-label={t.label} onCancel={(e) => { e.preventDefault(); onCancel(); }}
      className="m-auto w-[min(40rem,calc(100vw-2rem))] rounded-xl border bg-white p-0 text-foreground shadow-xl backdrop:bg-black/50">
      <form method="dialog" className="flex max-h-[85vh] flex-col gap-4 overflow-y-auto p-6" onSubmit={(e) => { e.preventDefault(); onSubmit(note, input); }}>
        <h2 className="text-lg font-bold">{t.label}</h2>
        {error && <Alert>{error}</Alert>}
        <Fields fields={t.fields ?? []} value={input} onChange={setInput} ctx={{ ...ctx, form: input }} />
        <FormField label="Note" optional>
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} placeholder="Recorded in the status history" />
        </FormField>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={onCancel}>Cancel</Button>
          <Button type="submit" disabled={pending}>{t.label}</Button>
        </div>
      </form>
    </dialog>
  );
}

// File links (original names) for history.
function MediaLinks({ value }: { value: unknown }) {
  const ids = (Array.isArray(value) ? value : [value]).map(String);
  const [meta, setMeta] = useState<Record<string, Media>>({});
  const key = ids.join(",");
  useEffect(() => {
    fetch(`/api/dynamic/media?ids=${ids.map(encodeURIComponent).join(",")}`).then((r) => (r.ok ? r.json() : []))
      .then((list: Media[]) => setMeta(Object.fromEntries(list.map((m) => [m.id, m])))).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);
  return <>{ids.map((id, i) => <span key={id}>{i > 0 && ", "}<a href={`/api/dynamic/media/${id}`} target="_blank" rel="noreferrer" className="text-primary underline">{meta[id]?.name ?? "file"}</a></span>)}</>;
}

function ComponentInput({ f, value, onChange, ctx }: { f: Field; value: unknown; onChange: (v: unknown) => void; ctx: Ctx }) {
  const comp = ctx.components[f.component!];
  if (!comp) return <Alert>Component {f.component} not found.</Alert>;
  const title = (
    <legend className="px-1 text-base font-bold">
      {f.label} <span className="text-muted-foreground font-normal">({comp.label}{f.repeatable ? ", repeatable" : ""})</span>
      {f.required && <span className="text-destructive ml-1">*</span>}
    </legend>
  );

  if (!f.repeatable) {
    return (
      <fieldset className="rounded-lg border p-4">
        {title}
        <Fields fields={comp.fields} value={(value as Data) ?? {}} onChange={onChange} ctx={ctx} />
      </fieldset>
    );
  }
  const list = (value as Data[]) ?? [];
  return (
    <fieldset className="flex flex-col gap-3 rounded-lg border p-4">
      {title}
      {list.map((item, i) => (
        <div key={i} className="relative rounded-md border border-dashed p-4 pt-8">
          <Button type="button" variant="ghost" aria-label="Delete item" className="text-destructive absolute top-1 right-1" onClick={() => onChange(list.filter((_, j) => j !== i))}>
            <X />
          </Button>
          <Fields fields={comp.fields} value={item} onChange={(d) => onChange(list.map((x, j) => (j === i ? d : x)))} ctx={ctx} />
        </div>
      ))}
      <Button type="button" variant="outline" className="w-fit" onClick={() => onChange([...list, {}])}><Plus /> Add {comp.label}</Button>
    </fieldset>
  );
}

// Upload directly to /api/dynamic/media (via the shell rewrite, auth from cookie). Value = file id.
function MediaInput({ multiple, value, onChange }: { multiple: boolean; value: unknown; onChange: (v: unknown) => void }) {
  const ids = (multiple ? (value as string[]) : value ? [value as string] : []) ?? [];
  const [meta, setMeta] = useState<Record<string, Media>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();

  useEffect(() => {
    const missing = ids.filter((id) => !meta[id]);
    if (missing.length === 0) return;
    fetch(`/api/dynamic/media?ids=${missing.map(encodeURIComponent).join(",")}`)
      .then((r) => (r.ok ? r.json() : []))
      .then((list: Media[]) => setMeta((m) => ({ ...m, ...Object.fromEntries(list.map((x) => [x.id, x])) })));
  }, [ids.join(",")]); // eslint-disable-line react-hooks/exhaustive-deps

  const upload = async (files: FileList | null) => {
    if (!files?.length) return;
    setBusy(true);
    setError(undefined);
    const uploaded: Media[] = [];
    for (const file of multiple ? [...files] : [files[0]]) {
      const body = new FormData();
      body.append("file", file);
      const res = await fetch("/api/dynamic/media", { method: "POST", body });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(json.error ?? "Upload failed");
        break;
      }
      uploaded.push(json);
    }
    setMeta((m) => ({ ...m, ...Object.fromEntries(uploaded.map((x) => [x.id, x])) }));
    if (uploaded.length) onChange(multiple ? [...ids, ...uploaded.map((u) => u.id)] : uploaded[0].id);
    setBusy(false);
  };

  return (
    <div className="flex flex-col gap-2 font-normal">
      {ids.map((id) => (
        <div key={id} className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm">
          <FileText className="size-4 shrink-0" />
          <a href={`/api/dynamic/media/${id}`} target="_blank" rel="noreferrer" className="text-primary truncate underline">{meta[id]?.name ?? id}</a>
          <button type="button" aria-label="Delete file" className="text-destructive ml-auto" onClick={() => onChange(multiple ? ids.filter((x) => x !== id) : null)}>
            <X className="size-4" />
          </button>
        </div>
      ))}
      {(multiple || ids.length === 0) && (
        <label className="border-primary text-primary flex h-10 w-fit cursor-pointer items-center gap-2 rounded-md border px-3 text-sm font-medium">
          <Upload className="size-4" /> {busy ? "Uploading..." : multiple ? "Upload file" : "Choose file"}
          <input type="file" multiple={multiple} className="sr-only" disabled={busy} onChange={(e) => { upload(e.target.files); e.target.value = ""; }} />
        </label>
      )}
      {error && <Alert>{error}</Alert>}
    </div>
  );
}

const ACTION: Record<Revision["action"], { label: string; color: "green" | "blue" | "red" | "yellow" }> = {
  create: { label: "Created", color: "green" },
  update: { label: "Updated", color: "blue" },
  delete: { label: "Deleted", color: "red" },
  transition: { label: "Status changed", color: "yellow" },
};
const when = (s?: string) => (s ? new Date(s).toLocaleString("en-GB", { dateStyle: "medium", timeStyle: "short" }) : "–");

// Audit trail: who, when, which field changed from what to what (create/update/delete/transition).
function HistoryCard({ ct, entry, history, ctx }: { ct: ContentType; entry: Entry; history: Revision[]; ctx: Ctx }) {
  const fieldLabel = (f: string) => (f === "workflowStatus" ? "Status" : (ct.fields.find((x) => x.name === f)?.label ?? f));
  // classification attribute names (attributes-type fields) for history
  const [attrs, setAttrs] = useState<Record<string, string>>({});
  const classIds = ct.fields.filter((f) => f.type === "attributes").map((f) => entry[f.from!]).filter(Boolean).join(",");
  useEffect(() => {
    for (const id of classIds ? classIds.split(",") : [])
      fetch(`/api/dynamic/classifications/${encodeURIComponent(id)}/attributes`).then((r) => (r.ok ? r.json() : []))
        .then((defs: AttributeDef[]) => setAttrs((a) => ({ ...a, ...Object.fromEntries(defs.map((d) => [d.id, d.label])) }))).catch(() => {});
  }, [classIds]);
  const order = (f: string) => ct.fields.findIndex((x) => x.name === f); // status (-1) first, then schema field order
  const lk = { ...ctx, attrs, state: (k: string) => stateOf(ct, k)?.label ?? k };
  const created = history.find((h) => h.action === "create");
  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b pb-4">
        <h2 className="text-lg font-bold">Change History</h2>
        <p className="text-sm text-slate-500">
          Version {entry.version ?? "–"} · created {created?.byName ? `by ${created.byName} ` : ""}{when(entry.createdAt)} · updated {when(entry.updatedAt)}
        </p>
      </div>
      {history.length === 0 && <p className="text-sm text-slate-500">No history yet (record was created before the audit trail was enabled).</p>}
      <ol className="flex flex-col">
        {[...history].reverse().map((h) => (
          <li key={h.version} className="grid grid-cols-[9.5rem_1fr] gap-4 border-b border-slate-100 py-3 last:border-0">
            <div className="flex flex-col gap-1 text-sm">
              <Badge color={ACTION[h.action].color} className="w-fit">{ACTION[h.action].label}</Badge>
              <span className="text-slate-500">{when(h.at)}</span>
            </div>
            <div className="flex flex-col gap-1 text-sm">
              <span><span className="font-semibold text-slate-900">{h.byName || h.by}</span> <span className="text-slate-400">· v{h.version}</span></span>
              {h.changes.length > 0 && (
                <ul className="flex flex-col gap-0.5 text-slate-600">
                  {[...h.changes].sort((a, b) => order(a.field) - order(b.field)).map((c) => {
                    const d = describeChange(ct.fields.find((x) => x.name === c.field), c.from, c.to, lk, h.action === "create");
                    return (
                      <li key={c.field}>
                        <span className="font-medium text-slate-700">{fieldLabel(c.field)}</span>
                        {d.inline !== undefined ? <>: {d.inline}</> : (
                          <ul className="ml-4 list-disc text-slate-600">{d.lines!.map((l, i) => <li key={i}>{l}</li>)}</ul>
                        )}
                      </li>
                    );
                  })}
                </ul>
              )}
              {h.data && (() => {
                const tf = ct.workflow?.transitions.find((x) => x.key === h.transition)?.fields ?? [];
                return (
                  <ul className="flex flex-col gap-0.5 text-slate-600">
                    {Object.entries(h.data).sort(([a], [b]) => tf.findIndex((x) => x.name === a) - tf.findIndex((x) => x.name === b)).map(([k, v]) => {
                      const f = tf.find((x) => x.name === k);
                      const d = describeChange(f, undefined, v, lk, true);
                      return (
                        <li key={k}>
                          <span className="font-medium text-slate-700">{f?.label ?? k}</span>
                          {d.inline !== undefined ? <>: {f?.type === "media" ? <MediaLinks value={v} /> : d.inline}</> : <ul className="ml-4 list-disc">{d.lines!.map((l, i) => <li key={i}>{l}</li>)}</ul>}
                        </li>
                      );
                    })}
                  </ul>
                );
              })()}
              {h.note && <span className="text-slate-500">“{h.note}”</span>}
            </div>
          </li>
        ))}
      </ol>
    </Card>
  );
}

// List (e.g. profile "sites") into a single field: take the first one (same as server).
const single = (f: Field, v: unknown) => (!f.multiple && Array.isArray(v) ? v[0] : v);

// Display value of a locked field: relations & domains show labels, not IDs.
function shown(f: Field, v: unknown, ctx: Ctx): string {
  if (v == null) return "";
  const ids = Array.isArray(v) ? v.map(String) : [String(v)];
  if (f.type === "relation") return ids.map((id) => ctx.options[f.target!]?.find((o) => o.id === id)?.label || (id === ctx.me?.id ? ctx.me.label : id)).join(", ");
  if (f.type === "domain") return ids.map((id) => ctx.domains[f.domain!]?.find((d) => d.valueId === id)?.valueDescription ?? id).join(", ");
  return ids.join(", ");
}
