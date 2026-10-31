import React from "react";
import { Stack } from "expo-router";
import { brandTokens } from "@/theme/tokens";

export default function AthleteLayout() {
  return (
    <Stack
      screenOptions={{
        headerShown: false,
        contentStyle: {
          backgroundColor: brandTokens.colors.surface950,
        },
      }}
    >
      <Stack.Screen name="index" />
    </Stack>
  );
}
