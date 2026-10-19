import { PageHeader, api, getMe } from "@eam/ui";
import { SchemaForm } from "../../_lib/SchemaForm";
import type { Component, ContentType, DomainCode } from "../../_lib/types";
import { Forbidden } from "../../_lib/Forbidden";

export default async function NewContentType() {
  if (!(await getMe())?.access.superAdmin) return <Forbidden crumbs={["Dynamic", "Content-Type Builder"]} />;
  const [cts, comps, domains] = await Promise.all([
    api<ContentType[]>("dynamic", "/content-types"),
    api<Component[]>("dynamic", "/components"),
    api<DomainCode[]>("dynamic", "/domains").catch(() => []),
  ]);
  return (
    <>
      <PageHeader crumbs={["Dynamic", "Content-Type Builder", "New"]} title="New Content Type" />
      <SchemaForm kind="content-types" contentTypes={cts} components={comps} domains={domains} />
    </>
  );
}
