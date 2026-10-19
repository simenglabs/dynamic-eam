import { Alert, MENU, PageHeader, api, can, getMe } from "@eam/ui";
import { EntryForm } from "../../_lib/EntryForm";
import { guard, isElement } from "../../_lib/guard";
import { loadFormData } from "../../_lib/loadForm";
import { saveProfile } from "../../_lib/actions";
import type { ContentType, Entry, Revision } from "../../_lib/types";

const UID = "user-profile";

// User profile & placement (dynamic content type user-profile; fields managed in Content-Type Builder).
export default async function UserProfile({ params, searchParams }: {
  params: Promise<{ userId: string }>;
  searchParams: Promise<{ saved?: string }>;
}) {
  const [{ userId }, { saved }, me] = await Promise.all([params, searchParams, getMe()]);
  const crumbs = [MENU.find((m) => m.zone === "administration")?.label ?? "Administration", "Users Management", "Profile"];
  const res = await guard(
    Promise.all([
      api<ContentType>("dynamic", `/content-types/${UID}`),
      api<Entry>("dynamic", `/profiles/${encodeURIComponent(userId)}`),
    ]),
    crumbs,
  );
  if (isElement(res)) return res;
  const [ct, profile] = res;
  const exists = (profile.version ?? 0) > 0;
  const [{ components, options, domains }, history] = await Promise.all([
    loadFormData(ct),
    exists ? api<Revision[]>("dynamic", `/content-types/${UID}/entries/${encodeURIComponent(userId)}/history`).catch(() => []) : Promise.resolve([]),
  ]);
  return (
    <>
      <PageHeader crumbs={crumbs} title="Profile & Placement" subtitle={`User ${userId}`} />
      {saved && <div className="px-8 pt-2"><Alert className="border-green-100 bg-green-50 text-green-800">Profile saved.</Alert></div>}
      <EntryForm
        ct={ct}
        entry={exists ? profile : undefined}
        history={history}
        components={components}
        options={options}
        domains={domains}
        canSave={can(me, `ct/${UID}`, "update")}
        canDelete={false}
        transitions={[]}
        saveAction={saveProfile.bind(null, userId)}
        back={{ href: `/administration/user-management/${encodeURIComponent(userId)}`, label: "Back to user" }}
      />
    </>
  );
}
