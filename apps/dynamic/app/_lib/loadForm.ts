import { api } from "@eam/ui";
import { domainCodes, relationTargets, transitionFields, type Component, type ContentType, type DomainValue, type Option } from "./types";

// Data pendukung EntryForm: component, opsi tiap target relation, dan value tiap domain (termasuk di dalam component).
export async function loadFormData(ct: ContentType) {
  const list = await api<Component[]>("dynamic", "/components");
  const components = Object.fromEntries(list.map((c) => [c.uid, c]));
  const all = [...ct.fields, ...transitionFields(ct)];
  const targets = [...relationTargets(all, components)];
  const opts = await Promise.all(
    targets.map((t) => api<Option[]>("dynamic", `/content-types/${encodeURIComponent(t)}/options`).catch(() => [] as Option[])),
  );
  const codes = [...domainCodes(all, components)];
  const vals = await Promise.all(
    codes.map((c) => api<DomainValue[]>("dynamic", `/domains/${encodeURIComponent(c)}/values`).catch(() => [] as DomainValue[])),
  );
  return {
    components,
    options: Object.fromEntries(targets.map((t, i) => [t, opts[i]])),
    domains: Object.fromEntries(codes.map((c, i) => [c, vals[i]])),
  };
}
