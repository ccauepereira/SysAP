import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { brandTokens } from "@/theme/tokens";

export default function IndexScreen() {
  return (
    <View style={styles.container}>
      <ActivityIndicator size="large" color={brandTokens.colors.brandGold500} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: brandTokens.colors.surface950,
    justifyContent: "center",
    alignItems: "center",
  },
});
