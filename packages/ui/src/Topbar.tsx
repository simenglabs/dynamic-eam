"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { ChevronDown, LogOut, Search } from "lucide-react";
import type { ZoneDef } from "./menu";

type User = { name: string; username: string; roles: string[] } | null;

const initials = (name: string) =>
  name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]).join("").toUpperCase();

// Top bar: menu search (⌘K / Ctrl+K) and profile menu.
export function Topbar({ menu, user }: { menu: ZoneDef[]; user: User }) {
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        input.current?.focus();
      }
    };
    addEventListener("keydown", onKey);
    return () => removeEventListener("keydown", onKey);
  }, []);

  const all = useMemo(
    () =>
      menu.flatMap((z) =>
        z.items.flatMap((i) => ("items" in i ? i.items.map((l) => ({ ...l, label: `${i.label} › ${l.label}` })) : [i]))
          .map((l) => ({ label: l.label, zone: z.label, href: `/${l.zone ?? z.zone}${l.path}` })),
      ),
    [menu],
  );
  const hits = q.trim() ? all.filter((h) => `${h.label} ${h.zone}`.toLowerCase().includes(q.trim().toLowerCase())).slice(0, 8) : [];

  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-4 border-b border-slate-200/70 bg-[var(--background)]/85 px-8 backdrop-blur">
      <div className="relative w-full max-w-xl">
        <Search className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-slate-400" />
        <input
          ref={input}
          value={q}
          onChange={(e) => { setQ(e.target.value); setOpen(true); }}
          onFocus={() => setOpen(true)}
          onBlur={() => setTimeout(() => setOpen(false), 150)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && hits[0]) location.assign(hits[0].href);
            if (e.key === "Escape") e.currentTarget.blur();
          }}
          placeholder="Search menu…"
          aria-label="Search menu"
          className="h-10 w-full rounded-xl border border-slate-200 bg-white pr-14 pl-10 text-sm outline-none placeholder:text-slate-400 focus:border-primary-500 focus:ring-3 focus:ring-primary-100"
        />
        <kbd className="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 rounded-md border border-slate-200 bg-slate-50 px-1.5 text-[11px] text-slate-500">⌘ K</kbd>
        {open && hits.length > 0 && (
          <ul role="listbox" className="absolute top-12 right-0 left-0 z-30 overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg">
            {hits.map((h) => (
              <li key={h.href} role="option" aria-selected={false}>
                <a href={h.href} className="flex items-center justify-between px-4 py-2 text-sm hover:bg-slate-50">
                  <span>{h.label}</span>
                  <span className="text-xs text-slate-400">{h.zone}</span>
                </a>
              </li>
            ))}
          </ul>
        )}
      </div>

      <details className="relative ml-auto">
        <summary aria-label="Account menu" className="flex cursor-pointer list-none items-center gap-3 rounded-xl px-2 py-1.5 hover:bg-white">
          <span className="bg-primary-100 text-primary-800 flex size-9 items-center justify-center rounded-full text-sm font-bold">
            {initials(user?.name || user?.username || "?")}
          </span>
          <span className="hidden flex-col leading-tight sm:flex">
            <span className="text-sm font-semibold text-slate-900">{user?.name}</span>
            <span className="text-xs text-slate-500 capitalize">{user?.roles.join(", ")}</span>
          </span>
          <ChevronDown className="size-4 text-slate-400" />
        </summary>
        <div className="absolute top-12 right-0 z-30 w-56 rounded-xl border border-slate-200 bg-white p-2 shadow-lg">
          <p className="px-2 pt-1 text-sm font-semibold">{user?.name}</p>
          <p className="mb-2 px-2 text-xs text-slate-500">@{user?.username}</p>
          <form action="/logout" method="post">
            <button className="text-destructive flex w-full items-center gap-2 rounded-lg px-2 py-2 text-sm hover:bg-red-50">
              <LogOut className="size-4" /> Sign out
            </button>
          </form>
        </div>
      </details>
    </header>
  );
}
