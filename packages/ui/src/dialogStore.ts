// Antrean dialog (satu tampil, sisanya menunggu). Murni tanpa React; diuji di dialogStore.test.ts.
export type DialogRequest = {
  kind: "alert" | "confirm" | "prompt";
  title: string;
  message?: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean; // tombol konfirmasi merah (hapus, dsb)
  placeholder?: string; // prompt
  defaultValue?: string; // prompt
  multiline?: boolean; // prompt: textarea
};

type Pending = DialogRequest & { resolve: (v: unknown) => void };

export function createDialogStore() {
  const queue: Pending[] = [];
  const listeners = new Set<() => void>();
  const emit = () => listeners.forEach((l) => l());
  return {
    open(req: DialogRequest): Promise<unknown> {
      return new Promise((resolve) => {
        queue.push({ ...req, resolve });
        if (queue.length === 1) emit();
      });
    },
    current: (): DialogRequest | undefined => queue[0],
    // confirm: boolean, prompt: string | null, alert: diabaikan.
    answer(value: unknown) {
      const head = queue.shift();
      if (!head) return;
      head.resolve(value);
      emit();
    },
    subscribe(l: () => void) {
      listeners.add(l);
      return () => listeners.delete(l);
    },
  };
}
