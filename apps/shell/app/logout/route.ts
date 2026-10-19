import { NextResponse } from "next/server";
import { TOKEN_COOKIE } from "@eam/ui";

// POST saja (form di sidebar) supaya link/gambar dari luar tidak bisa me-logout user.
export async function POST(req: Request) {
  const res = NextResponse.redirect(new URL("/login", req.url), 303);
  res.cookies.delete(TOKEN_COOKIE);
  return res;
}
