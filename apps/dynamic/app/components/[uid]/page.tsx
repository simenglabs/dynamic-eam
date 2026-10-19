import { PageHeader, api, getMe } from "@eam/ui";
import { SchemaForm } from "../../_lib/SchemaForm";
import { Forbidden } from "../../_lib/Forbidden";
import { guard, isElement } from "../../_lib/guard";
import type { Component, ContentType, DomainCode } from "../../_lib/types";

export default async function EditComponent({ params }: { params: Promise<{ uid: string }> }) {
  const { uid } = await params;
  const crumbs = ["Dynamic", "Components", uid];
  if (!(await getMe())?.access.superAdmin) return <Forbidden crumbs={crumbs} />;
  const res = await guard(
    Promise.all([
      api<Component>("dynamic", `/components/${encodeURIComponent(uid)}`),
      api<ContentType[]>("dynamic", "/content-types"),
      api<Component[]>("dynamic", "/components"),
      api<DomainCode[]>("dynamic", "/domains").catch(() => []),
    ]),
    crumbs,
  );
  if (isElement(res)) return res;
  const [comp, cts, comps, domains] = res;
  return (
    <>
      <PageHeader crumbs={crumbs} title={comp.label} />
      <SchemaForm kind="components" initial={comp} contentTypes={cts} components={comps} domains={domains} />
    </>
  );
}
