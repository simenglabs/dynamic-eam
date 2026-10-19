import { api, permTree, type DynamicMenu } from "@eam/ui";

// Matrix rows: static menus + dynamic menus (content types) from the dynamic service.
export async function loadTree() {
  const cts = await api<DynamicMenu[]>("dynamic", "/content-types").catch(() => []);
  return permTree(cts);
}
