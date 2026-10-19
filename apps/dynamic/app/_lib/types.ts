// Sama dengan struct di services/dynamic/schema.go.
export const FIELD_TYPES = ["text", "longtext", "number", "boolean", "date", "select", "domain", "relation", "media", "component", "attributes"] as const;
export type FieldType = (typeof FIELD_TYPES)[number];

export type Field = {
  name: string;
  label: string;
  type: FieldType;
  required: boolean;
  unique?: boolean;
  default?: unknown;
  options?: string[];
  target?: string;
  multiple?: boolean;
  component?: string;
  repeatable?: boolean;
  uniqueBy?: string; // repeatable component: field yang harus unik dalam list
  domain?: string; // tipe domain: kode domain (mis. PRIORITY)
  autoFill?: AutoFill;
  where?: Condition[]; // relation: filter pilihan
  from?: string; // attributes: field relasi ke master-classification
};

// Filter relasi: field di entri target = value. value literal atau "{field}" = nilai field lain di form ini.
export type Condition = { field: string; value: string };

// Definisi atribut classification (GET /classifications/:id/attributes).
export type AttributeDef = {
  id: string; code: string; label: string; dataType: "ALN" | "NUMERIC" | "DATE" | "BOOLEAN"; unit?: string;
  domain?: string; length?: number; mandatory: boolean; values?: { valueId: string; valueDescription: string }[];
};

// Filter relasi dengan nilai dari form; ref yang masih kosong diabaikan (sama dengan server).
export function resolveWhere(where: Condition[] | undefined, form: Record<string, unknown>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const c of where ?? []) {
    const ref = /^\{(.+)\}$/.exec(c.value)?.[1];
    const v = ref ? form[ref] : c.value;
    if (v != null && v !== "") out[c.field] = String(v);
  }
  return out;
}

// Isi otomatis: relation (field dari entri relasi), profile (profil user), today, user. lock = selalu dihitung server.
// pattern: kode berpola, mis. "AST-{seq:5}" / "{classification.classificationCode}-{YYYY}-{seq:4}" (dibuat server saat Save)
export type AutoFill = { source: "relation" | "profile" | "today" | "user" | "pattern"; from?: string; field?: string; lock?: boolean; pattern?: string };

export type DomainValue = { valueId: string; valueDescription: string; isDefault: boolean };
export type DomainCode = { code: string; description: string };

// Kode domain yang dipakai field (termasuk di dalam component, rekursif).
export function domainCodes(fields: Field[], comps: Record<string, { fields: Field[] }>, seen = new Set<string>()): Set<string> {
  const out = new Set<string>();
  for (const f of fields) {
    if (f.type === "domain" && f.domain) out.add(f.domain);
    if (f.type === "component" && f.component && !seen.has(f.component)) {
      seen.add(f.component);
      domainCodes(comps[f.component]?.fields ?? [], comps, seen).forEach((d) => out.add(d));
    }
  }
  return out;
}

export type Row = { columns: number; fields: string[] }; // "" = slot kosong
export type Section = { id: string; title: string; rows: Row[] };
export type Layout = { sections: Section[] };

export const COLORS = ["neutral", "green", "red", "yellow", "blue"] as const;
export type State = { key: string; label: string; color: (typeof COLORS)[number]; editable: boolean };
// fields: isian saat menjalankan transisi (mis. lampiran + jenisnya), tercatat di riwayat.
export type Transition = { key: string; label: string; from: string[]; to: string; fields?: Field[] };
export type View = { key: string; label: string; parent: string; states: string[] };
export type Workflow = { states: State[]; initial: string; transitions: Transition[]; views: View[] };

export type ContentType = {
  uid: string;
  label: string;
  parent: string;
  hidden?: boolean;
  scopeField?: string; // field relasi ke Sites: data dibatasi per site akses user
  displayField: string;
  fields: Field[];
  layout: Layout | null;
  workflow: Workflow | null;
};

export type Component = { uid: string; label: string; fields: Field[] };

export type StatusChange = { from: string; to: string; transition: string; by: string; at: string; note: string };
export type Entry = Record<string, unknown> & { id: string; version?: number; createdAt?: string; updatedAt?: string; workflowStatus?: string; workflowHistory?: StatusChange[] };
export type Revision = {
  version: number;
  action: "create" | "update" | "delete" | "transition";
  by: string;
  byName: string;
  at: string;
  changes: { field: string; from: unknown; to: unknown }[];
  note?: string;
  transition?: string; // key transisi (action "transition")
  data?: Record<string, unknown>; // isian form transisi
};
export type Option = { id: string; label: string };
export type Media = { id: string; name: string; mime: string; size: number };

// Target relation yang dipakai content type, termasuk di dalam component (rekursif).
export function relationTargets(fields: Field[], comps: Record<string, Component>, seen = new Set<string>()): Set<string> {
  const out = new Set<string>();
  for (const f of fields) {
    if (f.type === "relation" && f.target) out.add(f.target);
    if (f.type === "component" && f.component && !seen.has(f.component)) {
      seen.add(f.component);
      relationTargets(comps[f.component]?.fields ?? [], comps, seen).forEach((t) => out.add(t));
    }
  }
  return out;
}

export const MAX_COLUMNS = 4;

// Sama dengan ContentType.EffectiveLayout di Go: tanpa layout = section "Umum" 2 kolom; field yang belum
// ditempatkan masuk section terakhir; rujukan ke field yang sudah dihapus jadi slot kosong.
export function effectiveLayout(ct: { fields: Field[]; layout: Layout | null }): Layout {
  const names = new Set(ct.fields.map((f) => f.name));
  const sections: Section[] = (ct.layout?.sections ?? []).map((s) => ({
    ...s,
    rows: s.rows.map((r) => ({ columns: r.columns, fields: r.fields.map((f) => (names.has(f) ? f : "")) })),
  }));
  const placed = new Set(sections.flatMap((s) => s.rows.flatMap((r) => r.fields)));
  const rest = ct.fields.map((f) => f.name).filter((n) => !placed.has(n));
  if (rest.length === 0) return { sections };
  if (sections.length === 0) sections.push({ id: "umum", title: "Umum", rows: [] });
  const last = sections[sections.length - 1];
  for (let i = 0; i < rest.length; i += 2) last.rows.push({ columns: 2, fields: rest.slice(i, i + 2) });
  return { sections };
}

export const stateOf = (ct: ContentType, key?: string) => ct.workflow?.states.find((s) => s.key === key);
export const editable = (ct: ContentType, status?: string) => !ct.workflow || (stateOf(ct, status)?.editable ?? true);

// Sama dengan ContentType.DisplayLabel di Go: "nomor" atau template "{nomor} - {nama}".
export function displayLabel(display: string, data: Record<string, unknown>): string {
  const val = (k: string) => (data[k] === undefined || data[k] === null ? "" : String(data[k]));
  if (!/[{}]/.test(display)) return val(display.trim());
  return display.replace(/\{([a-zA-Z][a-zA-Z0-9]*)\}/g, (_, k: string) => val(k));
}

// Field isian semua transisi workflow (untuk memuat opsi relasi & nilai domain di form).
export const transitionFields = (ct: ContentType): Field[] => (ct.workflow?.transitions ?? []).flatMap((t) => t.fields ?? []);
