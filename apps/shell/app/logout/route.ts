import { NextResponse } from "next/server";
import { TOKEN_COOKIE } from "@eam/ui";

// POST only (form in the top bar) so external links/images can't log the user out.
export async function POST(req: Request) {
  const res = NextResponse.redirect(new URL("/login", req.url), 303);
  res.cookies.delete(TOKEN_COOKIE);
  return res;
}
