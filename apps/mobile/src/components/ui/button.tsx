import React from "react";
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  type GestureResponderEvent,
  type StyleProp,
  type ViewStyle,
} from "react-native";
import { brandTokens } from "@/theme/tokens";

interface ButtonProps {
  onPress?: (event: GestureResponderEvent) => void;
  title: string;
  variant?: "primary" | "secondary" | "outline" | "ghost";
  loading?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  accessibilityLabel?: string;
  accessibilityHint?: string;
}

export function Button({
  onPress,
  title,
  variant = "primary",
  loading = false,
  disabled = false,
  style,
  accessibilityLabel,
  accessibilityHint,
}: ButtonProps) {
  const isInteractive = !disabled && !loading;

  return (
    <Pressable
      onPress={isInteractive ? onPress : undefined}
      disabled={!isInteractive}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel || title}
      accessibilityHint={accessibilityHint}
      accessibilityState={{
        disabled: !isInteractive,
        busy: loading,
      }}
      style={({ pressed }) => [
        styles.base,
        styles[variant],
        disabled && styles.disabled,
        pressed && isInteractive && styles[`${variant}Pressed`],
        style,
      ]}
    >
      {loading ? (
        <ActivityIndicator
          size="small"
          color={
            variant === "primary"
              ? brandTokens.colors.textOnGold
              : brandTokens.colors.brandGold500
          }
        />
      ) : (
        <Text
          style={[
            styles.text,
            styles[`${variant}Text`],
            disabled && styles.disabledText,
          ]}
        >
          {title}
        </Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    height: 48,
    borderRadius: brandTokens.borderRadius.md,
    justifyContent: "center",
    alignItems: "center",
    paddingHorizontal: brandTokens.spacing.md,
    flexDirection: "row",
  },
  text: {
    ...brandTokens.typography.button,
    textAlign: "center",
  },
  primary: {
    backgroundColor: brandTokens.colors.brandGold500,
  },
  primaryPressed: {
    backgroundColor: brandTokens.colors.brandGold700,
  },
  primaryText: {
    color: brandTokens.colors.textOnGold,
    fontWeight: "700",
  },
  secondary: {
    backgroundColor: brandTokens.colors.surface800,
  },
  secondaryPressed: {
    backgroundColor: brandTokens.colors.surface700,
  },
  secondaryText: {
    color: brandTokens.colors.textPrimary,
  },
  outline: {
    backgroundColor: "transparent",
    borderWidth: 1,
    borderColor: brandTokens.colors.brandGold500,
  },
  outlinePressed: {
    backgroundColor: brandTokens.colors.surface800,
  },
  outlineText: {
    color: brandTokens.colors.brandGold500,
  },
  ghost: {
    backgroundColor: "transparent",
  },
  ghostPressed: {
    backgroundColor: brandTokens.colors.surface800,
  },
  ghostText: {
    color: brandTokens.colors.brandGold300,
  },
  disabled: {
    opacity: 0.45,
  },
  disabledText: {
    color: brandTokens.colors.textMuted,
  },
});
