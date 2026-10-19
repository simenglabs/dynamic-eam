// node --test apps/dynamic/app/_lib/layoutOps.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { moveRow, moveSection, placeField, unplaceField } from "./layoutOps.ts";
import type { Layout } from "./types.ts";

const base = (): Layout => ({
  sections: [
    { id: "a", title: "A", rows: [{ columns: 2, fields: ["x", "y"] }, { columns: 3, fields: ["z"] }] },
    { id: "b", title: "B", rows: [] },
  ],
});
const cells = (l: Layout) => l.sections.map((s) => s.rows.map((r) => r.fields));

test("field from tray to empty cell", () => {
  const l = placeField(base(), "w", undefined, { si: 0, ri: 1, ci: 2 });
  assert.deepEqual(l.sections[0].rows[1].fields, ["z", "", "w"]);
});

test("field from tray to filled cell: old content returns to tray", () => {
  const l = placeField(base(), "w", undefined, { si: 0, ri: 0, ci: 0 });
  assert.deepEqual(l.sections[0].rows[0].fields, ["w", "y"]);
});

test("between cells: swap", () => {
  const l = placeField(base(), "x", { si: 0, ri: 0, ci: 0 }, { si: 0, ri: 1, ci: 0 });
  assert.deepEqual(cells(l)[0], [["z", "y"], ["x", "", ""]]);
});

test("onto own cell: unchanged", () => {
  assert.deepEqual(cells(placeField(base(), "x", { si: 0, ri: 0, ci: 0 }, { si: 0, ri: 0, ci: 0 })), cells(base()));
});

test("release field to tray", () => {
  assert.deepEqual(unplaceField(base(), { si: 0, ri: 0, ci: 1 }).sections[0].rows[0].fields, ["x", ""]);
});

test("move row to another section (end) and up within the same section", () => {
  const l = moveRow(base(), { si: 0, ri: 0 }, { si: 1, ri: 0 });
  assert.deepEqual(cells(l), [[["z"]], [["x", "y"]]]);
  const up = moveRow(base(), { si: 0, ri: 1 }, { si: 0, ri: 0 });
  assert.deepEqual(cells(up)[0], [["z"], ["x", "y"]]);
  const down = moveRow(base(), { si: 0, ri: 0 }, { si: 0, ri: 2 }); // to the end of its own section
  assert.deepEqual(cells(down)[0], [["z"], ["x", "y"]]);
});

test("move section", () => {
  assert.deepEqual(moveSection(base(), 1, 0).sections.map((s) => s.id), ["b", "a"]);
});

test("does not mutate input", () => {
  const l = base();
  placeField(l, "w", undefined, { si: 0, ri: 0, ci: 0 });
  moveRow(l, { si: 0, ri: 0 }, { si: 1, ri: 0 });
  assert.deepEqual(l, base());
});
