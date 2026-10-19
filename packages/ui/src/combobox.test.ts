// node --test packages/ui/src/combobox.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { filterOptions, moveIndex, toggleValue } from "./comboboxLogic.ts";

const opts = [
  { value: "LOW", label: "LOW — Rendah" },
  { value: "MEDIUM", label: "MEDIUM — Sedang" },
  { value: "HIGH", label: "HIGH — Tinggi" },
  { value: "x", label: "Gardu Distribusi", hint: "GDD01001" },
];

test("cari tidak peka huruf besar, cocok di label, value, atau hint", () => {
  assert.deepEqual(filterOptions(opts, "sedang").map((o) => o.value), ["MEDIUM"]);
  assert.deepEqual(filterOptions(opts, "high").map((o) => o.value), ["HIGH"]);
  assert.deepEqual(filterOptions(opts, "gdd01").map((o) => o.value), ["x"]);
  assert.deepEqual(filterOptions(opts, "  ").length, 4);
});

test("semua kata harus cocok (urutan bebas)", () => {
  assert.deepEqual(filterOptions(opts, "distribusi gardu").map((o) => o.value), ["x"]);
  assert.deepEqual(filterOptions(opts, "gardu tinggi").length, 0);
});

test("navigasi keyboard: berputar & lompat ujung", () => {
  assert.equal(moveIndex(-1, 4, "ArrowDown"), 0);
  assert.equal(moveIndex(3, 4, "ArrowDown"), 0);
  assert.equal(moveIndex(0, 4, "ArrowUp"), 3);
  assert.equal(moveIndex(2, 4, "Home"), 0);
  assert.equal(moveIndex(1, 4, "End"), 3);
  assert.equal(moveIndex(1, 0, "ArrowDown"), -1, "tanpa opsi");
});

test("multi: toggle tambah/hapus tanpa mengubah urutan", () => {
  assert.deepEqual(toggleValue(["a", "b"], "c"), ["a", "b", "c"]);
  assert.deepEqual(toggleValue(["a", "b", "c"], "b"), ["a", "c"]);
});
