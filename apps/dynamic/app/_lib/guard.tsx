import { notFound } from "next/navigation";
import { Alert, ApiError, PageHeader } from "@eam/ui";

// 403 -> access denied message, 404 -> not found, 502/503 (e.g. Elasticsearch down) -> service message,
// anything else is thrown to the error boundary.
export async function guard<T>(p: Promise<T>, crumbs: string[]): Promise<T | React.ReactElement> {
  try {
    return await p;
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) notFound();
    if (e instanceof ApiError && e.status === 403)
      return (
        <>
          <PageHeader crumbs={crumbs} title={crumbs.at(-1)!} />
          <div className="px-8 py-4"><Alert>You do not have access to this page.</Alert></div>
        </>
      );
    if (e instanceof ApiError && (e.status === 502 || e.status === 503))
      return (
        <>
          <PageHeader crumbs={crumbs} title={crumbs.at(-1)!} />
          <div className="px-8 py-4">
            <Alert className="border-amber-100 bg-amber-50 text-amber-800">
              The search service is currently unavailable, so the record list cannot be shown yet. Your data is still safely stored. ({e.message})
            </Alert>
          </div>
        </>
      );
    throw e;
  }
}

export const isElement = (x: unknown): x is React.ReactElement => typeof x === "object" && x !== null && "$$typeof" in x;
