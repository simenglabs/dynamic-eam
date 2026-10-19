export type User = {
  id: string;
  username: string;
  displayName: string;
  email: string;
  roles: string[];
  isActive: boolean;
  isLocked: boolean;
  failedAttempts: number;
  createdAt: string;
  updatedAt: string;
};
