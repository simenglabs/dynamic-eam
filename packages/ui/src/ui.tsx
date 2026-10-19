import NextLink from "next/link";
import type { ComponentProps, ReactNode } from "react";

// ponytail: varian ditulis tangan, tanpa cva/shadcn CLI. Tambah shadcn kalau butuh komponen interaktif kompleks.
const cx = (...c: (string | false | undefined)[]) => c.filter(Boolean).join(" ");

const BUTTON = {
  primary: "bg-primary text-white hover:bg-primary-800 shadow-xs",
  outline: "bg-white border border-slate-200 text-slate-700 hover:bg-slate-50 shadow-xs",
  ghost: "text-slate-600 hover:bg-slate-100",
  danger: "bg-destructive text-white hover:bg-destructive/90 shadow-xs",
};

export function Button({ variant = "primary", className, ...p }: ComponentProps<"button"> & { variant?: keyof typeof BUTTON }) {
  return (
    <button
      className={cx(
        "inline-flex h-10 items-center justify-center gap-2 rounded-lg px-4 text-sm font-semibold whitespace-nowrap transition-colors disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4",
        BUTTON[variant],
        className,
      )}
      {...p}
    />
  );
}

// Badge pill dengan titik: warna + teks (status tidak pernah warna saja).
const BADGE = {
  green: "bg-green-50 text-green-700 [--dot:var(--green)]",
  red: "bg-red-50 text-red-700 [--dot:var(--red)]",
  yellow: "bg-amber-50 text-amber-700 [--dot:var(--yellow)]",
  blue: "bg-blue-50 text-blue-700 [--dot:var(--blue)]",
  neutral: "bg-slate-100 text-slate-600 [--dot:var(--neutral)]",
};

export function Badge({ color = "neutral", className, children, ...p }: ComponentProps<"span"> & { color?: keyof typeof BADGE }) {
  return (
    <span className={cx("inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium whitespace-nowrap", BADGE[color], className)} {...p}>
      <span aria-hidden className="size-1.5 rounded-full bg-[var(--dot)]" />
      {children}
    </span>
  );
}

export function Card({ className, ...p }: ComponentProps<"div">) {
  return <div className={cx("flex flex-col gap-4 rounded-2xl border border-slate-200/80 bg-white p-6 shadow-[0_1px_2px_rgba(15,23,42,0.04)]", className)} {...p} />;
}

const FIELD = "h-10 rounded-lg border border-input bg-white px-3 text-sm font-normal outline-none transition-shadow placeholder:text-slate-400 focus-visible:border-primary-500 focus-visible:ring-3 focus-visible:ring-primary-100 disabled:bg-slate-50 disabled:text-slate-500";

export function Input({ className, ...p }: ComponentProps<"input">) {
  return <input className={cx(FIELD, className)} {...p} />;
}


export function Textarea({ className, ...p }: ComponentProps<"textarea">) {
  return <textarea className={cx(FIELD, "h-auto min-h-24 py-2", className)} {...p} />;
}

export function Table({ className, ...p }: ComponentProps<"table">) {
  return (
    <div className="-mx-6 w-[calc(100%+3rem)] overflow-x-auto">
      <table className={cx("w-full text-sm", className)} {...p} />
    </div>
  );
}
export const Th = ({ className, ...p }: ComponentProps<"th">) => (
  <th className={cx("border-y border-slate-100 bg-slate-50 px-4 py-3 text-left text-xs font-semibold tracking-wide text-slate-500 whitespace-nowrap first:pl-6 last:pr-6", className)} {...p} />
);
export const Tr = ({ className, ...p }: ComponentProps<"tr">) => (
  <tr className={cx("border-b border-slate-100 transition-colors last:border-0 hover:bg-slate-50/70", className)} {...p} />
);
export const Td = ({ className, ...p }: ComponentProps<"td">) => (
  <td className={cx("max-w-64 truncate px-4 py-3.5 whitespace-nowrap first:pl-6 last:pr-6", className)} {...p} />
);

// Judul halaman: crumbs kecil di atas, judul besar, subtitle opsional, aksi di kanan.
export function PageHeader({ crumbs, title, subtitle, actions }: { crumbs: string[]; title: string; subtitle?: string; actions?: ReactNode }) {
  return (
    <header className="flex flex-wrap items-end justify-between gap-4 px-8 pt-6 pb-2">
      <div>
        <nav aria-label="breadcrumb" className="mb-1 text-xs text-slate-500">{crumbs.join("  /  ")}</nav>
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">{title}</h1>
        {subtitle && <p className="mt-1 text-slate-500">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </header>
  );
}

export function Field({ label, required, optional, children }: { label: string; required?: boolean; optional?: boolean; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="text-sm font-medium text-slate-700">
        {label}
        {required && <span className="text-destructive ml-0.5">*</span>}
        {optional && <span className="ml-1 font-normal text-slate-400">(opsional)</span>}
      </span>
      {children}
    </label>
  );
}

export function Alert({ className, ...p }: ComponentProps<"div">) {
  return <div role="alert" className={cx("rounded-xl border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-700", className)} {...p} />;
}

export function Pagination({ page, size, total, href }: { page: number; size: number; total: number; href: (page: number) => string }) {
  const last = Math.max(1, Math.ceil(total / size));
  const from = total === 0 ? 0 : (page - 1) * size + 1;
  const btn = "inline-flex h-9 min-w-9 items-center justify-center rounded-lg border px-3 text-sm font-medium";
  return (
    <div className="flex items-center justify-between text-sm text-slate-500">
      <span>Menampilkan {from}–{Math.min(page * size, total)} dari {total}</span>
      <nav aria-label="Pagination" className="flex gap-1.5">
        {page > 1 ? <NextLink href={href(page - 1)} className={cx(btn, "border-slate-200 bg-white text-slate-700 hover:bg-slate-50")}>Prev</NextLink> : <span className={cx(btn, "border-slate-100 text-slate-300")}>Prev</span>}
        <span className={cx(btn, "bg-primary border-primary text-white")}>{page}</span>
        {page < last ? <NextLink href={href(page + 1)} className={cx(btn, "border-slate-200 bg-white text-slate-700 hover:bg-slate-50")}>Next</NextLink> : <span className={cx(btn, "border-slate-100 text-slate-300")}>Next</span>}
      </nav>
    </div>
  );
}

// Bar aksi atas form (Back + tombol): menempel di atas saat scroll.
export function ActionBar({ children }: { children: ReactNode }) {
  return <div className="sticky top-16 z-10 mx-8 mt-4 flex items-center justify-between rounded-2xl border border-slate-200/80 bg-white/90 px-5 py-3 backdrop-blur">{children}</div>;
}
