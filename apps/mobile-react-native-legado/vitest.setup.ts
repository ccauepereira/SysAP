import { beforeEach, vi } from "vitest";

// Mock image asset imports for React Native Image require() calls in Node environment
const mockAssetLoader = (module: any) => {
  module.exports = "mock-image-asset";
};

if (typeof require !== "undefined" && require.extensions) {
  require.extensions[".png"] = mockAssetLoader;
  require.extensions[".jpg"] = mockAssetLoader;
  require.extensions[".jpeg"] = mockAssetLoader;
  require.extensions[".svg"] = mockAssetLoader;
}

beforeEach(() => {
  vi.restoreAllMocks();
});
