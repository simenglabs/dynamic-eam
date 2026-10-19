import { MENU, PageHeader, api, can, getMe } from "@eam/ui";
import { EntryForm } from "../../../_lib/EntryForm";
import { Forbidden } from "../../../_lib/Forbidden";
import { guard, isElement } from "../../../_lib/guard";
import { loadFormData } from "../../../_lib/loadForm";
import { type ContentType } from "../../../_lib/types";

export default async function NewEntry({ params }: { params: Promise<{ uid: string }> }) {
  const [{ uid }, me] = await Promise.all([params, getMe()]);
  const ct = await guard(api<ContentType>("dynamic", `/content-types/${encodeURIComponent(uid)}`), ["Dynamic", uid]);
  if (isElement(ct)) return ct;
  const crumbs = [MENU.find((m) => m.zone === ct.parent)?.label ?? ct.parent, ct.label, "New"];
  if (!can(me, `ct/${uid}`, "create")) return <Forbidden crumbs={crumbs} />;
  const [{ components, options, domains }, me2] = await Promise.all([
    loadFormData(ct),
    api<Record<string, unknown>>("dynamic", "/profiles/me").catch(() => ({}) as Record<string, unknown>),
  ]);
  const placement = {
    defaultSite: me2.defaultSite as string | undefined,
    organization: me2.organization as string | undefined,
    userId: me?.id,
    userName: me?.name,
    profile: me2,
  };
  return (
    <>
      <PageHeader crumbs={crumbs} title={`Add New ${ct.label}`} />
      <EntryForm ct={ct} components={components} options={options} domains={domains} placement={placement} canSave canDelete={false} transitions={[]} />
    </>
  );
}
