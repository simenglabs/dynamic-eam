import { MENU, PageHeader, api, can, getMe } from "@eam/ui";
import { EntryForm } from "../../../_lib/EntryForm";
import { guard, isElement } from "../../../_lib/guard";
import { loadFormData } from "../../../_lib/loadForm";
import { displayLabel, type ContentType, type Entry, type Revision } from "../../../_lib/types";

export default async function EditEntry({ params, searchParams }: {
  params: Promise<{ uid: string; id: string }>;
  searchParams: Promise<{ view?: string }>;
}) {
  const [{ uid, id }, { view }, me] = await Promise.all([params, searchParams, getMe()]);
  const qs = view ? `?view=${encodeURIComponent(view)}` : "";
  const res = await guard(
    Promise.all([
      api<ContentType>("dynamic", `/content-types/${encodeURIComponent(uid)}`),
      api<Entry>("dynamic", `/content-types/${encodeURIComponent(uid)}/entries/${encodeURIComponent(id)}${qs}`),
    ]),
    ["Dynamic", uid],
  );
  if (isElement(res)) return res;
  const [ct, entry] = res;
  const [{ components, options, domains }, history] = await Promise.all([
    loadFormData(ct),
    // Riwayat butuh izin read content type; pengguna lewat view saja tidak melihatnya.
    api<Revision[]>("dynamic", `/content-types/${encodeURIComponent(uid)}/entries/${encodeURIComponent(id)}/history`).catch(() => []),
  ]);
  const title = displayLabel(ct.displayField, entry) || id;
  // Tombol transisi: sah dari status saat ini DAN user punya izin "update" pada kunci transisi tsb.
  const transitions = (ct.workflow?.transitions ?? [])
    .filter((t) => t.from.includes(entry.workflowStatus ?? "") && can(me, `ct/${uid}/t/${t.key}`, "update"))
    .map((t) => ({ key: t.key, label: t.label }));
  const v = ct.workflow?.views.find((x) => x.key === view);
  return (
    <>
      <PageHeader crumbs={[MENU.find((m) => m.zone === (v?.parent ?? ct.parent))?.label ?? ct.parent, v?.label ?? ct.label, title]} title={title} />
      <EntryForm
        ct={ct}
        entry={entry}
        history={history}
        components={components}
        options={options}
        domains={domains}
        canSave={can(me, `ct/${uid}`, "update")}
        canDelete={can(me, `ct/${uid}`, "delete")}
        transitions={transitions}
        view={view}
        placement={{ userId: me?.id, userName: me?.name }}
      />
    </>
  );
}
