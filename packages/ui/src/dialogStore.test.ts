// node --test packages/ui/src/dialogStore.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { createDialogStore } from "./dialogStore.ts";

test("confirm: resolves true/false, queued in order", async () => {
  const s = createDialogStore();
  const seen: (string | undefined)[] = [];
  s.subscribe(() => seen.push(s.current()?.title));
  const a = s.open({ kind: "confirm", title: "A" });
  const b = s.open({ kind: "confirm", title: "B" });
  assert.equal(s.current()?.title, "A", "second dialog waits for the first");
  s.answer(true);
  assert.equal(s.current()?.title, "B");
  s.answer(false);
  assert.equal(s.current(), undefined);
  assert.equal(await a, true);
  assert.equal(await b, false);
  assert.deepEqual(seen, ["A", "B", undefined]);
});

test("prompt: text, or null on cancel", async () => {
  const s = createDialogStore();
  const ok = s.open({ kind: "prompt", title: "Note" });
  s.answer("please check");
  assert.equal(await ok, "please check");
  const cancel = s.open({ kind: "prompt", title: "Note" });
  s.answer(null);
  assert.equal(await cancel, null);
});

test("answer without an active dialog is ignored", () => {
  const s = createDialogStore();
  assert.doesNotThrow(() => s.answer(true));
});
