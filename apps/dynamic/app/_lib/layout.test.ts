// node --test apps/dynamic/app/_lib/layout.test.ts — same cases as the Go domain (layout_test.go).
import { test } from "node:test";
import assert from "node:assert/strict";
import { editable, effectiveLayout, type ContentType, type Field } from "./types.ts";

const f = (name: string): Field => ({ name, label: name, type: "text", required: false });
const fields = ["a", "b", "c", "d", "e"].map(f);

test("no layout: General section, 2 columns, field order", () => {
  const l = effectiveLayout({ fields, layout: null });
  assert.equal(l.sections.length, 1);
  assert.equal(l.sections[0].title, "General");
  assert.deepEqual(l.sections[0].rows.map((r) => r.fields), [["a", "b"], ["c", "d"], ["e"]]);
});

test("unplaced fields go to the last section, arranged order is kept", () => {
  const l = effectiveLayout({
    fields,
    layout: { sections: [{ id: "x", title: "X", rows: [{ columns: 3, fields: ["c", "", "a"] }] }, { id: "y", title: "Y", rows: [] }] },
  });
  assert.deepEqual(l.sections[0].rows[0].fields, ["c", "", "a"]);
  assert.deepEqual(l.sections[1].rows.flatMap((r) => r.fields), ["b", "d", "e"]);
});

test("references to deleted fields become empty slots", () => {
  const l = effectiveLayout({ fields: [f("a")], layout: { sections: [{ id: "x", title: "X", rows: [{ columns: 2, fields: ["a", "hilang"] }] }] } });
  assert.deepEqual(l.sections[0].rows[0].fields, ["a", ""]);
});

test("editable follows workflow state; unknown status stays editable", () => {
  const ct = {
    workflow: { states: [{ key: "draft", label: "D", color: "neutral", editable: true }, { key: "ok", label: "OK", color: "green", editable: false }], initial: "draft", transitions: [], views: [] },
  } as unknown as ContentType;
  assert.equal(editable(ct, "draft"), true);
  assert.equal(editable(ct, "ok"), false);
  assert.equal(editable(ct, "lama"), true);
  assert.equal(editable({ workflow: null } as unknown as ContentType, "apa"), true);
});

test("displayLabel: field name or {field} template", async () => {
  const { displayLabel } = await import("./types.ts");
  const data = { nomor: "PM-1", nama: "Kabel", jumlah: 10 };
  assert.equal(displayLabel("nomor", data), "PM-1");
  assert.equal(displayLabel("{nomor} - {nama}", data), "PM-1 - Kabel");
  assert.equal(displayLabel("#{nomor} ({hilang})", data), "#PM-1 ()");
  assert.equal(displayLabel("{jumlah} unit", data), "10 unit");
});
