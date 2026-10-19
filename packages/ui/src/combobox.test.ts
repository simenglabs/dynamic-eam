// node --test packages/ui/src/combobox.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { filterOptions, moveIndex, toggleValue } from "./comboboxLogic.ts";

const opts = [
  { value: "LOW", label: "LOW — Low" },
  { value: "MEDIUM", label: "MEDIUM — Medium" },
  { value: "HIGH", label: "HIGH — High" },
  { value: "x", label: "Distribution Substation", hint: "GDD01001" },
];

test("search is case-insensitive, matches label, value, or hint", () => {
  assert.deepEqual(filterOptions(opts, "medium").map((o) => o.value), ["MEDIUM"]);
  assert.deepEqual(filterOptions(opts, "high").map((o) => o.value), ["HIGH"]);
  assert.deepEqual(filterOptions(opts, "gdd01").map((o) => o.value), ["x"]);
  assert.deepEqual(filterOptions(opts, "  ").length, 4);
});

test("all words must match (any order)", () => {
  assert.deepEqual(filterOptions(opts, "substation distribution").map((o) => o.value), ["x"]);
  assert.deepEqual(filterOptions(opts, "substation high").length, 0);
});

test("keyboard navigation: wraps & jumps to ends", () => {
  assert.equal(moveIndex(-1, 4, "ArrowDown"), 0);
  assert.equal(moveIndex(3, 4, "ArrowDown"), 0);
  assert.equal(moveIndex(0, 4, "ArrowUp"), 3);
  assert.equal(moveIndex(2, 4, "Home"), 0);
  assert.equal(moveIndex(1, 4, "End"), 3);
  assert.equal(moveIndex(1, 0, "ArrowDown"), -1, "no options");
});

test("multi: toggle add/remove without changing order", () => {
  assert.deepEqual(toggleValue(["a", "b"], "c"), ["a", "b", "c"]);
  assert.deepEqual(toggleValue(["a", "b", "c"], "b"), ["a", "c"]);
});
