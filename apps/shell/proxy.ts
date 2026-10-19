import { jwtVerify } from "jose";
import { NextResponse, type NextRequest } from "next/server";
const TOKEN_COOKIE = "eam_token"; // = TOKEN_COOKIE in @eam/ui & common.CookieName in Go

function secret() {
  const s = process.env.JWT_SECRET;
  if (!s && process.env.NODE_ENV === "production") throw new Error("JWT_SECRET must be set in production");
  return new TextEncoder().encode(s ?? "dev-secret-change-me"); // must match services/common
}

// Login gate for the shell + all zones (zones are reached through shell rewrites).
// Go services still verify the token themselves; this only redirects users without a session to /login.
export async function proxy(req: NextRequest) {
  const cookie = req.cookies.get(TOKEN_COOKIE)?.value;
  const token = cookie ?? req.headers.get("authorization")?.replace(/^Bearer /, ""); // Bearer for API clients
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
