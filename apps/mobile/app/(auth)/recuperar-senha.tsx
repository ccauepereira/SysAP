import React from "react";
import { StyleSheet, Text, View } from "react-native";
import { useRouter } from "expo-router";
import { SafeAreaView } from "react-native-safe-area-context";
import { BrandLogo } from "@/components/ui/brand-logo";
import { Button } from "@/components/ui/button";
import { brandTokens } from "@/theme/tokens";

export default function RecuperarSenhaScreen() {
  const router = useRouter();

  return (
    <SafeAreaView style={styles.safeArea}>
      <View style={styles.container}>
        <BrandLogo size={68} />

        <View style={styles.card}>
          <Text style={styles.title}>Recuperação de Senha</Text>
          <Text style={styles.description}>
            O fluxo de autoatendimento para recuperação de senha por desafio seguro
            estará disponível na Fase 3.
          </Text>

          <Button
            title="Voltar ao login"
            variant="outline"
            onPress={() => router.replace("/(auth)/login")}
            style={styles.backButton}
            accessibilityLabel="Voltar ao login"
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
  title: {
    ...brandTokens.typography.title,
    color: brandTokens.colors.textPrimary,
    marginBottom: brandTokens.spacing.sm,
  },
  description: {
    ...brandTokens.typography.body,
    color: brandTokens.colors.textSecondary,
    lineHeight: 22,
    marginBottom: brandTokens.spacing.lg,
  },
  backButton: {
    marginTop: brandTokens.spacing.sm,
  },
});
