"use client";

import Link from "next/link";
import { useState, useTransition } from "react";
import { ArrowLeft, ChevronDown, ChevronRight, Save, Search, Trash2 } from "lucide-react";
import { Alert, Badge, Button, Card, Field, Input } from "@eam/ui/ui";
import { confirmDialog } from "@eam/ui/dialog";
import { Checkbox } from "@eam/ui/checkbox";
import { deleteRole, saveRole } from "./actions";
import { ACTIONS, type Action, type PermTree, type Role } from "./types";

const slug = (s: string) => s.toLowerCase().trim().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 48);

export function RoleForm({ role, tree, canSave, canDelete }: { role?: Role; tree: PermTree; canSave: boolean; canDelete: boolean }) {
  const readOnly = !canSave || !!role?.system;
  const [name, setName] = useState(role?.name ?? "");
  const [id, setId] = useState(role?.id ?? "");
  const [idTouched, setIdTouched] = useState(!!role);
  const [description, setDescription] = useState(role?.description ?? "");
  const [perms, setPerms] = useState<Record<string, Action[]>>(role?.permissions ?? {});
  const [error, setError] = useState<string>();
  const [pending, start] = useTransition();
  const [q, setQ] = useState("");
  const [opened, setOpened] = useState<Set<string>>(new Set()); // expanded top-level menus
  const toggleZone = (z: string) => setOpened((s) => { const n = new Set(s); if (n.has(z)) n.delete(z); else n.add(z); return n; });
  const needle = q.trim().toLowerCase();
  const zones = tree
    .map((z) => ({ ...z, items: needle && !z.label.toLowerCase().includes(needle) ? z.items.filter((i) => `${i.label} ${i.key}`.toLowerCase().includes(needle)) : z.items }))
    .filter((z) => !needle || z.items.length > 0);

  const has = (key: string, a: Action) => !!perms[key]?.includes(a);
  const set = (keys: string[], a: Action, on: boolean) =>
    setPerms((p) => {
      const next = { ...p };
      for (const k of keys) {
        const cur = new Set(next[k] ?? []);
        if (on) cur.add(a);
        else cur.delete(a);
        // read is required when another action is checked; unchecking read revokes all. Transition keys (".../t/...") only need "update".
        if (on && a !== "read" && !k.includes("/t/")) cur.add("read");
        if (!on && a === "read") cur.clear();
        next[k] = ACTIONS.filter((x) => cur.has(x));
      }
      return next;
    });

  const submit = () =>
    start(async () => {
      const body = { id, name, description, permissions: perms };
      setError((await saveRole(role?.id, JSON.stringify(body)))?.error);
    });
  const remove = async () => {
    if (!role || !(await confirmDialog({ title: `Delete role ${role.name}?`, message: "A role still assigned to users can't be deleted.", confirmText: "Delete", danger: true }))) return;
    start(async () => setError((await deleteRole(role.id))?.error));
  };

  // all actions of one row (workflow transitions: update only)
  const rowActions = (key: string) => (key.includes("/t/") ? (["update"] as Action[]) : [...ACTIONS]);
  const allBox = (keys: string[], label: string) => {
    const pairs = keys.flatMap((k) => rowActions(k).map((a) => [k, a] as const));
    const n = pairs.filter(([k, a]) => has(k, a)).length;
    return (
      <Checkbox aria-label={label} disabled={readOnly} checked={n === pairs.length && n > 0} indeterminate={n > 0 && n < pairs.length}
        onChange={(e) => setPerms((p) => {
          const next = { ...p };
          for (const k of keys) next[k] = e.target.checked ? rowActions(k) : [];
          return next;
        })} />
    );
  };
  const box = (keys: string[], a: Action, label: string) => {
    const n = keys.filter((k) => has(k, a)).length;
    return (
      <Checkbox
        aria-label={label}
        disabled={readOnly}
        checked={n === keys.length}
        indeterminate={n > 0 && n < keys.length}
        onChange={(e) => set(keys, a, e.target.checked)}
      />
    );
  };

  return (
    <form action={submit}>
      <div className="sticky top-16 z-10 mx-8 mt-4 flex items-center justify-between rounded-2xl border border-slate-200/80 bg-white/90 px-5 py-3 backdrop-blur">
        <Link href="/role" className="flex items-center gap-2"><ArrowLeft className="size-5" /> Back</Link>
        {!readOnly && (
          <div className="flex gap-2">
            {role && canDelete && <Button type="button" variant="ghost" className="text-destructive" onClick={remove} disabled={pending}><Trash2 /> Delete</Button>}
            <Button disabled={pending}><Save /> {pending ? "Saving..." : "Save"}</Button>
          </div>
        )}
      </div>
      <div className="flex flex-col gap-4 px-8 py-4">
        {error && <Alert>{error}</Alert>}
        {role?.system && <Alert className="border-primary/30 bg-accent text-foreground">System role: full access, cannot be changed.</Alert>}
        <Card>
          <h2 className="border-b pb-4 text-lg font-bold">Role</h2>
          <div className="grid gap-x-4 gap-y-5 md:grid-cols-3">
            <Field label="Name" required>
              <Input value={name} required disabled={readOnly} onChange={(e) => { setName(e.target.value); if (!idTouched) setId(slug(e.target.value)); }} />
            </Field>
            <Field label="ID" required>
              {role ? (
                <p className="border-b px-3 py-2 font-normal">{role.id}</p>
              ) : (
                <Input value={id} required pattern="[a-z][a-z0-9-]{1,47}" title="lowercase letters, digits, '-'" onChange={(e) => { setId(e.target.value); setIdTouched(true); }} />
              )}
            </Field>
            <Field label="Description" optional>
              <Input value={description} disabled={readOnly} onChange={(e) => setDescription(e.target.value)} />
            </Field>
          </div>
        </Card>

        {!role?.system && (
          <Card>
            <div className="flex flex-wrap items-end gap-3 border-b pb-4">
              <div className="mr-auto">
                <h2 className="text-lg font-bold">Menu permissions</h2>
                <p className="text-muted-foreground text-sm">Click a top-level menu name to see its submenus. Checking a top-level menu row = all of its submenus. Actions other than read automatically add read.</p>
              </div>
              <label className="relative">
                <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-slate-400" />
                <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search menu…" aria-label="Search menu" className="w-56 pl-8" />
              </label>
              <Button type="button" variant="ghost" onClick={() => setOpened(new Set(tree.map((z) => z.zone)))}>Expand all</Button>
              <Button type="button" variant="ghost" onClick={() => setOpened(new Set())}>Collapse all</Button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-base">
                <thead>
                  <tr>
                    <th className="sticky top-0 z-10 bg-slate-200 p-2.5 text-left">Menu</th>
                    <th className="sticky top-0 z-10 w-20 bg-slate-200 p-2.5">All</th>
                    {ACTIONS.map((a) => <th key={a} className="sticky top-0 z-10 w-24 bg-slate-200 p-2.5 capitalize">{a}</th>)}
                  </tr>
                </thead>
                <tbody>
                  <tr className="border-b bg-amber-50">
                    <td className="p-2.5 font-bold">All menus <span className="text-muted-foreground font-normal">(including new ones)</span></td>
                    <td className="text-center">{allBox(["*"], "all menus all actions")}</td>
                    {ACTIONS.map((a) => <td key={a} className="text-center">{box(["*"], a, `all menus ${a}`)}</td>)}
                  </tr>
                  {zones.map((z) => {
                    const keys = z.items.filter((i) => !i.transition).map((i) => i.key);
                    const isOpen = !!needle || opened.has(z.zone);
                    const granted = z.items.filter((i) => (perms[i.key] ?? []).length > 0).length;
                    return [
                      <tr key={z.zone} className="border-b bg-slate-50">
                        <td className="p-2.5">
                          <button type="button" onClick={() => toggleZone(z.zone)} aria-expanded={isOpen} className="flex items-center gap-2 font-bold">
                            {isOpen ? <ChevronDown className="size-4 text-slate-400" /> : <ChevronRight className="size-4 text-slate-400" />}
                            {z.label}
                            <Badge color={granted ? "green" : "neutral"} className="font-normal">{granted}/{z.items.length} granted</Badge>
                          </button>
                        </td>
                        <td className="text-center">{allBox(z.items.map((i) => i.key), `${z.label} all actions`)}</td>
                        {ACTIONS.map((a) => <td key={a} className="text-center">{box(keys, a, `${z.label} ${a}`)}</td>)}
                      </tr>,
                      ...(isOpen ? z.items : []).map((i) => (
                        <tr key={i.key} className="border-b hover:bg-[#E5F5F8]">
                          <td className="p-2.5 pl-12">
                            {i.label} <span className="text-muted-foreground text-xs">{i.key}</span>
                          </td>
                          <td className="text-center">{allBox([i.key], `${i.label} all actions`)}</td>
                          {ACTIONS.map((a) => (
                            <td key={a} className="text-center">
                              {/* workflow transitions have a single permission: update (= allowed to run) */}
                              {!i.transition || a === "update" ? box([i.key], a, `${i.label} ${a}`) : null}
                            </td>
                          ))}
                        </tr>
                      )),
                    ];
                  })}
                </tbody>
              </table>
            </div>
          </Card>
        )}
      </div>
    </form>
  );
}
