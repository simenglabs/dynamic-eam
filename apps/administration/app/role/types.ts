export const ACTIONS = ["read", "create", "update", "delete", "download"] as const;
export type Action = (typeof ACTIONS)[number];

export type Role = {
  id: string;
  name: string;
  description: string;
  system: boolean;
  permissions: Record<string, Action[]>;
};

export type PermTree = { zone: string; label: string; items: { key: string; label: string; transition: boolean }[] }[];
