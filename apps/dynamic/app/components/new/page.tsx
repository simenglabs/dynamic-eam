import { PageHeader, api, getMe } from "@eam/ui";
import { SchemaForm } from "../../_lib/SchemaForm";
import { Forbidden } from "../../_lib/Forbidden";
import type { Component, ContentType, DomainCode } from "../../_lib/types";

export default async function NewComponent() {
  if (!(await getMe())?.access.superAdmin) return <Forbidden crumbs={["Dynamic", "Components"]} />;
  const [cts, comps, domains] = await Promise.all([
    api<ContentType[]>("dynamic", "/content-types"),
    api<Component[]>("dynamic", "/components"),
    api<DomainCode[]>("dynamic", "/domains").catch(() => []),
  ]);
  return (
    <>
      <PageHeader crumbs={["Dynamic", "Components", "New"]} title="New Component" />
      <SchemaForm kind="components" contentTypes={cts} components={comps} domains={domains} />
    </>
  );
}
