import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { BrandLogo } from "@/components/ui/brand-logo";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/contexts/auth-context";
import { brandTokens } from "@/theme/tokens";

export default function OwnerScreen() {
  const { logout, status } = useAuth();

  return (
    <SafeAreaView style={styles.safeArea}>
      <View style={styles.container}>
        <BrandLogo size={68} />

        <View style={styles.card}>
          <Text style={styles.greeting}>Olá, Owner</Text>
          <Text style={styles.message}>
            A área de cadastro de atletas será implementada na Fase 3.1.
          </Text>

          <Button
            title="Sair"
            variant="outline"
            onPress={logout}
            loading={status === "loading"}
            style={styles.logoutButton}
            accessibilityLabel="Sair"
            accessibilityHint="Encerra a sessão atual e retorna para o login"
          />
        </View>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: brandTokens.colors.surface950,
  },
  container: {
    flex: 1,
    justifyContent: "center",
    paddingHorizontal: brandTokens.spacing.md,
  },
  card: {
    backgroundColor: brandTokens.colors.surface900,
    borderRadius: brandTokens.borderRadius.lg,
    padding: brandTokens.spacing.lg,
    borderWidth: 1,
    borderColor: brandTokens.colors.borderDefault,
  },
  greeting: {
    ...brandTokens.typography.title,
    color: brandTokens.colors.textPrimary,
    marginBottom: brandTokens.spacing.sm,
  },
  message: {
    ...brandTokens.typography.body,
    color: brandTokens.colors.textSecondary,
    lineHeight: 22,
    marginBottom: brandTokens.spacing.lg,
  },
  logoutButton: {
    marginTop: brandTokens.spacing.sm,
  },
});
