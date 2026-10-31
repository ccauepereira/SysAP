import React from "react";
import { Stack } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "@/contexts/auth-context";
import { brandTokens } from "@/theme/tokens";

export default function RootLayout() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <StatusBar style="light" backgroundColor={brandTokens.colors.surface950} />
        <Stack
          screenOptions={{
            headerShown: false,
            contentStyle: {
              backgroundColor: brandTokens.colors.surface950,
            },
            animation: "fade",
          }}
        >
          <Stack.Screen name="(auth)" options={{ headerShown: false }} />
          <Stack.Screen name="(owner)" options={{ headerShown: false }} />
          <Stack.Screen name="(athlete)" options={{ headerShown: false }} />
        </Stack>
      </AuthProvider>
    </SafeAreaProvider>
  );
}
