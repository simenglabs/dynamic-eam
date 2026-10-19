import { jwtVerify } from "jose";
import { NextResponse, type NextRequest } from "next/server";
const TOKEN_COOKIE = "eam_token"; // = TOKEN_COOKIE di @eam/ui & common.CookieName di Go

function secret() {
  const s = process.env.JWT_SECRET;
  if (!s && process.env.NODE_ENV === "production") throw new Error("JWT_SECRET wajib di-set di production");
  return new TextEncoder().encode(s ?? "dev-secret-change-me"); // harus sama dengan services/common
}

// Gerbang login untuk shell + semua zone (zone diakses lewat rewrite shell).
// Service Go tetap memverifikasi token sendiri; ini hanya supaya user tanpa sesi diarahkan ke /login.
export async function proxy(req: NextRequest) {
  const cookie = req.cookies.get(TOKEN_COOKIE)?.value;
  const token = cookie ?? req.headers.get("authorization")?.replace(/^Bearer /, ""); // Bearer untuk klien API
  const valid = !!token && (await jwtVerify(token, secret(), { algorithms: ["HS256"] }).then(() => true, () => false));
  if (valid) return NextResponse.next();

  const res = req.nextUrl.pathname.startsWith("/api/")
    ? NextResponse.json({ error: "unauthorized" }, { status: 401 })
    : NextResponse.redirect(new URL(`/login?next=${encodeURIComponent(req.nextUrl.pathname + req.nextUrl.search)}`, req.url));
  if (cookie) res.cookies.delete(TOKEN_COOKIE);
  return res;
}

export const config = {
  matcher: ["/((?!login|logout|_next/static|_next/image|favicon.ico|api/administration/auth/login).*)"],
};
