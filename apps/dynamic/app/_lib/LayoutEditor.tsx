"use client";

import { useState, type DragEvent } from "react";
import { ArrowDown, ArrowUp, GripVertical, Plus, X } from "lucide-react";
import { Button, Card, Input } from "@eam/ui/ui";
import { MAX_COLUMNS, effectiveLayout, type Field, type Layout, type Section } from "./types";
import { moveRow, moveSection, placeField, unplaceField } from "./layoutOps";
import { Combobox } from "@eam/ui/combobox";

// Drag payload (native HTML5 drag & drop, no library). Dropdowns & arrow buttons remain for keyboard users.
type Drag =
  | { kind: "field"; name: string; from?: { si: number; ri: number; ci: number } }
  | { kind: "row"; si: number; ri: number }
  | { kind: "section"; si: number };

const MIME = "application/x-eam-layout";

const swap = <T,>(xs: T[], i: number, j: number) => {
  const next = [...xs];
  [next[i], next[j]] = [next[j], next[i]];
  return next;
};
const slug = (s: string) => s.toLowerCase().trim().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 48) || "section";
const cellsOf = (columns: number, fields: string[]) => Array.from({ length: columns }, (_, k) => fields[k] ?? "");

// Form layout editor: sections (order & title) -> rows (column count) -> field per column (left to right).
// value null = default layout (all fields, 2 columns).
export function LayoutEditor({ fields, value, onChange }: { fields: Field[]; value: Layout | null; onChange: (l: Layout | null) => void }) {
  const [over, setOver] = useState<string>(); // id of the hovered drop target
  const named = fields.filter((f) => f.name);

  if (!value) {
    return (
      <Card>
        <h2 className="border-b pb-4 text-lg font-bold">Layout Form</h2>
        <p className="text-muted-foreground">Default: one section, 2 fields per row in field order.</p>
        <Button type="button" variant="outline" className="w-fit" onClick={() => onChange(effectiveLayout({ fields: named, layout: null }))}>
          Customize layout
        </Button>
      </Card>
    );
  }

  const placed = new Set(value.sections.flatMap((s) => s.rows.flatMap((r) => r.fields)).filter(Boolean));
  const unplaced = named.filter((f) => !placed.has(f.name));
  const labelOf = (n: string) => named.find((f) => f.name === n)?.label || n;
  const setSections = (sections: Section[]) => onChange({ sections });

  // ---------- drag & drop ----------
  const start = (d: Drag) => (e: DragEvent) => {
    e.dataTransfer.setData(MIME, JSON.stringify(d));
    e.dataTransfer.effectAllowed = "move";
    e.stopPropagation();
  };
  const read = (e: DragEvent): Drag | null => {
    try {
      return JSON.parse(e.dataTransfer.getData(MIME));
    } catch {
      return null;
    }
  };
  // Targets accept only certain drag kinds; dataTransfer.types is readable during dragover (its data is not).
  const target = (id: string, onDrop: (d: Drag) => void) => ({
    onDragOver: (e: DragEvent) => {
      if (!e.dataTransfer.types.includes(MIME)) return;
      e.preventDefault();
      e.stopPropagation();
      setOver(id);
    },
    onDragLeave: () => setOver((o) => (o === id ? undefined : o)),
    onDrop: (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setOver(undefined);
      const d = read(e);
      if (d) onDrop(d);
    },
  });
  const ring = (id: string) => (over === id ? " ring-2 ring-primary ring-offset-1" : "");

  const dropOnCell = (si: number, ri: number, ci: number) => (d: Drag) => {
    if (d.kind === "field") onChange(placeField(value, d.name, d.from, { si, ri, ci }));
  };
  const dropOnTray = (d: Drag) => {
    if (d.kind === "field" && d.from) onChange(unplaceField(value, d.from));
  };
  const dropRow = (si: number, ri: number) => (d: Drag) => {
    if (d.kind === "row") onChange(moveRow(value, d, { si, ri }));
  };
  const dropSection = (si: number) => (d: Drag) => {
    if (d.kind === "section" && d.si !== si) onChange(moveSection(value, d.si, si));
  };

  // ---------- regular editing ----------
  const setSection = (si: number, s: Section) => setSections(value.sections.map((x, j) => (j === si ? s : x)));
  const setCell = (si: number, ri: number, ci: number, name: string) =>
    onChange(name ? placeField(value, name, undefined, { si, ri, ci }) : unplaceField(value, { si, ri, ci }));
  const addSection = () => {
    const title = `Section ${value.sections.length + 1}`;
    let id = slug(title);
    while (value.sections.some((s) => s.id === id)) id += "-x";
    setSections([...value.sections, { id, title, rows: [{ columns: 2, fields: [] }] }]);
  };

  const chip = "flex min-h-9 cursor-grab items-center gap-1 rounded-md border bg-white px-2 text-sm font-medium shadow-xs active:cursor-grabbing";

  return (
    <Card>
      <div className="flex items-center justify-between border-b pb-4">
        <div>
          <h2 className="text-lg font-bold">Layout Form</h2>
          <p className="text-muted-foreground text-sm">Drag fields into columns, drag ⠿ to move rows/sections. The number of columns per row can be changed.</p>
        </div>
        <div className="flex gap-2">
          <Button type="button" variant="ghost" onClick={() => onChange(null)}>Use default</Button>
          <Button type="button" variant="outline" onClick={addSection}><Plus /> Section</Button>
        </div>
      </div>

      {/* Tray: unplaced fields. Dropping a field here from a column removes it from the layout. */}
      <div {...target("tray", dropOnTray)} className={"flex min-h-12 flex-wrap items-center gap-2 rounded-md border border-dashed bg-amber-50 p-2" + ring("tray")}>
        <span className="text-muted-foreground mr-1 text-sm">Not placed:</span>
        {unplaced.map((f) => (
          <span key={f.name} draggable onDragStart={start({ kind: "field", name: f.name })} className={chip}>
            <GripVertical className="text-muted-foreground size-3" /> {f.label || f.name}
          </span>
        ))}
        {unplaced.length === 0 && <span className="text-muted-foreground text-sm">all fields are placed (drag here to remove)</span>}
      </div>

      {value.sections.map((s, si) => (
        <fieldset key={s.id} {...target(`s${si}`, dropSection(si))} className={"flex flex-col gap-3 rounded-lg border p-4" + ring(`s${si}`)}>
          <div className="flex items-center gap-2">
            <span draggable onDragStart={start({ kind: "section", si })} aria-label="Drag section" title="Drag to move section" className="cursor-grab p-1">
              <GripVertical className="text-muted-foreground size-5" />
            </span>
            <Input aria-label="Section title" value={s.title} onChange={(e) => setSection(si, { ...s, title: e.target.value })} className="flex-1 font-bold" />
            <Button type="button" variant="ghost" aria-label="Move section up" disabled={si === 0} onClick={() => setSections(swap(value.sections, si, si - 1))}><ArrowUp /></Button>
            <Button type="button" variant="ghost" aria-label="Move section down" disabled={si === value.sections.length - 1} onClick={() => setSections(swap(value.sections, si, si + 1))}><ArrowDown /></Button>
            <Button type="button" variant="ghost" aria-label="Delete section" className="text-destructive" onClick={() => setSections(value.sections.filter((_, j) => j !== si))}><X /></Button>
          </div>

          {s.rows.map((r, ri) => (
            <div key={ri} {...target(`r${si}-${ri}`, dropRow(si, ri))} className={"flex items-start gap-2 rounded-md bg-slate-50 p-2" + ring(`r${si}-${ri}`)}>
              <span draggable onDragStart={start({ kind: "row", si, ri })} aria-label="Drag row" title="Drag to move row" className="cursor-grab pt-6">
                <GripVertical className="text-muted-foreground size-4" />
              </span>
              <label className="flex w-28 shrink-0 flex-col text-xs font-bold">
                Columns
                <Combobox value={String(r.columns)} required aria-label="Columns per row" onChange={(v) => {
                  const columns = Number(v);
                  setSection(si, { ...s, rows: s.rows.map((x, j) => (j === ri ? { columns, fields: x.fields.slice(0, columns) } : x)) });
                }} options={Array.from({ length: MAX_COLUMNS }, (_, i) => ({ value: String(i + 1), label: `${i + 1} per row` }))} />
              </label>
              <div className="grid flex-1 gap-2 pt-4" style={{ gridTemplateColumns: `repeat(${r.columns}, minmax(0, 1fr))` }}>
                {cellsOf(r.columns, r.fields).map((cur, ci) => (
                  <div key={ci} {...target(`c${si}-${ri}-${ci}`, dropOnCell(si, ri, ci))} className={"flex min-h-11 items-center gap-1 rounded-md border border-dashed bg-white p-1" + ring(`c${si}-${ri}-${ci}`)}>
                    {cur ? (
                      <span draggable onDragStart={start({ kind: "field", name: cur, from: { si, ri, ci } })} className={chip + " flex-1"}>
                        <GripVertical className="text-muted-foreground size-3" />
                        <span className="truncate">{labelOf(cur)}</span>
                        <button type="button" aria-label={`Remove ${labelOf(cur)}`} className="text-muted-foreground hover:text-destructive ml-auto" onClick={() => setCell(si, ri, ci, "")}>
                          <X className="size-3" />
                        </button>
                      </span>
                    ) : (
                      <Combobox aria-label={`Column ${ci + 1} field`} value="" placeholder="— drag / select field —" className="min-w-0 flex-1"
                        onChange={(v) => v && setCell(si, ri, ci, v)} options={unplaced.map((f) => ({ value: f.name, label: f.label || f.name, hint: f.name }))} />
                    )}
                  </div>
                ))}
              </div>
              <Button type="button" variant="ghost" aria-label="Delete row" className="text-destructive mt-4 px-1" onClick={() => setSection(si, { ...s, rows: s.rows.filter((_, j) => j !== ri) })}><X /></Button>
            </div>
          ))}
          {/* Drop target at the end of the section so rows can move to the bottom / into an empty section. */}
          <div {...target(`end${si}`, dropRow(si, s.rows.length))} className={"flex items-center gap-2 rounded-md" + ring(`end${si}`)}>
            <Button type="button" variant="outline" className="w-fit" onClick={() => setSection(si, { ...s, rows: [...s.rows, { columns: 2, fields: [] }] })}>
              <Plus /> Row
            </Button>
            <span className="text-muted-foreground text-xs">or drag a row here</span>
          </div>
        </fieldset>
      ))}
    </Card>
  );
}
