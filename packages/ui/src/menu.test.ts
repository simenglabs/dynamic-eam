// node --test packages/ui/src/menu.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { isLinkActive } from "./menu.ts";

const at = (pathname: string, view = "") => ({ zone: "dynamic", pathname, view });
const tnd = { zone: "dynamic", path: "/c/asset-tnd" };
const approval = { zone: "dynamic", path: "/c/asset-tnd?view=approval" };
const report = { zone: "dynamic", path: "/c/asset-tnd?view=report" };

test("plain list: only the menu without a view is active", () => {
  assert.deepEqual([tnd, approval, report].map((l) => isLinkActive(l, at("/c/asset-tnd"))), [true, false, false]);
});

test("approval view: only that view's menu is active", () => {
  assert.deepEqual([tnd, approval, report].map((l) => isLinkActive(l, at("/c/asset-tnd", "approval"))), [false, true, false]);
});

test("detail & form keep their menu active (including via view)", () => {
  assert.equal(isLinkActive(tnd, at("/c/asset-tnd/ABC")), true);
  assert.equal(isLinkActive(approval, at("/c/asset-tnd/ABC", "approval")), true);
  assert.equal(isLinkActive(tnd, at("/c/asset-tnd/ABC", "approval")), false);
});

test("similar path is not active, other zone is not active", () => {
  assert.equal(isLinkActive({ zone: "dynamic", path: "/c/asset" }, at("/c/asset-tnd")), false);
  assert.equal(isLinkActive({ zone: "renkon", path: "/c/asset-tnd" }, at("/c/asset-tnd")), false);
});

test("hidden content type: not in sidebar, still in Role matrix", async () => {
  const { buildMenu, permTree } = await import("./menu.ts");
  const dyn = [{ uid: "user-profile", label: "User Profile", parent: "administration", hidden: true }];
  const admin = buildMenu(dyn, { superAdmin: true, permissions: {} }).find((z) => z.zone === "administration")!;
  assert.equal(admin.items.some((i) => "path" in i && i.path === "/c/user-profile"), false);
  const rows = permTree(dyn).find((z) => z.zone === "administration")!.items;
  assert.equal(rows.some((r) => r.key === "ct/user-profile"), true);
});
