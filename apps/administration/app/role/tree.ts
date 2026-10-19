import { api, permTree, type DynamicMenu } from "@eam/ui";

// Baris matriks: menu statis + menu dinamis (content type) dari service dynamic.
export async function loadTree() {
  const cts = await api<DynamicMenu[]>("dynamic", "/content-types").catch(() => []);
  return permTree(cts);
}
