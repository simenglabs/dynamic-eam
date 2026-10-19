// Format nilai riwayat perubahan agar terbaca orang awam: ID relasi -> label, kode domain -> deskripsi,
// list component -> item ditambah / dihapus / diubah, atribut -> nama atribut.
import type { DomainValue, Field } from "./types.ts";

export type Lookup = {
  options: Record<string, { id: string; label: string }[]>;
  domains: Record<string, DomainValue[]>;
  components: Record<string, { fields: Field[] }>;
  attrs?: Record<string, string>; // id atribut -> label
  state?: (key: string) => string; // label status workflow
};

const empty = (v: unknown) => v === null || v === undefined || v === "" || (Array.isArray(v) && v.length === 0);

export function showValue(f: Field | undefined, v: unknown, lk: Lookup): string {
  if (empty(v)) return "–";
  if (Array.isArray(v) && f?.type !== "component") return v.map((x) => showValue(f, x, lk)).join(", ");
  switch (f?.type) {
    case "boolean":
      return v ? "Ya" : "Tidak";
    case "number":
      return typeof v === "number" ? v.toLocaleString("id-ID") : String(v);
    case "date": {
      const d = new Date(String(v));
      return isNaN(+d) ? String(v) : d.toLocaleDateString("id-ID", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" });
    }
    case "relation":
      return lk.options[f.target!]?.find((o) => o.id === v)?.label ?? `(data ${String(v).slice(0, 6)}…)`;
    case "domain":
      return lk.domains[f.domain!]?.find((d) => d.valueId === v)?.valueDescription ?? String(v);
    case "media":
      return Array.isArray(v) ? `${v.length} file` : "1 file";
    case "component":
      return Array.isArray(v) ? `${v.length} item` : summary(lk.components[f.component!]?.fields ?? [], v as Record<string, unknown>, lk);
    case "attributes":
      return Object.entries(v as Record<string, unknown>).map(([id, x]) => `${lk.attrs?.[id] ?? "atribut"}: ${typeof x === "boolean" ? (x ? "Ya" : "Tidak") : String(x)}`).join(", ");
  }
  if (!f && lk.state) return lk.state(String(v)); // workflowStatus
  return typeof v === "object" ? JSON.stringify(v) : String(v);
}

// "Label: nilai" untuk field yang terisi.
function summary(fields: Field[], item: Record<string, unknown>, lk: Lookup): string {
  return fields.filter((f) => !empty(item?.[f.name])).map((f) => `${f.label}: ${showValue(f, item[f.name], lk)}`).join(", ");
}

// Satu perubahan field: ringkas satu baris, atau beberapa baris untuk component/atribut.
export function describeChange(f: Field | undefined, from: unknown, to: unknown, lk: Lookup, creating: boolean): { inline?: string; lines?: string[] } {
  if (f?.type === "component") {
    const fields = lk.components[f.component!]?.fields ?? [];
    if (!f.repeatable) return { lines: diffObject(fields, (from ?? {}) as Record<string, unknown>, (to ?? {}) as Record<string, unknown>, lk, creating) };
    const a = (Array.isArray(from) ? from : []) as Record<string, unknown>[];
    const b = (Array.isArray(to) ? to : []) as Record<string, unknown>[];
    if (creating) return { lines: b.map((x) => summary(fields, x, lk)) };
    const keyOf = (x: Record<string, unknown>, i: number) => (f.uniqueBy && !empty(x[f.uniqueBy]) ? String(x[f.uniqueBy]) : `#${i + 1}`);
    const title = (x: Record<string, unknown>, i: number) => {
      const k = fields.find((cf) => cf.name === f.uniqueBy) ?? fields[0];
      return k && !empty(x[k.name]) ? showValue(k, x[k.name], lk) : `Item ${i + 1}`;
    };
    const before = new Map(a.map((x, i) => [keyOf(x, i), x]));
    const after = new Map(b.map((x, i) => [keyOf(x, i), x]));
    const lines: string[] = [];
    b.forEach((x, i) => {
      const old = before.get(keyOf(x, i));
      if (!old) lines.push(`Ditambah: ${summary(fields, x, lk)}`);
      else for (const l of diffObject(fields, old, x, lk, false)) lines.push(`${title(x, i)} — ${l}`);
    });
    a.forEach((x, i) => !after.has(keyOf(x, i)) && lines.push(`Dihapus: ${title(x, i)}`));
    return { lines };
  }
  if (f?.type === "attributes") {
    const attr = (id: string) => ({ name: id, label: lk.attrs?.[id] ?? "Atribut", type: "text", required: false }) as Field;
    const a = (from ?? {}) as Record<string, unknown>, b = (to ?? {}) as Record<string, unknown>;
    const ids = [...new Set([...Object.keys(a), ...Object.keys(b)])];
    return { lines: diffObject(ids.map(attr), a, b, lk, creating) };
  }
  return { inline: creating ? showValue(f, to, lk) : `${showValue(f, from, lk)} → ${showValue(f, to, lk)}` };
}

function diffObject(fields: Field[], a: Record<string, unknown>, b: Record<string, unknown>, lk: Lookup, creating: boolean): string[] {
  return fields
    .filter((f) => JSON.stringify(a[f.name] ?? null) !== JSON.stringify(b[f.name] ?? null))
    .map((f) => (creating ? `${f.label}: ${showValue(f, b[f.name], lk)}` : `${f.label}: ${showValue(f, a[f.name], lk)} → ${showValue(f, b[f.name], lk)}`));
}
