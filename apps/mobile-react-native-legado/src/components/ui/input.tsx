import React, { useState } from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
  type KeyboardTypeOptions,
  type ReturnKeyTypeOptions,
  type StyleProp,
  type ViewStyle,
} from "react-native";
import { brandTokens } from "@/theme/tokens";

interface InputProps {
  label: string;
  value: string;
  onChangeText: (text: string) => void;
  placeholder?: string;
  secureTextEntry?: boolean;
  keyboardType?: KeyboardTypeOptions;
  returnKeyType?: ReturnKeyTypeOptions;
  onSubmitEditing?: () => void;
  autoCapitalize?: "none" | "sentences" | "words" | "characters";
  autoCorrect?: boolean;
  maxLength?: number;
  editable?: boolean;
  error?: string | null;
  helperText?: string;
  containerStyle?: StyleProp<ViewStyle>;
  accessibilityLabel?: string;
  accessibilityHint?: string;
  testID?: string;
}

export function Input({
  label,
  value,
  onChangeText,
  placeholder,
  secureTextEntry = false,
  keyboardType = "default",
  returnKeyType = "done",
  onSubmitEditing,
  autoCapitalize = "none",
  autoCorrect = false,
  maxLength,
  editable = true,
  error,
  helperText,
  containerStyle,
  accessibilityLabel,
  accessibilityHint,
  testID,
}: InputProps) {
  const [isFocused, setIsFocused] = useState(false);
  const [isPasswordVisible, setIsPasswordVisible] = useState(false);

  const showPasswordToggle = secureTextEntry;
  const isActualSecure = secureTextEntry && !isPasswordVisible;

  return (
    <View style={[styles.container, containerStyle]}>
      <Text style={styles.label}>{label}</Text>
      <View
        style={[
          styles.inputContainer,
          isFocused && styles.inputContainerFocused,
          Boolean(error) && styles.inputContainerError,
          !editable && styles.inputContainerDisabled,
        ]}
      >
        <TextInput
          value={value}
          onChangeText={onChangeText}
          placeholder={placeholder}
          placeholderTextColor={brandTokens.colors.textMuted}
          secureTextEntry={isActualSecure}
          keyboardType={keyboardType}
          returnKeyType={returnKeyType}
          onSubmitEditing={onSubmitEditing}
          autoCapitalize={autoCapitalize}
          autoCorrect={autoCorrect}
          maxLength={maxLength}
          editable={editable}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setIsFocused(false)}
          accessibilityLabel={accessibilityLabel || label}
          accessibilityHint={accessibilityHint}
          accessibilityRole="none"
          testID={testID}
          style={styles.input}
        />
        {showPasswordToggle && (
          <Pressable
            onPress={() => setIsPasswordVisible((prev) => !prev)}
            accessibilityRole="button"
            accessibilityLabel={
              isPasswordVisible ? "Ocultar senha" : "Ver senha"
            }
            accessibilityHint="Alterna a visibilidade da senha"
            style={styles.toggleButton}
          >
            <Text style={styles.toggleText}>
              {isPasswordVisible ? "Ocultar" : "Mostrar"}
            </Text>
          </Pressable>
        )}
      </View>
      {error ? (
        <Text style={styles.errorText} accessibilityRole="alert">
          {error}
        </Text>
      ) : helperText ? (
        <Text style={styles.helperText}>{helperText}</Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: brandTokens.spacing.md,
  },
  label: {
    ...brandTokens.typography.caption,
    color: brandTokens.colors.textSecondary,
    marginBottom: brandTokens.spacing.xs,
    fontWeight: "600",
  },
  inputContainer: {
    height: 48,
    backgroundColor: brandTokens.colors.surface800,
    borderRadius: brandTokens.borderRadius.md,
    borderWidth: 1,
    borderColor: brandTokens.colors.borderDefault,
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: brandTokens.spacing.md,
  },
  inputContainerFocused: {
    borderColor: brandTokens.colors.brandGold500,
  },
  inputContainerError: {
    borderColor: brandTokens.colors.borderError,
  },
  inputContainerDisabled: {
    opacity: 0.5,
  },
  input: {
    flex: 1,
    color: brandTokens.colors.textPrimary,
    ...brandTokens.typography.body,
    height: "100%",
  },
  toggleButton: {
    paddingVertical: brandTokens.spacing.xs,
    paddingHorizontal: brandTokens.spacing.sm,
    justifyContent: "center",
    alignItems: "center",
  },
  toggleText: {
    ...brandTokens.typography.caption,
    color: brandTokens.colors.brandGold300,
    fontWeight: "600",
  },
  errorText: {
    ...brandTokens.typography.caption,
    color: brandTokens.colors.statusError,
    marginTop: brandTokens.spacing.xs,
  },
  helperText: {
    ...brandTokens.typography.caption,
    color: brandTokens.colors.textMuted,
    marginTop: brandTokens.spacing.xs,
  },
});
