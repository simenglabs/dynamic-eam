// EAM logo: dark green square with an "asset node" symbol (center dot linked to 4 dots). Also used as the favicon (icon.svg).
export function LogoMark({ className = "size-9" }: { className?: string }) {
  return (
    <svg viewBox="0 0 40 40" className={className} aria-hidden>
      <rect width="40" height="40" rx="10" fill="var(--primary-700, #185c40)" />
      <g stroke="#fff" strokeWidth="2.6" strokeLinecap="round">
        <path d="M20 10v20M10 20h20" />
      </g>
      <g fill="#fff">
        <circle cx="20" cy="20" r="4.2" />
        <circle cx="20" cy="9.5" r="2.4" />
        <circle cx="20" cy="30.5" r="2.4" />
        <circle cx="9.5" cy="20" r="2.4" />
        <circle cx="30.5" cy="20" r="2.4" />
      </g>
      <circle cx="20" cy="20" r="1.8" fill="var(--primary-700, #185c40)" />
    </svg>
  );
}

export function Logo() {
  return (
    <span className="flex items-center gap-2.5">
      <LogoMark />
      <span className="flex flex-col leading-tight">
        <span className="text-lg font-bold tracking-tight text-slate-900">EAM</span>
        <span className="text-[11px] text-slate-500">Asset Management</span>
      </span>
    </span>
  );
}
