"use client";

import { useActionState, useEffect, useState } from "react";
import { Eye, EyeOff, Lock, User } from "lucide-react";
import { Alert, Button, Input } from "@eam/ui/ui";
import { LogoMark } from "@eam/ui/logo";
import { login, type LoginState } from "./actions";

export function LoginForm({ next }: { next: string }) {
  const [state, action, pending] = useActionState<LoginState, FormData>(login, {});
  const [show, setShow] = useState(false);
  useEffect(() => {
    if (state.redirectTo) location.assign(state.redirectTo);
  }, [state.redirectTo]);

  return (
    <form action={action} className="flex w-full max-w-[420px] flex-col gap-5 rounded-2xl border border-slate-200/80 bg-white p-8 shadow-[0_1px_2px_rgba(15,23,42,0.04)]">
      <div className="flex flex-col items-center gap-3 text-center">
        <LogoMark className="size-12" />
        <div>
          <h1 className="text-xl font-bold tracking-tight text-slate-900">Sign in to EAM</h1>
          <p className="text-sm text-slate-500">Asset Management</p>
        </div>
      </div>
      <input type="hidden" name="next" value={next} />

      <label className="flex flex-col gap-1.5 text-sm font-medium text-slate-700">
        Email or username
        <span className="relative">
          <User className="text-muted-foreground absolute top-3 left-3 size-4" />
          <Input name="username" defaultValue={state.username} autoComplete="username" required autoFocus placeholder="name@company.com" className="w-full pl-10" />
        </span>
      </label>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="password" className="text-sm font-medium text-slate-700">Password</label>
        <span className="relative">
          <Lock className="text-muted-foreground absolute top-3 left-3 size-4" />
          <Input id="password" name="password" type={show ? "text" : "password"} autoComplete="current-password" required placeholder="password" className="w-full px-10" />
          <button type="button" onClick={() => setShow(!show)} aria-label={show ? "Hide password" : "Show password"} className="text-muted-foreground absolute top-3 right-3">
            {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
          </button>
        </span>
      </div>

      {state.error && (
        <Alert aria-live="assertive">
          {state.error}
          {state.attemptsRemaining !== undefined && state.attemptsRemaining > 0 && (
            <span className="block text-xs">Login attempts remaining: {state.attemptsRemaining}</span>
          )}
        </Alert>
      )}

      <Button disabled={pending || !!state.redirectTo} className="w-full">{pending || state.redirectTo ? "Processing..." : "Sign in"}</Button>
    </form>
  );
}
