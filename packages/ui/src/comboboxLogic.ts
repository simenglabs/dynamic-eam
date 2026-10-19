// Logika murni Combobox (diuji di combobox.test.ts).
export type ComboOption = { value: string; label: string; hint?: string };

// Semua kata kueri harus muncul di label/value/hint (tidak peka huruf besar, urutan bebas).
export function filterOptions<T extends ComboOption>(options: T[], query: string): T[] {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (words.length === 0) return options;
  return options.filter((o) => {
    const hay = `${o.label} ${o.value} ${o.hint ?? ""}`.toLowerCase();
    return words.every((w) => hay.includes(w));
  });
}

// Indeks aktif berikutnya untuk tombol navigasi; berputar di ujung. -1 = tidak ada opsi.
export function moveIndex(current: number, count: number, key: "ArrowDown" | "ArrowUp" | "Home" | "End"): number {
  if (count === 0) return -1;
  switch (key) {
    case "ArrowDown":
      return current < 0 || current >= count - 1 ? 0 : current + 1;
    case "ArrowUp":
      return current <= 0 ? count - 1 : current - 1;
    case "Home":
      return 0;
    case "End":
      return count - 1;
  }
}

export const toggleValue = (values: string[], v: string) => (values.includes(v) ? values.filter((x) => x !== v) : [...values, v]);
