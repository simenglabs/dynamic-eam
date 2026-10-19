// Operasi drag & drop layout (murni, tidak memutasi input). Diuji di layoutOps.test.ts.
import type { Layout } from "./types";

export type Cell = { si: number; ri: number; ci: number };
export type RowPos = { si: number; ri: number };

const clone = (l: Layout): Layout => structuredClone(l);
const cellsOf = (columns: number, fields: string[]) => Array.from({ length: columns }, (_, k) => fields[k] ?? "");

function setCell(l: Layout, c: Cell, name: string) {
  const row = l.sections[c.si].rows[c.ri];
  row.fields = cellsOf(row.columns, row.fields);
  row.fields[c.ci] = name;
}

// placeField: taruh field di sel `to`. Dari sel lain = tukar isi; dari baki (from undefined) = isi lama kembali ke baki.
export function placeField(layout: Layout, name: string, from: Cell | undefined, to: Cell): Layout {
  const l = clone(layout);
  const prev = l.sections[to.si].rows[to.ri].fields[to.ci] ?? "";
  if (from) setCell(l, from, prev);
  setCell(l, to, name);
  return l;
}

export function unplaceField(layout: Layout, from: Cell): Layout {
  const l = clone(layout);
  setCell(l, from, "");
  return l;
}

// moveRow: pindah baris ke posisi sebelum baris `to` (boleh lintas section). to.ri = jumlah baris = taruh di akhir.
export function moveRow(layout: Layout, from: RowPos, to: RowPos): Layout {
  const l = clone(layout);
  const [row] = l.sections[from.si].rows.splice(from.ri, 1);
  const at = from.si === to.si && from.ri < to.ri ? to.ri - 1 : to.ri;
  l.sections[to.si].rows.splice(at, 0, row);
  return l;
}

export function moveSection(layout: Layout, from: number, to: number): Layout {
  const l = clone(layout);
  const [s] = l.sections.splice(from, 1);
  l.sections.splice(to, 0, s);
  return l;
}
