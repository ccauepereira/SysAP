import React from "react";
import { Image, StyleSheet, Text, View } from "react-native";
import { brandTokens } from "@/theme/tokens";

interface BrandLogoProps {
  size?: number;
  showText?: boolean;
}

export function BrandLogo({ size = 72, showText = true }: BrandLogoProps) {
  return (
    <View style={styles.container}>
      <Image
        // Canonic brand asset
        source={require("../../../../assets/brand/artur-performance-logo.png")}
        style={{ width: size, height: size }}
        resizeMode="contain"
        accessibilityLabel="Artur Performance Logo"
      />
      {showText && (
        <View style={styles.textContainer}>
          <Text style={styles.brandTitle}>Artur Performance</Text>
          <Text style={styles.brandSubtitle}>SysAP</Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: "center",
    justifyContent: "center",
    marginVertical: brandTokens.spacing.md,
  },
  textContainer: {
    alignItems: "center",
    marginTop: brandTokens.spacing.sm,
  },
  brandTitle: {
    ...brandTokens.typography.title,
    color: brandTokens.colors.textPrimary,
    letterSpacing: 0.5,
  },
  brandSubtitle: {
    ...brandTokens.typography.caption,
    color: brandTokens.colors.brandGold300,
    fontWeight: "700",
    letterSpacing: 2,
    marginTop: 2,
    textTransform: "uppercase",
  },
});
