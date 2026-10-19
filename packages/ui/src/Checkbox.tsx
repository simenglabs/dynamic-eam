"use client";

import { useEffect, useRef, type ComponentProps, type ReactNode } from "react";
import { Check, Minus } from "lucide-react";

// Checkbox custom di atas <input type="checkbox"> bawaan: keyboard, label, FormData, dan required tetap jalan.
export function Checkbox({ label, indeterminate, className, ...p }: Omit<ComponentProps<"input">, "type"> & { label?: ReactNode; indeterminate?: boolean }) {
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = !!indeterminate;
  }, [indeterminate]);
  return (
    <label className={"inline-flex cursor-pointer items-center gap-2 text-sm select-none has-[:disabled]:cursor-not-allowed has-[:disabled]:opacity-50 " + (className ?? "")}>
      <span className="relative inline-flex size-[1.1rem] shrink-0">
        <input ref={ref} type="checkbox" className="peer absolute inset-0 m-0 cursor-pointer appearance-none rounded-[5px] border border-slate-300 bg-white transition-colors checked:border-primary checked:bg-primary indeterminate:border-primary indeterminate:bg-primary focus-visible:ring-3 focus-visible:ring-primary-100 focus-visible:outline-none disabled:cursor-not-allowed" {...p} />
        <Check aria-hidden className="pointer-events-none absolute inset-0 m-auto hidden size-3 text-white peer-checked:block peer-indeterminate:hidden" strokeWidth={3.5} />
        <Minus aria-hidden className="pointer-events-none absolute inset-0 m-auto hidden size-3 text-white peer-indeterminate:block" strokeWidth={3.5} />
      </span>
      {label}
    </label>
  );
}
