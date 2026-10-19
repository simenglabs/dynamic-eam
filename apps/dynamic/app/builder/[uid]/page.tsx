import { PageHeader, api, getMe } from "@eam/ui";
import { SchemaForm } from "../../_lib/SchemaForm";
import { guard, isElement } from "../../_lib/guard";
import { Forbidden } from "../../_lib/Forbidden";
import type { Component, ContentType, DomainCode } from "../../_lib/types";

export default async function EditContentType({ params }: { params: Promise<{ uid: string }> }) {
  const { uid } = await params;
  const crumbs = ["Dynamic", "Content-Type Builder", uid];
  if (!(await getMe())?.access.superAdmin) return <Forbidden crumbs={crumbs} />;
  const res = await guard(
    Promise.all([
      api<ContentType>("dynamic", `/content-types/${encodeURIComponent(uid)}`),
      api<ContentType[]>("dynamic", "/content-types"),
      api<Component[]>("dynamic", "/components"),
      api<DomainCode[]>("dynamic", "/domains").catch(() => []),
    ]),
    crumbs,
  );
  if (isElement(res)) return res;
  const [ct, cts, comps, domains] = res;
  return (
    <>
      <PageHeader crumbs={crumbs} title={ct.label} />
      <SchemaForm kind="content-types" initial={ct} contentTypes={cts} components={comps} domains={domains} />
    </>
  );
}
