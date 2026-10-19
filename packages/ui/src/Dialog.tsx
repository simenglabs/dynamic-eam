"use client";

import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { AlertTriangle, Info } from "lucide-react";
import { createDialogStore, type DialogRequest } from "./dialogStore";
import { Button, Input, Textarea } from "./ui";

// Replacement for window.alert/confirm/prompt: native modal <dialog> (focus trapped, Esc = cancel).
// Mount <DialogHost/> once in the root layout (RootHtml already does).
const store = createDialogStore();

type Opts = Omit<DialogRequest, "kind">;
const opts = (o: Opts | string): Opts => (typeof o === "string" ? { title: o } : o);

export const alertDialog = (o: Opts | string) => store.open({ kind: "alert", ...opts(o) }) as Promise<void>;
export const confirmDialog = (o: Opts | string) => store.open({ kind: "confirm", ...opts(o) }) as Promise<boolean>;
export const promptDialog = (o: Opts | string) => store.open({ kind: "prompt", ...opts(o) }) as Promise<string | null>;

export function DialogHost() {
  const req = useSyncExternalStore(store.subscribe, store.current, () => undefined);
  const ref = useRef<HTMLDialogElement>(null);
  const [text, setText] = useState("");

  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (req) {
      setText(req.defaultValue ?? "");
      if (!d.open) d.showModal();
    } else if (d.open) {
      d.close();
    }
  }, [req]);

  const cancel = () => store.answer(req?.kind === "prompt" ? null : false);
  const ok = () => store.answer(req?.kind === "prompt" ? text : true);

  return (
    <dialog
      ref={ref}
      aria-labelledby="eam-dialog-title"
      onCancel={(e) => {
        e.preventDefault(); // Esc: close via the store so the Promise resolves
        cancel();
      }}
      onClick={(e) => e.target === ref.current && cancel()} // backdrop click
      className="m-auto w-[min(28rem,calc(100vw-2rem))] rounded-xl border bg-white p-0 text-foreground shadow-xl backdrop:bg-black/50"
    >
      {req && (
        <form
          method="dialog"
          onSubmit={(e) => {
            e.preventDefault();
            ok();
          }}
          className="flex flex-col gap-4 p-6"
        >
          <div className="flex items-start gap-3">
            <span className={"flex size-10 shrink-0 items-center justify-center rounded-full " + (req.danger ? "bg-destructive/10 text-destructive" : "bg-accent text-primary")}>
              {req.danger ? <AlertTriangle className="size-5" /> : <Info className="size-5" />}
            </span>
            <div className="flex flex-col gap-1 pt-1">
              <h2 id="eam-dialog-title" className="text-lg font-bold">{req.title}</h2>
              {req.message && <p className="text-muted-foreground whitespace-pre-line">{req.message}</p>}
            </div>
          </div>
          {req.kind === "prompt" &&
            (req.multiline ? (
              <Textarea autoFocus value={text} placeholder={req.placeholder} onChange={(e) => setText(e.target.value)} />
            ) : (
              <Input autoFocus value={text} placeholder={req.placeholder} onChange={(e) => setText(e.target.value)} />
            ))}
          <div className="flex justify-end gap-2">
            {req.kind !== "alert" && (
              <Button type="button" variant="outline" onClick={cancel}>{req.cancelText ?? "Cancel"}</Button>
            )}
            <Button autoFocus={req.kind !== "prompt"} variant={req.danger ? "danger" : "primary"}>
              {req.confirmText ?? (req.kind === "alert" ? "OK" : "Continue")}
            </Button>
          </div>
        </form>
      )}
    </dialog>
  );
}
