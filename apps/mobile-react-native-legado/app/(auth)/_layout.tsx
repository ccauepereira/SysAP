import React from "react";
import { Stack } from "expo-router";
import { brandTokens } from "@/theme/tokens";

export default function AuthLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        contentStyle: {
          backgroundColor: brandTokens.colors.surface950,
        },
      }}
    >
      <Stack.Screen name="login" />
      <Stack.Screen name="ativar" />
      <Stack.Screen name="recuperar-senha" />
    </Stack>
  );
}
