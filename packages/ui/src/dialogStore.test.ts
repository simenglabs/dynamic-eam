// node --test packages/ui/src/dialogStore.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { createDialogStore } from "./dialogStore.ts";

test("confirm: resolve true/false, antrean berurutan", async () => {
  const s = createDialogStore();
  const seen: (string | undefined)[] = [];
  s.subscribe(() => seen.push(s.current()?.title));
  const a = s.open({ kind: "confirm", title: "A" });
  const b = s.open({ kind: "confirm", title: "B" });
  assert.equal(s.current()?.title, "A", "dialog kedua menunggu yang pertama");
  s.answer(true);
  assert.equal(s.current()?.title, "B");
  s.answer(false);
  assert.equal(s.current(), undefined);
  assert.equal(await a, true);
  assert.equal(await b, false);
  assert.deepEqual(seen, ["A", "B", undefined]);
});

test("prompt: teks atau null saat batal", async () => {
  const s = createDialogStore();
  const ok = s.open({ kind: "prompt", title: "Catatan" });
  s.answer("mohon cek");
  assert.equal(await ok, "mohon cek");
  const cancel = s.open({ kind: "prompt", title: "Catatan" });
  s.answer(null);
  assert.equal(await cancel, null);
});

test("answer tanpa dialog aktif diabaikan", () => {
  const s = createDialogStore();
  assert.doesNotThrow(() => s.answer(true));
});
