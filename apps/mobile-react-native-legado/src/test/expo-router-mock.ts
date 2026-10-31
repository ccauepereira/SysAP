import { vi } from "vitest";

export const push = vi.fn();
export const replace = vi.fn();
export const back = vi.fn();

export const useRouter = vi.fn(() => ({
  push,
  replace,
  back,
}));

let mockSegments: string[] = [];

export function __setSegments(segments: string[]) {
  mockSegments = segments;
}

export const useSegments = vi.fn(() => mockSegments);

export const Stack = {
  Screen: ({ children }: any) => children || null,
};
