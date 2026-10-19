"use client";

import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { Check, ChevronDown, Search, X } from "lucide-react";
import { filterOptions, moveIndex, toggleValue, type ComboOption } from "./comboboxLogic";

export type { ComboOption };

type Common = {
  options: ComboOption[];
  placeholder?: string;
  required?: boolean;
  disabled?: boolean;
  name?: string; // for plain <form>: value is submitted via a hidden input
  id?: string;
  className?: string;
  "aria-label"?: string;
  clearable?: boolean; // single: show a clear button
};
type Single = Common & { multiple?: false; value: string; onChange: (v: string) => void };
type Multi = Common & { multiple: true; value: string[]; onChange: (v: string[]) => void };

// Searchable dropdown (ARIA combobox + listbox pattern). Single or multiple.
export function Combobox(props: Single | Multi) {
  const { options, placeholder = "-- Select --", required, disabled, name, className, clearable = !props.required } = props;
  const selected = props.multiple ? props.value : props.value ? [props.value] : [];
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState("");
  const [active, setActive] = useState(-1);
  const root = useRef<HTMLDivElement>(null);
  const search = useRef<HTMLInputElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const listId = useId();
  const hits = useMemo(() => filterOptions(options, q), [options, q]);
  const labelOf = (v: string) => options.find((o) => o.value === v)?.label ?? v;

  useEffect(() => {
    if (!open) return;
    setActive(hits.findIndex((o) => selected.includes(o.value)));
    search.current?.focus();
    const away = (e: MouseEvent) => !root.current?.contains(e.target as Node) && close();
    document.addEventListener("mousedown", away);
    return () => document.removeEventListener("mousedown", away);
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => setActive((a) => Math.min(a, hits.length - 1)), [hits.length]);
  useEffect(() => {
    if (active >= 0) document.getElementById(`${listId}-${active}`)?.scrollIntoView({ block: "nearest" });
  }, [active, listId]);

  // refocus: return focus to the trigger (after picking / Esc) so keyboard users keep their place.
  const close = (refocus = false) => {
    setOpen(false);
    setQ("");
    if (refocus) trigger.current?.focus();
  };
  const pick = (v: string) => {
    if (props.multiple) props.onChange(toggleValue(props.value, v));
    else {
      props.onChange(v);
      close(true);
    }
  };
  const onKey = (e: KeyboardEvent) => {
    if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Home" || e.key === "End") {
      e.preventDefault();
      if (!open) return setOpen(true);
      setActive((a) => moveIndex(a, hits.length, e.key as "ArrowDown"));
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (!open) setOpen(true);
      else if (hits[active]) pick(hits[active].value);
    } else if (e.key === "Escape" && open) {
      e.preventDefault();
      e.stopPropagation(); // don't also close an outer modal
      close(true);
    } else if (e.key === "Tab") close();
  };

  const summary = props.multiple
    ? selected.length === 0 ? placeholder : selected.length <= 2 ? selected.map(labelOf).join(", ") : `${selected.length} selected`
    : selected[0] ? labelOf(selected[0]) : placeholder;

  return (
    <div ref={root} className={"relative " + (className ?? "")} onKeyDown={onKey}>
      {name && (props.multiple ? selected.map((v) => <input key={v} type="hidden" name={name} value={v} />) : <input type="hidden" name={name} value={selected[0] ?? ""} />)}
      {/* invisible input so native form `required` validation still works */}
      {required && <input tabIndex={-1} aria-hidden required value={selected.length ? "x" : ""} onChange={() => {}} className="pointer-events-none absolute bottom-0 left-4 h-px w-px opacity-0" />}
      <button
        ref={trigger}
        type="button"
        id={props.id}
        role="combobox"
        aria-label={props["aria-label"]}
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-controls={listId}
        disabled={disabled}
        onClick={() => (open ? close() : setOpen(true))}
        className="flex h-10 w-full items-center gap-2 rounded-lg border border-input bg-white pr-2 pl-3 text-left text-sm outline-none transition-shadow focus-visible:border-primary-500 focus-visible:ring-3 focus-visible:ring-primary-100 disabled:bg-slate-50 disabled:text-slate-500"
      >
        <span className={"flex-1 truncate " + (selected.length ? "text-slate-900" : "text-slate-400") + (clearable && !props.multiple && selected.length > 0 && !disabled ? " mr-6" : "")}>{summary}</span>
        <ChevronDown className={"size-4 text-slate-400 transition-transform " + (open ? "rotate-180" : "")} />
      </button>
      {/* Clear button: sibling of the trigger (not inside it) to avoid nested interactive elements. */}
      {clearable && !props.multiple && selected.length > 0 && !disabled && (
        <button
          type="button"
          aria-label="Clear selection"
          onClick={() => {
            props.onChange("");
            trigger.current?.focus();
          }}
          className="absolute top-1/2 right-8 -translate-y-1/2 rounded p-0.5 text-slate-400 hover:bg-slate-100 hover:text-slate-700"
        >
          <X className="size-3.5" />
        </button>
      )}

      {open && (
        <div className="absolute z-40 mt-1.5 w-full min-w-56 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-lg">
          <div className="relative border-b border-slate-100 p-2">
            <Search className="pointer-events-none absolute top-1/2 left-4.5 size-4 -translate-y-1/2 text-slate-400" />
            <input
              ref={search}
              value={q}
              onChange={(e) => {
                setQ(e.target.value);
                setActive(0);
              }}
              placeholder="Search…"
              aria-label="Search options"
              aria-controls={listId}
              aria-activedescendant={active >= 0 ? `${listId}-${active}` : undefined}
              className="h-9 w-full rounded-lg bg-slate-50 pr-3 pl-8 text-sm outline-none placeholder:text-slate-400 focus:bg-white focus:ring-2 focus:ring-primary-100"
            />
          </div>
          <ul id={listId} role="listbox" aria-multiselectable={props.multiple || undefined} className="max-h-64 overflow-y-auto p-1">
            {hits.map((o, i) => {
              const on = selected.includes(o.value);
              return (
                <li
                  key={o.value}
                  id={`${listId}-${i}`}
                  role="option"
                  aria-selected={on}
                  onMouseEnter={() => setActive(i)}
                  onMouseDown={(e) => e.preventDefault()} // keep focus in the search field
                  onClick={() => pick(o.value)}
                  className={"flex cursor-pointer items-center gap-2 rounded-lg px-2.5 py-2 text-sm " + (i === active ? "bg-slate-100" : "") + (on ? " font-medium text-primary-800" : " text-slate-700")}
                >
                  {props.multiple && (
                    <span className={"flex size-4 shrink-0 items-center justify-center rounded border " + (on ? "border-primary bg-primary text-white" : "border-slate-300")}>
                      {on && <Check className="size-3" strokeWidth={3} />}
                    </span>
                  )}
                  <span className="flex-1 truncate">{o.label}</span>
                  {o.hint && <span className="text-xs text-slate-400">{o.hint}</span>}
                  {!props.multiple && on && <Check className="size-4 text-primary-700" />}
                </li>
              );
            })}
            {hits.length === 0 && <li className="px-3 py-6 text-center text-sm text-slate-400">No results</li>}
          </ul>
        </div>
      )}
    </div>
  );
}
