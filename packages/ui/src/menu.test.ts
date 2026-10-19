// node --test packages/ui/src/menu.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { isLinkActive } from "./menu.ts";

const at = (pathname: string, view = "") => ({ zone: "dynamic", pathname, view });
const tnd = { zone: "dynamic", path: "/c/asset-tnd" };
const approval = { zone: "dynamic", path: "/c/asset-tnd?view=approval" };
const report = { zone: "dynamic", path: "/c/asset-tnd?view=report" };

test("list biasa: hanya menu tanpa view yang aktif", () => {
  assert.deepEqual([tnd, approval, report].map((l) => isLinkActive(l, at("/c/asset-tnd"))), [true, false, false]);
});

test("view approval: hanya menu view itu yang aktif", () => {
  assert.deepEqual([tnd, approval, report].map((l) => isLinkActive(l, at("/c/asset-tnd", "approval"))), [false, true, false]);
});

test("detail & form tetap mengaktifkan menunya (termasuk lewat view)", () => {
  assert.equal(isLinkActive(tnd, at("/c/asset-tnd/ABC")), true);
  assert.equal(isLinkActive(approval, at("/c/asset-tnd/ABC", "approval")), true);
  assert.equal(isLinkActive(tnd, at("/c/asset-tnd/ABC", "approval")), false);
});

test("path mirip tidak ikut aktif, zone lain tidak aktif", () => {
  assert.equal(isLinkActive({ zone: "dynamic", path: "/c/asset" }, at("/c/asset-tnd")), false);
  assert.equal(isLinkActive({ zone: "renkon", path: "/c/asset-tnd" }, at("/c/asset-tnd")), false);
});

test("content type hidden: tidak di sidebar, tetap di matriks Role", async () => {
  const { buildMenu, permTree } = await import("./menu.ts");
  const dyn = [{ uid: "user-profile", label: "Profil User", parent: "administration", hidden: true }];
  const admin = buildMenu(dyn, { superAdmin: true, permissions: {} }).find((z) => z.zone === "administration")!;
  assert.equal(admin.items.some((i) => "path" in i && i.path === "/c/user-profile"), false);
  const rows = permTree(dyn).find((z) => z.zone === "administration")!.items;
  assert.equal(rows.some((r) => r.key === "ct/user-profile"), true);
});
