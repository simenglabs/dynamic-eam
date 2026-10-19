// node --test apps/dynamic/app/_lib/history.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { describeChange, showValue, type Lookup } from "./history.ts";
import type { Field } from "./types.ts";

const valueFields: Field[] = [
  { name: "valueId", label: "Value ID", type: "text", required: true },
  { name: "valueDescription", label: "Value Description", type: "text", required: true },
  { name: "isActive", label: "Active", type: "boolean", required: false },
];
const lk: Lookup = {
  options: { "master-site": [{ id: "S1", label: "5127 - UP3 SURABAYA BARAT" }] },
  domains: { PRIORITY: [{ valueId: "HIGH", valueDescription: "High", isDefault: false }] },
  components: { "domain-value": { fields: valueFields } },
  attrs: { A1: "Pole height" },
};
const values: Field = { name: "values", label: "Values", type: "component", component: "domain-value", repeatable: true, uniqueBy: "valueId", required: false };

test("relation, domain, boolean, date shown as labels", () => {
  assert.equal(showValue({ name: "site", label: "Site", type: "relation", target: "master-site", required: false }, "S1", lk), "5127 - UP3 SURABAYA BARAT");
  assert.match(showValue({ name: "site", label: "Site", type: "relation", target: "master-site", required: false }, "XYZ123456", lk), /^\(record XYZ123…\)$/);
  assert.equal(showValue({ name: "p", label: "P", type: "domain", domain: "PRIORITY", required: false }, "HIGH", lk), "High");
  assert.equal(showValue({ name: "b", label: "B", type: "boolean", required: false }, false, lk), "No");
  assert.equal(showValue({ name: "d", label: "D", type: "date", required: false }, "2026-10-01", lk), "1 Oct 2026");
});

test("component list: added, changed, removed per item", () => {
  const before = [{ valueId: "LOW", valueDescription: "Low", isActive: true }, { valueId: "MID", valueDescription: "Medium" }];
  const after = [{ valueId: "LOW", valueDescription: "Lower", isActive: false }, { valueId: "HIGH", valueDescription: "High" }];
  assert.deepEqual(describeChange(values, before, after, lk, false).lines, [
    "LOW — Value Description: Low → Lower",
    "LOW — Active: Yes → No",
    "Added: Value ID: HIGH, Value Description: High",
    "Removed: MID",
  ]);
  assert.deepEqual(describeChange(values, null, [before[1]], lk, true).lines, ["Value ID: MID, Value Description: Medium"]);
});

test("attributes use attribute name", () => {
  const f: Field = { name: "specs", label: "Specifications", type: "attributes", from: "classification", required: false };
  assert.deepEqual(describeChange(f, { A1: 9 }, { A1: 12 }, lk, false).lines, ["Pole height: 9 → 12"]);
});
