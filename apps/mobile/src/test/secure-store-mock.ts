import { vi } from "vitest";

const store = new Map<string, string>();

export const AFTER_FIRST_UNLOCK = "AFTER_FIRST_UNLOCK";

export const setItemAsync = vi.fn(async (key: string, value: string) => {
  store.set(key, value);
});

export const getItemAsync = vi.fn(async (key: string) => {
  return store.get(key) || null;
});

export const deleteItemAsync = vi.fn(async (key: string) => {
  store.delete(key);
});

export function __clearStore() {
  store.clear();
}
