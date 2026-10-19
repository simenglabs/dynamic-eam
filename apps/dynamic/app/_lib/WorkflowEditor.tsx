"use client";

import { ArrowDown, ArrowUp, Plus, X } from "lucide-react";
import { MENU } from "@eam/ui/menu";
import { Badge, Button, Card, Field, Input } from "@eam/ui/ui";
import { confirmDialog } from "@eam/ui/dialog";
import { COLORS, type Field as FieldT, type State, type Workflow } from "./types";
import { Combobox } from "@eam/ui/combobox";
import { Checkbox } from "@eam/ui/checkbox";

const key = (s: string) => s.toLowerCase().trim().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 48);
const swap = <T,>(xs: T[], i: number, j: number) => {
  const next = [...xs];
  [next[i], next[j]] = [next[j], next[i]];
  return next;
};

const DEFAULT: Workflow = {
  states: [
    { key: "draft", label: "Draft", color: "neutral", editable: true },
    { key: "submitted", label: "Menunggu Approval", color: "yellow", editable: false },
    { key: "approved", label: "Disetujui", color: "green", editable: false },
    { key: "rejected", label: "Ditolak", color: "red", editable: true },
  ],
  initial: "draft",
  transitions: [
    { key: "submit", label: "Submit", from: ["draft", "rejected"], to: "submitted" },
    { key: "approve", label: "Approve", from: ["submitted"], to: "approved" },
    { key: "reject", label: "Reject", from: ["submitted"], to: "rejected" },
  ],
  views: [],
};

function StateChecks({ states, value, onChange, label }: { states: State[]; value: string[]; onChange: (v: string[]) => void; label: string }) {
  return (
    <span className="flex flex-wrap gap-3 text-sm" role="group" aria-label={label}>
      {states.map((s) => (
        <Checkbox key={s.key} checked={value.includes(s.key)} label={s.label || s.key}
          onChange={(e) => onChange(e.target.checked ? [...value, s.key] : value.filter((x) => x !== s.key))} />
      ))}
    </span>
  );
}

// Editor workflow: status (+ terkunci/tidak), status awal setelah Save, transisi (tombol aksi), dan view per status (menu).
// renderFields: editor field isian transisi (disediakan SchemaForm, memakai editor field yang sama).
export function WorkflowEditor({ value, onChange, renderFields }: {
  value: Workflow | null;
  onChange: (w: Workflow | null) => void;
  renderFields: (fields: FieldT[], onChange: (fs: FieldT[]) => void) => React.ReactNode;
}) {
  if (!value) {
    return (
      <Card>
        <h2 className="border-b pb-4 text-lg font-bold">Workflow</h2>
        <p className="text-muted-foreground">Tanpa workflow: data tidak punya status dan selalu bisa diedit.</p>
        <Button type="button" variant="outline" className="w-fit" onClick={() => onChange(structuredClone(DEFAULT))}>Aktifkan workflow (contoh approval)</Button>
      </Card>
    );
  }
  const w = value;
  const set = (patch: Partial<Workflow>) => onChange({ ...w, ...patch });
  const setState = (i: number, patch: Partial<State>) => {
    const old = w.states[i].key;
    const states = w.states.map((s, j) => (j === i ? { ...s, ...patch } : s));
    if (patch.key === undefined || patch.key === old) return set({ states });
    // Ganti key: ikut perbarui semua rujukan.
    const ren = (k: string) => (k === old ? patch.key! : k);
    set({
      states,
      initial: ren(w.initial),
      transitions: w.transitions.map((t) => ({ ...t, from: t.from.map(ren), to: ren(t.to) })),
      views: w.views.map((v) => ({ ...v, states: v.states.map(ren) })),
    });
  };

  return (
    <Card>
      <div className="flex items-center justify-between border-b pb-4">
        <div>
          <h2 className="text-lg font-bold">Workflow</h2>
          <p className="text-muted-foreground text-sm">Izin menjalankan transisi & membuka view diatur di Administration › Role.</p>
        </div>
        <Button type="button" variant="ghost" className="text-destructive" onClick={async () => (await confirmDialog({ title: "Matikan workflow?", message: "Status, transisi, dan view akan dihapus dari schema saat disimpan.", confirmText: "Matikan", danger: true })) && onChange(null)}>Matikan workflow</Button>
      </div>

      <section className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <h3 className="font-bold">Status</h3>
          <Button type="button" variant="outline" onClick={() => set({ states: [...w.states, { key: `status-${w.states.length + 1}`, label: "", color: "neutral", editable: true }] })}><Plus /> Status</Button>
        </div>
        {w.states.map((s, i) => (
          <div key={i} className="grid items-end gap-2 rounded-md bg-slate-50 p-2 md:grid-cols-[1fr_1fr_9rem_auto_auto]">
            <Field label="Label" required>
              <Input value={s.label} required onChange={(e) => setState(i, { label: e.target.value, ...(s.key.startsWith("status-") && { key: key(e.target.value) || s.key }) })} />
            </Field>
            <Field label="Key" required>
              <Input value={s.key} required pattern="[a-z][a-z0-9-]{0,47}" onChange={(e) => setState(i, { key: e.target.value })} />
            </Field>
            <Field label="Warna" required>
              <Combobox value={s.color} required aria-label="Warna" onChange={(v) => setState(i, { color: v as State["color"] })}
                options={COLORS.map((c) => ({ value: c, label: c }))} />
            </Field>
            <Checkbox className="pb-3" checked={s.editable} onChange={(e) => setState(i, { editable: e.target.checked })} label="Bisa diedit" />
            <div className="flex pb-1">
              <Badge color={s.color} className="mr-2 self-center">{s.label || s.key}</Badge>
              <Button type="button" variant="ghost" aria-label="Status naik" disabled={i === 0} onClick={() => set({ states: swap(w.states, i, i - 1) })}><ArrowUp /></Button>
              <Button type="button" variant="ghost" aria-label="Status turun" disabled={i === w.states.length - 1} onClick={() => set({ states: swap(w.states, i, i + 1) })}><ArrowDown /></Button>
              <Button type="button" variant="ghost" aria-label="Hapus status" className="text-destructive" disabled={w.states.length === 1} onClick={() => set({ states: w.states.filter((_, j) => j !== i) })}><X /></Button>
            </div>
          </div>
        ))}
        <Field label="Status awal setelah Save pertama" required>
          <Combobox value={w.initial} required aria-label="Status awal" className="w-64" onChange={(v) => set({ initial: v })} options={w.states.map((s) => ({ value: s.key, label: s.label || s.key }))} />
        </Field>
      </section>

      <section className="flex flex-col gap-2 border-t pt-4">
        <div className="flex items-center justify-between">
          <h3 className="font-bold">Transisi (tombol aksi)</h3>
          <Button type="button" variant="outline" onClick={() => set({ transitions: [...w.transitions, { key: `aksi-${w.transitions.length + 1}`, label: "", from: [], to: w.states[0].key }] })}><Plus /> Transisi</Button>
        </div>
        {w.transitions.map((t, i) => {
          const upd = (patch: Partial<typeof t>) => set({ transitions: w.transitions.map((x, j) => (j === i ? { ...x, ...patch } : x)) });
          return (
            <div key={i} className="grid items-start gap-2 rounded-md bg-slate-50 p-2 md:grid-cols-[1fr_1fr_2fr_1fr_auto]">
              <Field label="Label tombol" required>
                <Input value={t.label} required onChange={(e) => upd({ label: e.target.value, ...(t.key.startsWith("aksi-") && { key: key(e.target.value) || t.key }) })} />
              </Field>
              <Field label="Key" required>
                <Input value={t.key} required pattern="[a-z][a-z0-9-]{0,47}" onChange={(e) => upd({ key: e.target.value })} />
              </Field>
              <Field label="Dari status" required>
                <StateChecks states={w.states} value={t.from} onChange={(from) => upd({ from })} label={`${t.label} dari status`} />
              </Field>
              <Field label="Menjadi" required>
                <Combobox value={t.to} required aria-label="Menjadi" onChange={(v) => upd({ to: v })} options={w.states.map((s) => ({ value: s.key, label: s.label || s.key }))} />
              </Field>
              <Button type="button" variant="ghost" aria-label="Hapus transisi" className="text-destructive mt-7" onClick={() => set({ transitions: w.transitions.filter((_, j) => j !== i) })}><X /></Button>
              <div className="md:col-span-full">
                <details open={!!t.fields?.length} className="rounded-md border border-dashed bg-white p-3">
                  <summary className="cursor-pointer text-sm font-medium">
                    Isian saat {t.label || "transisi"} {t.fields?.length ? `(${t.fields.length} field)` : <span className="font-normal text-slate-500">— mis. wajib upload lampiran + pilih jenisnya</span>}
                  </summary>
                  <div className="mt-3 flex flex-col gap-3">{renderFields(t.fields ?? [], (fields) => upd({ fields }))}</div>
                </details>
              </div>
            </div>
          );
        })}
      </section>

      <section className="flex flex-col gap-2 border-t pt-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="font-bold">View per status (menu)</h3>
            <p className="text-muted-foreground text-sm">Contoh: menu "Approval" berisi data berstatus Menunggu Approval, ditaruh di menu mana saja.</p>
          </div>
          <Button type="button" variant="outline" onClick={() => set({ views: [...w.views, { key: `view-${w.views.length + 1}`, label: "", parent: "dynamic", states: [] }] })}><Plus /> View</Button>
        </div>
        {w.views.map((v, i) => {
          const upd = (patch: Partial<typeof v>) => set({ views: w.views.map((x, j) => (j === i ? { ...x, ...patch } : x)) });
          return (
            <div key={i} className="grid items-start gap-2 rounded-md bg-slate-50 p-2 md:grid-cols-[1fr_1fr_1fr_2fr_auto]">
              <Field label="Nama menu" required>
                <Input value={v.label} required onChange={(e) => upd({ label: e.target.value, ...(v.key.startsWith("view-") && { key: key(e.target.value) || v.key }) })} />
              </Field>
              <Field label="Key" required>
                <Input value={v.key} required pattern="[a-z][a-z0-9-]{0,47}" onChange={(e) => upd({ key: e.target.value })} />
              </Field>
              <Field label="Tampil di menu" required>
                <Combobox value={v.parent} required aria-label="Tampil di menu" onChange={(p) => upd({ parent: p })} options={MENU.map((z) => ({ value: z.zone, label: z.label }))} />
              </Field>
              <Field label="Berisi status" required>
                <StateChecks states={w.states} value={v.states} onChange={(states) => upd({ states })} label={`${v.label} berisi status`} />
              </Field>
              <Button type="button" variant="ghost" aria-label="Hapus view" className="text-destructive mt-7" onClick={() => set({ views: w.views.filter((_, j) => j !== i) })}><X /></Button>
            </div>
          );
        })}
      </section>
    </Card>
  );
}
