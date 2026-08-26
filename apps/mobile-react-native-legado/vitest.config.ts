import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

const directory = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  resolve: {
    alias: {
      "@": path.resolve(directory, "src"),
      "react-native": path.resolve(directory, "src/test/react-native-shim.ts"),
      "react-native-safe-area-context": path.resolve(directory, "src/test/react-native-shim.ts"),
      "expo-secure-store": path.resolve(directory, "src/test/secure-store-mock.ts"),
      "expo-router": path.resolve(directory, "src/test/expo-router-mock.ts"),
    },
  },
  test: {
    environment: "node",
    setupFiles: ["./vitest.setup.ts"],
    include: ["src/**/*.test.{ts,tsx}", "app/**/*.test.{ts,tsx}"],
    restoreMocks: true,
  },
});
