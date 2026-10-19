"use client";

import NextLink from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { Bolt, ChevronDown, ClipboardList, Cog, Columns3Cog, House, Map, RectangleEllipsis, UserRound, type LucideIcon } from "lucide-react";
import { isLinkActive, type Link as MenuLink, type ZoneDef } from "./menu";
import { Logo } from "./Logo";

const ICONS: Record<string, LucideIcon> = {
  renkon: Columns3Cog,
  assets: Bolt,
  sigv2: Map,
  "work-order": ClipboardList,
  dynamic: RectangleEllipsis,
  master: Cog,
  administration: UserRound,
};

const ITEM = "flex h-10 items-center gap-3 rounded-xl px-3 text-[0.93rem] font-medium text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900";
const ACTIVE = "data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground data-[active=true]:font-semibold";
const SUB = "flex h-9 items-center rounded-lg px-3 text-sm text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 " + ACTIVE;

// zone = zone app yang sedang render (undefined = shell). Link dalam zone sama → NextLink (soft nav);
// beda zone → <a> (hard nav, beda Next.js app).
export function Sidebar({ zone, menu }: { zone?: string; menu: ZoneDef[] }) {
  const pathname = usePathname();
  const view = useSearchParams().get("view") ?? "";
  const isActive = (z: ZoneDef, l: MenuLink) => isLinkActive({ zone: l.zone ?? z.zone, path: l.path }, { zone, pathname, view });

  const SubLink = ({ z, l }: { z: ZoneDef; l: MenuLink }) => {
    const target = l.zone ?? z.zone;
    return target === zone ? (
      <NextLink href={l.path} data-active={isActive(z, l)} className={SUB}>{l.label}</NextLink>
    ) : (
      <a href={`/${target}${l.path}`} className={SUB}>{l.label}</a>
    );
  };

  return (
    <aside className="fixed inset-y-0 left-0 z-30 flex w-[var(--sidebar-width)] flex-col border-r border-slate-200/80 bg-white">
      <a href="/dashboard" aria-label="EAM — Dashboard" className="px-6 pt-6 pb-5">
        <Logo />
      </a>
      <nav aria-label="Menu utama" className="flex flex-1 flex-col gap-1 overflow-y-auto px-3 pb-6">
        <a href="/dashboard" data-active={!zone && pathname === "/dashboard"} className={`${ITEM} ${ACTIVE}`}>
          <House className="size-[1.15rem]" /> Dashboard
        </a>
        {menu.map((z) => {
          const Icon = ICONS[z.zone] ?? Cog;
          const here = z.items.some((i) => ("items" in i ? i.items.some((l) => isActive(z, l)) : isActive(z, i)));
          return (
            <details key={z.zone} open={here} className="group/zone">
              <summary data-active={here} className={`${ITEM} cursor-pointer list-none ${here ? "text-slate-900" : ""}`}>
                <Icon className="size-[1.15rem]" />
                <span className="flex-1">{z.label}</span>
                <ChevronDown className="size-4 text-slate-400 transition-transform group-open/zone:rotate-180" />
              </summary>
              <div className="mt-1 mb-2 ml-[1.35rem] flex flex-col gap-0.5 border-l border-slate-200 pl-3">
                {z.items.map((i) =>
                  "items" in i ? (
                    <details key={i.label} open={i.items.some((l) => isActive(z, l))} className="group/sub">
                      <summary className="flex h-9 cursor-pointer list-none items-center justify-between rounded-lg px-3 text-sm text-slate-500 hover:bg-slate-100">
                        {i.label}
                        <ChevronDown className="size-3.5 text-slate-400 transition-transform group-open/sub:rotate-180" />
                      </summary>
                      <div className="ml-3 flex flex-col gap-0.5 border-l border-slate-200 pl-2">
                        {i.items.map((l) => <SubLink key={l.path} z={z} l={l} />)}
                      </div>
                    </details>
                  ) : (
                    <SubLink key={i.path} z={z} l={i} />
                  ),
                )}
              </div>
            </details>
          );
        })}
      </nav>
    </aside>
  );
}
